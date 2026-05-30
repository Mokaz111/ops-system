package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProxyGrafana 反向代理 Grafana 请求（通过 cookie 决定目标实例）。
// 路由: ANY /api/v1/grafana/proxy/*path
func (h *GrafanaInstanceHandler) ProxyGrafana(c *gin.Context) {
	instanceIDStr, err := c.Cookie("grafana_proxy_instance")
	if err != nil || instanceIDStr == "" {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "missing grafana_proxy_instance cookie; call sso first")
		return
	}
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid instance id in cookie")
		return
	}

	inst, err := h.svc.Get(c.Request.Context(), instanceID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if inst.URL == "" {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "grafana instance has no URL configured")
		return
	}

	targetURL, err := url.Parse(inst.URL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "invalid grafana URL")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 保留完整路径（含 /api/v1/grafana/proxy 前缀），Grafana serve_from_sub_path 会自行剥离。
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = c.Request.URL.Path
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = targetURL.Host
		req.Header.Set("X-WEBAUTH-USER", inst.AdminUser)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Frame-Options")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		response.JSON(c, gin.H{"error": "grafana proxy error", "detail": err.Error()})
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
