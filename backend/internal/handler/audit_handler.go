package handler

import (
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditHandler 审计日志 HTTP（admin）。
type AuditHandler struct {
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List GET /api/v1/audits
func (h *AuditHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	filter := service.AuditListFilter{
		Action:   c.Query("action"),
		Resource: c.Query("resource"),
		Page:     page,
		PageSize: ps,
	}
	if s := c.Query("actor_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid actor_id")
			return
		}
		filter.ActorID = &id
	}
	if s := c.Query("tenant_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid tenant_id")
			return
		}
		filter.TenantID = &id
	}
	list, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
		return
	}
	response.JSON(c, gin.H{
		"items":     list,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}
