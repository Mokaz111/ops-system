package middleware

import (
	"net/http"
	"strings"

	"ops-system/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "user_id"
const ContextUsernameKey = "username"
const ContextRoleKey = "role"

// JWTAuth 校验 Bearer Token。secret 为空时不拦截（开发模式）。
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.Next()
			return
		}
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			unauthorized(c)
			return
		}
		raw := strings.TrimPrefix(h, "Bearer ")
		claims, err := auth.ParseUserToken(secret, raw)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set(ContextUserIDKey, claims.Subject)
		c.Set(ContextUsernameKey, claims.Username)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    http.StatusUnauthorized,
		"message": "missing or invalid authorization header",
	})
}
