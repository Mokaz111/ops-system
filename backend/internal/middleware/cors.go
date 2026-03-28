package middleware

import "github.com/gin-gonic/gin"

// CORS 跨域中间件。allowedOrigins 为空时允许所有来源（开发模式）；
// 非空时仅放行白名单中的域名。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		var respOrigin string
		if len(allowed) == 0 {
			if origin != "" {
				respOrigin = origin
			} else {
				respOrigin = "*"
			}
		} else {
			if _, ok := allowed[origin]; ok {
				respOrigin = origin
			} else {
				if c.Request.Method == "OPTIONS" {
					c.AbortWithStatus(403)
					return
				}
				c.Next()
				return
			}
		}

		c.Header("Access-Control-Allow-Origin", respOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
