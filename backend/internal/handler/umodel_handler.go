package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UModelHandler struct {
	svc     *service.UModelService
	userSvc *service.UserService
}

func NewUModelHandler(svc *service.UModelService, userSvc *service.UserService) *UModelHandler {
	return &UModelHandler{svc: svc, userSvc: userSvc}
}

func (h *UModelHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	if isAdmin(c) {
		raw := c.Query("workspace_id")
		if raw == "" {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "workspace_id required for admin")
			return uuid.Nil, false
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
			return uuid.Nil, false
		}
		return id, true
	}
	return resolveWorkspaceID(c, h.userSvc)
}

// ListEntities GET /api/v1/umodel/entities
func (h *UModelHandler) ListEntities(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	items, total, err := h.svc.ListEntities(c.Request.Context(), tenantID, c.Query("entity_type"), c.Query("keyword"), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": items, "total": total, "page": page, "page_size": ps})
}

// CreateEntity POST /api/v1/umodel/entities
func (h *UModelHandler) CreateEntity(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var body service.CreateEntityRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	e, err := h.svc.CreateEntity(c.Request.Context(), tenantID, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, e)
}

// DeleteEntity DELETE /api/v1/umodel/entities/:id
func (h *UModelHandler) DeleteEntity(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.DeleteEntity(c.Request.Context(), tenantID, id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"deleted": true})
}

// ListMetricSets GET /api/v1/umodel/metric-sets
func (h *UModelHandler) ListMetricSets(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	items, total, err := h.svc.ListMetricSets(c.Request.Context(), tenantID, c.Query("component"), c.Query("keyword"), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": items, "total": total, "page": page, "page_size": ps})
}

// CreateMetricSet POST /api/v1/umodel/metric-sets
func (h *UModelHandler) CreateMetricSet(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var body service.CreateMetricSetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	m, err := h.svc.CreateMetricSet(c.Request.Context(), tenantID, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// DeleteMetricSet DELETE /api/v1/umodel/metric-sets/:id
func (h *UModelHandler) DeleteMetricSet(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.DeleteMetricSet(c.Request.Context(), tenantID, id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"deleted": true})
}

// ListLogSets GET /api/v1/umodel/log-sets
func (h *UModelHandler) ListLogSets(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	items, total, err := h.svc.ListLogSets(c.Request.Context(), tenantID, c.Query("component"), c.Query("keyword"), page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": items, "total": total, "page": page, "page_size": ps})
}

// CreateLogSet POST /api/v1/umodel/log-sets
func (h *UModelHandler) CreateLogSet(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var body service.CreateLogSetRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	l, err := h.svc.CreateLogSet(c.Request.Context(), tenantID, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, l)
}

// DeleteLogSet DELETE /api/v1/umodel/log-sets/:id
func (h *UModelHandler) DeleteLogSet(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.DeleteLogSet(c.Request.Context(), tenantID, id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"deleted": true})
}

// ListDataLinks GET /api/v1/umodel/entities/:id/data-links
func (h *UModelHandler) ListDataLinks(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	entityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid entity id")
		return
	}
	items, err := h.svc.ListDataLinks(c.Request.Context(), tenantID, entityID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": items})
}

// CreateDataLink POST /api/v1/umodel/data-links
func (h *UModelHandler) CreateDataLink(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var body service.CreateDataLinkRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	d, err := h.svc.CreateDataLink(c.Request.Context(), tenantID, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, d)
}

// DeleteDataLink DELETE /api/v1/umodel/data-links/:id
func (h *UModelHandler) DeleteDataLink(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.DeleteDataLink(c.Request.Context(), tenantID, id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"deleted": true})
}

func (h *UModelHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEntityNotFound),
		errors.Is(err, service.ErrMetricSetNotFound),
		errors.Is(err, service.ErrLogSetNotFound),
		errors.Is(err, service.ErrDataLinkNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeNotFound, err.Error())
	case errors.Is(err, service.ErrUModelNameRequired):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, err.Error())
	}
}
