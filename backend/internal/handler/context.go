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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid "+key)
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
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "user service not configured")
		return nil, false
	}
	id, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	u, err := userSvc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	return u, true
}

// resolveTenantScope 解析列表/查询接口的租户作用域：
//   - admin：尊重 ?tenant_id=；未传则 nil（代表"全租户"）。
//   - 普通用户：必须有自己的 tenant_id；若 ?tenant_id= 与自身不符则 403。
//
// 返回值 (scope, ok)；ok=false 时已写入错误响应，上层直接 return。
func resolveTenantScope(c *gin.Context, userSvc *service.UserService) (*uuid.UUID, bool) {
	raw := c.Query("tenant_id")
	if isAdmin(c) {
		if raw == "" {
			return nil, true
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return nil, false
		}
		return &id, true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return nil, false
	}
	if u.TenantID == nil {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
		return nil, false
	}
	if raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return nil, false
		}
		if id != *u.TenantID {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return nil, false
		}
	}
	return u.TenantID, true
}

// assertTenantAccess 非 admin 用户必须命中 ownerTenant，否则写 403 并返回 false。
func assertTenantAccess(c *gin.Context, userSvc *service.UserService, ownerTenant uuid.UUID) bool {
	if isAdmin(c) {
		return true
	}
	u, ok := currentUser(c, userSvc)
	if !ok {
		return false
	}
	if u.TenantID == nil || *u.TenantID != ownerTenant {
		response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}
