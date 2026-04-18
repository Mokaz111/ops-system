package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MetricHandler 指标库 HTTP。
type MetricHandler struct {
	svc *service.MetricService
}

func NewMetricHandler(svc *service.MetricService) *MetricHandler {
	return &MetricHandler{svc: svc}
}

// List GET /api/v1/metrics
func (h *MetricHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tplID *uuid.UUID
	if raw := c.Query("template_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid template_id")
			return
		}
		tplID = &id
	}
	list, total, err := h.svc.List(c.Request.Context(),
		c.Query("component"), tplID, c.Query("keyword"), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/metrics/:id
func (h *MetricHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Create POST /api/v1/metrics (admin)
func (h *MetricHandler) Create(c *gin.Context) {
	var body service.CreateMetricRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Update PUT /api/v1/metrics/:id (admin)
func (h *MetricHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.UpdateMetricRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Delete DELETE /api/v1/metrics/:id (admin)
func (h *MetricHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// Related GET /api/v1/metrics/:id/related
func (h *MetricHandler) Related(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	list, err := h.svc.Related(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, list)
}

// Reparse POST /api/v1/metrics/reparse/:templateId?version=xxx
func (h *MetricHandler) Reparse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid templateId")
		return
	}
	res, err := h.svc.Reparse(c.Request.Context(), id, c.Query("version"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, res)
}

func (h *MetricHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrMetricNotFound),
		errors.Is(err, service.ErrIntegrationTemplateNotFound),
		errors.Is(err, service.ErrIntegrationVersionNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrMetricNameExists):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
