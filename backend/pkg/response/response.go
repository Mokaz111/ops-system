package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSON 统一成功响应。
func JSON(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// Error 业务/客户端错误（HTTP 4xx/5xx + 业务 code）。
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, gin.H{
		"code":    code,
		"message": message,
	})
}
