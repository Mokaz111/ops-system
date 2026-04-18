package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GrafanaHostHandler Grafana 主机注册 HTTP。
type GrafanaHostHandler struct {
	svc *service.GrafanaHostService
}

func NewGrafanaHostHandler(svc *service.GrafanaHostService) *GrafanaHostHandler {
	return &GrafanaHostHandler{svc: svc}
}

// List GET /api/v1/grafana/hosts
func (h *GrafanaHostHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if raw := c.Query("tenant_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return
		}
		tenantID = &id
	}
	list, total, err := h.svc.List(c.Request.Context(), c.Query("scope"), tenantID, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/grafana/hosts/:id
func (h *GrafanaHostHandler) Get(c *gin.Context) {
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

// Create POST /api/v1/grafana/hosts (admin)
func (h *GrafanaHostHandler) Create(c *gin.Context) {
	var body service.CreateGrafanaHostRequest
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

// Update PUT /api/v1/grafana/hosts/:id (admin)
func (h *GrafanaHostHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.UpdateGrafanaHostRequest
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

// Delete DELETE /api/v1/grafana/hosts/:id (admin)
func (h *GrafanaHostHandler) Delete(c *gin.Context) {
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

func (h *GrafanaHostHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrGrafanaHostNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
