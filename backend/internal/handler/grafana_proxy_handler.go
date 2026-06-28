package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const grafanaProxyPathPrefix = "/api/v1/grafana/proxy"

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

	useSubPath := grafanaUsesSubPath(inst.Source)
	rewriteLocation := !useSubPath

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		reqPath := c.Request.URL.Path
		if useSubPath {
			req.URL.Path = reqPath
		} else {
			req.URL.Path = stripProxyPrefix(reqPath, grafanaProxyPathPrefix)
		}
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = clientPublicHost(c.Request)
		// 我们需要在外部 Grafana 模式下改写 HTML 资源路径，先禁用压缩以便安全改写响应体。
		req.Header.Del("Accept-Encoding")
		// 防止 304 直接返回无响应体，导致路径改写逻辑失效。
		req.Header.Del("If-None-Match")
		req.Header.Del("If-Modified-Since")

		applyGrafanaProxyAuth(req, inst)
		setGrafanaForwardedHeaders(req, c.Request)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Frame-Options")
		if loc := resp.Header.Get("Location"); loc != "" {
			resp.Header.Set("Location", rewriteGrafanaLocation(loc, targetURL, grafanaProxyPathPrefix, rewriteLocation))
		}
		if shouldRewriteGrafanaBody(resp.Header.Get("Content-Type")) && resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
			publicAppURL := buildPublicAppURL(c.Request, grafanaProxyPathPrefix)
			rewritten := rewriteGrafanaAssetPaths(body, grafanaProxyPathPrefix, publicAppURL)
			if shouldRewriteGrafanaFrontendSettings(resp) {
				rewritten = rewriteGrafanaFrontendSettingsJSON(rewritten, grafanaProxyPathPrefix, publicAppURL)
			}
			resp.Body = io.NopCloser(bytes.NewReader(rewritten))
			resp.ContentLength = int64(len(rewritten))
			resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
			// 避免浏览器缓存旧的根路径资源模板。
			resp.Header.Set("Cache-Control", "no-store")
		}
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		response.JSON(c, gin.H{"error": "grafana proxy error", "detail": err.Error()})
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// grafanaUsesSubPath 平台实例经 Helm 配置 serve_from_sub_path，需保留完整代理路径；
// 外部登记实例通常运行在根路径，代理需剥离前缀。
func grafanaUsesSubPath(source string) bool {
	return source == "platform"
}

// applyGrafanaProxyAuth 注入 Grafana 认证凭据。
func applyGrafanaProxyAuth(req *http.Request, inst *model.GrafanaInstance) {
	req.Header.Del("Authorization")

	user := inst.AdminUser
	if user == "" {
		user = "admin"
	}
	req.Header.Set("X-WEBAUTH-USER", user)

	if inst.AdminPassword != "" {
		req.SetBasicAuth(user, inst.AdminPassword)
	} else if inst.AdminTokenEnc != "" {
		req.Header.Set("Authorization", "Bearer "+inst.AdminTokenEnc)
	}
}

func setGrafanaForwardedHeaders(req *http.Request, clientReq *http.Request) {
	proto := clientReq.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if clientReq.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	req.Header.Set("X-Forwarded-Proto", proto)
	req.Header.Set("X-Forwarded-Host", clientPublicHost(clientReq))
	req.Header.Set("X-Forwarded-Prefix", grafanaProxyPathPrefix)
	if clientReq.RemoteAddr != "" {
		req.Header.Set("X-Real-IP", clientReq.RemoteAddr)
	}
}

// clientPublicHost 获取浏览器实际访问的 Host（开发环境下 Vite 代理后 Host 可能是后端地址）。
func clientPublicHost(clientReq *http.Request) string {
	if h := clientReq.Header.Get("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	if origin := clientReq.Header.Get("Origin"); origin != "" {
		if u, err := url.Parse(origin); err == nil && u.Host != "" {
			return u.Host
		}
	}
	if ref := clientReq.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return clientReq.Host
}

func shouldRewriteGrafanaBody(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/javascript") ||
		strings.Contains(ct, "text/javascript") ||
		strings.Contains(ct, "text/css")
}

func shouldRewriteGrafanaFrontendSettings(resp *http.Response) bool {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return false
	}
	return strings.Contains(resp.Request.URL.Path, "/api/frontend/settings")
}

func buildPublicAppURL(clientReq *http.Request, proxyPrefix string) string {
	proto := clientReq.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if clientReq.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := clientPublicHost(clientReq)
	return proto + "://" + host + strings.TrimSuffix(proxyPrefix, "/") + "/"
}

// rewriteGrafanaAssetPaths 把 Grafana 返回的根路径资源统一改写到代理前缀下。
func rewriteGrafanaAssetPaths(body []byte, proxyPrefix, publicAppURL string) []byte {
	s := string(body)
	const protectedAPIPrefix = "__OPS_GRAFANA_PROXY_API__"
	const protectedBaseHref = "__OPS_GRAFANA_BASE_HREF__"
	s = strings.ReplaceAll(s, `"`+proxyPrefix+`/api/`, `"`+protectedAPIPrefix)
	s = strings.ReplaceAll(s, `'`+proxyPrefix+`/api/`, `'`+protectedAPIPrefix)

	s = strings.ReplaceAll(s, `<base href="/" />`, `<base href="`+protectedBaseHref+`/" />`)
	s = strings.ReplaceAll(s, `<base href="/">`, `<base href="`+protectedBaseHref+`/">`)
	s = strings.ReplaceAll(s, `<base href="`+proxyPrefix+`/" />`, `<base href="`+protectedBaseHref+`/" />`)
	s = strings.ReplaceAll(s, `<base href="`+proxyPrefix+`/">`, `<base href="`+protectedBaseHref+`/">`)

	replacements := map[string]string{
		`"/public/`:     `"` + proxyPrefix + `/public/`,
		`'/public/`:     `'` + proxyPrefix + `/public/`,
		`"public/`:      `"` + proxyPrefix + `/public/`,
		`'public/`:      `'` + proxyPrefix + `/public/`,
		`url(/public/`:  `url(` + proxyPrefix + `/public/`,
		`url('/public/`: `url('` + proxyPrefix + `/public/`,
		`"/avatar/`:     `"` + proxyPrefix + `/avatar/`,
		`'/avatar/`:     `'` + proxyPrefix + `/avatar/`,
		`"/login`:       `"` + proxyPrefix + `/login`,
		`'/login`:       `'` + proxyPrefix + `/login`,
		`"/logout`:      `"` + proxyPrefix + `/logout`,
		`'/logout`:      `'` + proxyPrefix + `/logout`,
	}
	for old, newV := range replacements {
		s = strings.ReplaceAll(s, old, newV)
	}
	// 外部 Grafana 默认 appSubUrl 为空，会导致前端路由在代理前缀下 404。
	s = strings.ReplaceAll(s, `"appSubUrl":""`, `"appSubUrl":"`+strings.TrimSuffix(proxyPrefix, "/")+`"`)
	// 切换组织时 Grafana 前端会用 appUrl 跳转，需避免跳到 localhost:3000。
	s = strings.ReplaceAll(s, `"appUrl":"http://localhost:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"https://localhost:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"http://127.0.0.1:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"https://127.0.0.1:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	// 部分版本在 analytics.identifier 中仍保留 localhost，切组织时可能被前端误用。
	s = strings.ReplaceAll(s, "@http://localhost:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@https://localhost:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@http://127.0.0.1:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@https://127.0.0.1:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, `"`+protectedAPIPrefix, `"`+proxyPrefix+`/api/`)
	s = strings.ReplaceAll(s, `'`+protectedAPIPrefix, `'`+proxyPrefix+`/api/`)
	s = strings.ReplaceAll(s, protectedBaseHref, strings.TrimSuffix(proxyPrefix, "/"))
	double := strings.TrimSuffix(proxyPrefix, "/") + strings.TrimSuffix(proxyPrefix, "/")
	s = strings.ReplaceAll(s, double, strings.TrimSuffix(proxyPrefix, "/"))
	return []byte(s)
}

func rewriteGrafanaFrontendSettingsJSON(body []byte, proxyPrefix, publicAppURL string) []byte {
	s := string(body)
	prefix := strings.TrimSuffix(proxyPrefix, "/")
	s = strings.ReplaceAll(s, `"appSubUrl":""`, `"appSubUrl":"`+prefix+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"http://localhost:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"https://localhost:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"http://127.0.0.1:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, `"appUrl":"https://127.0.0.1:3000/"`, `"appUrl":"`+publicAppURL+`"`)
	s = strings.ReplaceAll(s, "@http://localhost:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@https://localhost:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@http://127.0.0.1:3000/", "@"+publicAppURL)
	s = strings.ReplaceAll(s, "@https://127.0.0.1:3000/", "@"+publicAppURL)
	return []byte(s)
}

// stripProxyPrefix 从请求路径中剥离 Grafana 代理前缀，返回 Grafana 实际接收的路径。
func stripProxyPrefix(reqPath, prefix string) string {
	if reqPath == prefix {
		return "/"
	}
	if strings.HasPrefix(reqPath, prefix+"/") {
		return strings.TrimPrefix(reqPath, prefix)
	}
	return reqPath
}

// rewriteGrafanaLocation 重写 Grafana 返回的 Location 头。
func rewriteGrafanaLocation(loc string, target *url.URL, proxyPrefix string, addProxyPrefix bool) string {
	if loc == "" {
		return loc
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		return loc
	}
	if parsed.IsAbs() {
		// 外部 Grafana 可能错误返回 localhost 绝对跳转，强制回写到代理路径。
		isLocalRedirect := strings.EqualFold(parsed.Host, "localhost:3000") || strings.EqualFold(parsed.Host, "127.0.0.1:3000")
		if parsed.Host != target.Host && !isLocalRedirect {
			return loc
		}
		loc = parsed.RequestURI()
		parsed, err = url.Parse(loc)
		if err != nil {
			return loc
		}
	}
	if !addProxyPrefix {
		return loc
	}
	locPath := parsed.Path
	if locPath == "" {
		locPath = "/"
	}
	if strings.HasPrefix(locPath, proxyPrefix) {
		return loc
	}
	rewritten := strings.TrimSuffix(proxyPrefix, "/") + locPath
	if parsed.RawQuery != "" {
		rewritten += "?" + parsed.RawQuery
	}
	return rewritten
}
