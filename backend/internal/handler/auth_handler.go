package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证 HTTP。
type AuthHandler struct {
	authSvc   *service.AuthService
	userSvc   *service.UserService
	jwtSecret string
}

func NewAuthHandler(authSvc *service.AuthService, userSvc *service.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc, jwtSecret: jwtSecret}
}

type loginBody struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	token, u, err := h.authSvc.Login(c.Request.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(c, gin.H{
		"token": token,
		"user":  toUserPublic(u),
	})
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	if h.jwtSecret == "" {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "configure jwt.secret or OPS_JWT_SECRET before using /auth/me")
		return
	}
	id, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.userSvc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(c, toUserPublic(u))
}
