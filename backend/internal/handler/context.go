package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/service"

	"ops-system/backend/internal/middleware"
	"ops-system/backend/pkg/response"
)

// setGrafanaProxyCookie 下发 Grafana 反向代理定位 cookie。
// Secure 属性按 X-Forwarded-Proto 动态设置（https 时为 true），
// HttpOnly=true、SameSite=Lax，防止明文 HTTP 下被截获冒充登录。
func setGrafanaProxyCookie(c *gin.Context, instanceID uuid.UUID) {
	secure := c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("grafana_proxy_instance", instanceID.String(), 86400, "/api/v1/grafana/proxy", "", secure, true)
}

const grafanaProxyPrefix = "/api/v1/grafana/proxy"

// buildGrafanaProxyURL 根据可选的 redirect 子路径拼接完整代理 URL。
// redirect 仅允许相对路径（以 / 开头），防止 open redirect。
func buildGrafanaProxyURL(redirect string) string {
	if redirect == "" || redirect[0] != '/' {
		return grafanaProxyPrefix + "/"
	}
	return grafanaProxyPrefix + redirect
}

func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	s := c.GetString(middleware.ContextUserIDKey)
	if s == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	return id, err == nil
}

func isAdmin(c *gin.Context) bool {
	return c.GetString(middleware.ContextRoleKey) == "admin"
}

func parsePositiveIntQuery(c *gin.Context, key, defaultValue string) (int, bool) {
	raw := c.DefaultQuery(key, defaultValue)
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid "+key)
		return 0, false
	}
	return v, true
}

func parsePageAndSize(c *gin.Context, defaultPageSize int) (int, int, bool) {
	page, ok := parsePositiveIntQuery(c, "page", "1")
	if !ok {
		return 0, 0, false
	}
	ps, ok := parsePositiveIntQuery(c, "page_size", strconv.Itoa(defaultPageSize))
	if !ok {
		return 0, 0, false
	}
	return page, ps, true
}

func currentUser(c *gin.Context, userSvc *service.UserService) (*model.User, bool) {
	if userSvc == nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "user service not configured")
		return nil, false
	}
	id, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return nil, false
	}
	u, err := userSvc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return nil, false
	}
	return u, true
}

// resolveWorkspaceScope 解析列表/查询接口的租户作用域：
//   - admin：尊重 ?tenant_id=；未传则 nil（代表"全租户"）。
//   - 普通用户：必须有自己的 tenant_id；若 ?tenant_id= 与自身不符则 403。
//
// 返回值 (scope, ok)；ok=false 时已写入错误响应，上层直接 return。
func resolveWorkspaceScope(c *gin.Context, userSvc *service.UserService) (*uuid.UUID, bool) {
	raw := c.Query("workspace_id")
	if isAdmin(c) {
		if raw == "" {
			return nil, true
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return nil, false
		}
		return &id, true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return nil, false
	}
	if u.WorkspaceID == nil {
		if raw == "" {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return nil, false
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return nil, false
		}
		allowed, err := userSvc.CanAccessWorkspace(c.Request.Context(), u.ID, id, "read")
		if err != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return nil, false
		}
		return &id, true
	}
	if raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return nil, false
		}
		if id != *u.WorkspaceID {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return nil, false
		}
	}
	return u.WorkspaceID, true
}

// assertWorkspaceAccess 非 admin 用户必须命中 ownerTenant，否则写 403 并返回 false。
func assertWorkspaceAccess(c *gin.Context, userSvc *service.UserService, ownerTenant uuid.UUID) bool {
	if isAdmin(c) {
		return true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return false
	}
	if u.WorkspaceID == nil || *u.WorkspaceID != ownerTenant {
		allowed, err := userSvc.CanAccessWorkspace(c.Request.Context(), u.ID, ownerTenant, "read")
		if err != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return false
		}
	}
	return true
}

func resolveWorkspaceID(c *gin.Context, userSvc *service.UserService) (uuid.UUID, bool) {
	u, ok := currentUser(c, userSvc)
	if !ok {
		return uuid.Nil, false
	}
	if u.WorkspaceID == nil {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "user has no tenant")
		return uuid.Nil, false
	}
	return *u.WorkspaceID, true
}
