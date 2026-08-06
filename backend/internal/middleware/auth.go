package middleware

import (
	"net/http"
	"strings"

	"ops-system/backend/internal/auth"
	"ops-system/backend/internal/service"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "user_id"
const ContextUsernameKey = "username"
const ContextRoleKey = "role"
const ContextAuthMethodKey = "auth_method"
const ContextAPITokenScopeKey = "api_token_scope"
const ContextAPITokenIDKey = "api_token_id"

// JWTAuth 校验 Bearer JWT 或 ops_ API Token。
func JWTAuth(secret string, tokenSvc *service.APITokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			unauthorized(c)
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if strings.HasPrefix(raw, "ops_") {
			authenticateAPIToken(c, tokenSvc, raw)
			return
		}
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "authentication is not configured",
			})
			return
		}
		claims, err := auth.ParseUserToken(secret, raw)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set(ContextUserIDKey, claims.Subject)
		c.Set(ContextUsernameKey, claims.Username)
		c.Set(ContextRoleKey, claims.Role)
		c.Set(ContextAuthMethodKey, "jwt")
		c.Next()
	}
}

func authenticateAPIToken(c *gin.Context, tokenSvc *service.APITokenService, raw string) {
	if tokenSvc == nil {
		unauthorized(c)
		return
	}
	u, token, err := tokenSvc.Authenticate(c.Request.Context(), raw)
	if err != nil || u == nil || token == nil {
		unauthorized(c)
		return
	}
	c.Set(ContextUserIDKey, u.ID.String())
	c.Set(ContextUsernameKey, u.Username)
	c.Set(ContextRoleKey, u.Role)
	c.Set(ContextAuthMethodKey, "api_token")
	c.Set(ContextAPITokenScopeKey, token.Scope)
	c.Set(ContextAPITokenIDKey, token.ID.String())
	c.Next()
}

// RequireRole 要求当前登录用户具备指定角色。
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(ContextRoleKey) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "forbidden",
			})
			return
		}
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    http.StatusUnauthorized,
		"message": "missing or invalid authorization header",
	})
}
