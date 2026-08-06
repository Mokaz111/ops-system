package handler

import (
	"net/http"
	"strings"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// WebhookHandler 外部 webhook 回调。
type WebhookHandler struct {
	alertEvents *service.AlertEventService
	webhookToken string
}

func NewWebhookHandler(alertEvents *service.AlertEventService, webhookToken string) *WebhookHandler {
	return &WebhookHandler{alertEvents: alertEvents, webhookToken: webhookToken}
}

func (h *WebhookHandler) authorize(c *gin.Context) bool {
	if strings.TrimSpace(h.webhookToken) == "" {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "webhook token not configured")
		return false
	}
	token := strings.TrimSpace(c.GetHeader("X-Webhook-Token"))
	if token == "" {
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if token != h.webhookToken {
		response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "invalid webhook token")
		return false
	}
	return true
}

// Alertmanager POST /api/v1/webhooks/alertmanager
func (h *WebhookHandler) Alertmanager(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	var payload service.AlertmanagerWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid request body")
		return
	}
	if err := h.alertEvents.IngestAlertmanager(c.Request.Context(), &payload); err != nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
		return
	}
	response.JSON(c, gin.H{"status": "ok"})
}
