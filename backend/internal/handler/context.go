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
