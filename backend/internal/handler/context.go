package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ops-system/backend/internal/middleware"
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
