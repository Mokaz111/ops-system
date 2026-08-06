package handler

import (
	"errors"
	"net/http"
	"time"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APITokenHandler API Token HTTP。
type APITokenHandler struct {
	svc *service.APITokenService
}

func NewAPITokenHandler(svc *service.APITokenService) *APITokenHandler {
	return &APITokenHandler{svc: svc}
}

type createAPITokenBody struct {
	Name      string     `json:"name" binding:"required"`
	Scope     string     `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type apiTokenPublic struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	Scope       string     `json:"scope"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// List GET /api/v1/api-tokens
func (h *APITokenHandler) List(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return
	}
	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	items := make([]apiTokenPublic, 0, len(list))
	for i := range list {
		items = append(items, apiTokenPublic{
			ID:          list[i].ID,
			UserID:      list[i].UserID,
			Name:        list[i].Name,
			TokenPrefix: list[i].TokenPrefix,
			Scope:       list[i].Scope,
			ExpiresAt:   list[i].ExpiresAt,
			LastUsedAt:  list[i].LastUsedAt,
			CreatedAt:   list[i].CreatedAt,
			UpdatedAt:   list[i].UpdatedAt,
		})
	}
	response.JSON(c, gin.H{"items": items})
}

// Create POST /api/v1/api-tokens
func (h *APITokenHandler) Create(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return
	}
	var body createAPITokenBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	result, err := h.svc.Create(c.Request.Context(), userID, &service.CreateAPITokenRequest{
		Name:      body.Name,
		Scope:     body.Scope,
		ExpiresAt: body.ExpiresAt,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{
		"token": apiTokenPublic{
			ID:          result.Token.ID,
			UserID:      result.Token.UserID,
			Name:        result.Token.Name,
			TokenPrefix: result.Token.TokenPrefix,
			Scope:       result.Token.Scope,
			ExpiresAt:   result.Token.ExpiresAt,
			LastUsedAt:  result.Token.LastUsedAt,
			CreatedAt:   result.Token.CreatedAt,
			UpdatedAt:   result.Token.UpdatedAt,
		},
		"plain_text": result.Plain,
	})
}

// Revoke DELETE /api/v1/api-tokens/:id
func (h *APITokenHandler) Revoke(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), userID, id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *APITokenHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAPITokenNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeNotFound, err.Error())
	case errors.Is(err, service.ErrAPITokenNameReq),
		errors.Is(err, service.ErrAPITokenScope):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
