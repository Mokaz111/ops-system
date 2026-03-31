package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InstanceHandler 实例 HTTP。
type InstanceHandler struct {
	svc      *service.InstanceService
	scaleSvc *service.ScaleService
}

func NewInstanceHandler(svc *service.InstanceService, scaleSvc *service.ScaleService) *InstanceHandler {
	return &InstanceHandler{svc: svc, scaleSvc: scaleSvc}
}

type instanceResp struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	InstanceName string    `json:"instance_name"`
	InstanceType string    `json:"instance_type"`
	TemplateType string    `json:"template_type"`
	ReleaseName  string    `json:"release_name"`
	Namespace    string    `json:"namespace"`
	Spec         string    `json:"spec"`
	Status       string    `json:"status"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toInstanceResp(i *model.Instance) instanceResp {
	return instanceResp{
		ID:           i.ID,
		TenantID:     i.TenantID,
		InstanceName: i.InstanceName,
		InstanceType: i.InstanceType,
		TemplateType: i.TemplateType,
		ReleaseName:  i.ReleaseName,
		Namespace:    i.Namespace,
		Spec:         i.Spec,
		Status:       i.Status,
		URL:          i.URL,
		CreatedAt:    i.CreatedAt,
		UpdatedAt:    i.UpdatedAt,
	}
}

type createInstanceBody struct {
	TenantID     uuid.UUID `json:"tenant_id" binding:"required"`
	InstanceName string    `json:"instance_name" binding:"required"`
	InstanceType string    `json:"instance_type" binding:"required"`
	TemplateType string    `json:"template_type"`
	Spec         string    `json:"spec"`
}

// List GET /api/v1/instances
func (h *InstanceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if ps < 1 {
		ps = 20
	}
	var tenantID *uuid.UUID
	if s := c.Query("tenant_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid tenant_id")
			return
		}
		tenantID = &id
	}
	instanceType := c.Query("instance_type")
	status := c.Query("status")
	keyword := c.Query("keyword")

	list, total, err := h.svc.List(c.Request.Context(), page, ps, tenantID, instanceType, status, keyword)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	items := make([]instanceResp, 0, len(list))
	for i := range list {
		items = append(items, toInstanceResp(&list[i]))
	}
	response.JSON(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

// Create POST /api/v1/instances
func (h *InstanceHandler) Create(c *gin.Context) {
	var body createInstanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	inst, err := h.svc.Create(c.Request.Context(), &service.CreateInstanceRequest{
		TenantID:     body.TenantID,
		InstanceName: body.InstanceName,
		InstanceType: body.InstanceType,
		TemplateType: body.TemplateType,
		Spec:         body.Spec,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toInstanceResp(inst))
}

// Get GET /api/v1/instances/:id
func (h *InstanceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	inst, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toInstanceResp(inst))
}

type updateInstanceBody struct {
	InstanceName string `json:"instance_name"`
	Spec         string `json:"spec"`
	Status       string `json:"status"`
}

// Update PUT /api/v1/instances/:id
func (h *InstanceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body updateInstanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	inst, err := h.svc.Update(c.Request.Context(), id, &service.UpdateInstanceRequest{
		InstanceName: body.InstanceName,
		Spec:         body.Spec,
		Status:       body.Status,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, toInstanceResp(inst))
}

// Delete DELETE /api/v1/instances/:id
func (h *InstanceHandler) Delete(c *gin.Context) {
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

type scaleBody struct {
	ScaleType string `json:"scale_type" binding:"required"`
	Replicas  *int32 `json:"replicas"`
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
	Storage   string `json:"storage"`
}

// Scale POST /api/v1/instances/:id/scale
func (h *InstanceHandler) Scale(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body scaleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.scaleSvc.Scale(c.Request.Context(), id, &service.ScaleRequest{
		ScaleType: body.ScaleType,
		Replicas:  body.Replicas,
		CPU:       body.CPU,
		Memory:    body.Memory,
		Storage:   body.Storage,
	}); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// Metrics GET /api/v1/instances/:id/metrics
func (h *InstanceHandler) Metrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.svc.GetMetrics(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

func (h *InstanceHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrScaleInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTenantNotFoundForInstance):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInstanceNameRequired),
		errors.Is(err, service.ErrInvalidInstanceType):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidScaleType),
		errors.Is(err, service.ErrScaleNotSupported):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
