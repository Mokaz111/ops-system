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
func setGrafanaProxyCookie(c *gin.Context, instanceID uuid.UUID) {
	secure := c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("grafana_proxy_instance", instanceID.String(), 86400, "/api/v1/grafana/proxy", "", secure, true)
}

const grafanaProxyPrefix = "/api/v1/grafana/proxy"

// buildGrafanaProxyURL 根据可选的 redirect 子路径拼接完整代理 URL。
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

func userHasWorkspaceAccess(c *gin.Context, userSvc *service.UserService, userID, workspaceID uuid.UUID, action string) bool {
	if isAdmin(c) {
		return true
	}
	allowed, err := userSvc.CanAccessWorkspace(c.Request.Context(), userID, workspaceID, action)
	return err == nil && allowed
}

func parseWorkspaceIDQuery(c *gin.Context) (*uuid.UUID, bool) {
	raw := c.Query("workspace_id")
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

// resolveWorkspaceScope 解析列表/查询接口的租户作用域。
func resolveWorkspaceScope(c *gin.Context, userSvc *service.UserService) (*uuid.UUID, bool) {
	queryID, ok := parseWorkspaceIDQuery(c)
	if !ok {
		return nil, false
	}
	if isAdmin(c) {
		return queryID, true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return nil, false
	}
	if queryID == nil {
		ids, err := userSvc.ListUserWorkspaces(c.Request.Context(), u.ID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
			return nil, false
		}
		if len(ids) == 0 {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return nil, false
		}
		if len(ids) == 1 {
			id := ids[0]
			return &id, true
		}
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "workspace_id required")
		return nil, false
	}
	if !userHasWorkspaceAccess(c, userSvc, u.ID, *queryID, "read") {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
		return nil, false
	}
	return queryID, true
}

// assertWorkspaceAccess 非 admin 用户必须是工作空间成员。
func assertWorkspaceAccess(c *gin.Context, userSvc *service.UserService, ownerTenant uuid.UUID) bool {
	if isAdmin(c) {
		return true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return false
	}
	if !userHasWorkspaceAccess(c, userSvc, u.ID, ownerTenant, "read") {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
		return false
	}
	return true
}

func resolveWorkspaceID(c *gin.Context, userSvc *service.UserService) (uuid.UUID, bool) {
	u, ok := currentUser(c, userSvc)
	if !ok {
		return uuid.Nil, false
	}
	if queryID, ok := parseWorkspaceIDQuery(c); ok && queryID != nil {
		if !userHasWorkspaceAccess(c, userSvc, u.ID, *queryID, "read") {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return uuid.Nil, false
		}
		return *queryID, true
	}
	ids, err := userSvc.ListUserWorkspaces(c.Request.Context(), u.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
		return uuid.Nil, false
	}
	if len(ids) == 0 {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "user has no workspace membership")
		return uuid.Nil, false
	}
	if len(ids) > 1 {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "workspace_id required")
		return uuid.Nil, false
	}
	return ids[0], true
}
