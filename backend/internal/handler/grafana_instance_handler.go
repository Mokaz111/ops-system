package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GrafanaInstanceHandler Grafana 纳管实例注册 HTTP。
type GrafanaInstanceHandler struct {
	svc *service.GrafanaInstanceService
}

func NewGrafanaInstanceHandler(svc *service.GrafanaInstanceService) *GrafanaInstanceHandler {
	return &GrafanaInstanceHandler{svc: svc}
}

// List GET /api/v1/grafana/instances
func (h *GrafanaInstanceHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var zoneID *uuid.UUID
	if raw := c.Query("zone_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid zone_id")
			return
		}
		zoneID = &id
	}
	list, total, err := h.svc.List(c.Request.Context(), c.Query("source"), zoneID, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Get GET /api/v1/grafana/instances/:instanceId
func (h *GrafanaInstanceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Create POST /api/v1/grafana/instances (admin)
func (h *GrafanaInstanceHandler) Create(c *gin.Context) {
	var body service.CreateGrafanaInstanceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Update PUT /api/v1/grafana/instances/:instanceId (admin)
func (h *GrafanaInstanceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body service.UpdateGrafanaInstanceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// Delete DELETE /api/v1/grafana/instances/:instanceId (admin)
func (h *GrafanaInstanceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// Login POST /api/v1/grafana/instances/:instanceId/login
// 支持 ?redirect=/d/uid/title 指定登录后跳转的 Grafana 子路径。
func (h *GrafanaInstanceHandler) Login(c *gin.Context) {
	id, err := uuid.Parse(c.Param("instanceId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	m, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if m.URL == "" {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "grafana url not configured")
		return
	}
	setGrafanaProxyCookie(c, id)
	response.JSON(c, gin.H{
		"proxyUrl": buildGrafanaProxyURL(c.Query("redirect")),
	})
}

func (h *GrafanaInstanceHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrGrafanaInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeGrafanaInstanceNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeInvalidPagination, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
