package handler

import (
	"errors"
	"net/http"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InstanceHandler 实例 HTTP。
type InstanceHandler struct {
	svc                *service.InstanceService
	scaleSvc           *service.ScaleService
	userSvc            *service.UserService
	grafanaInstanceSvc *service.GrafanaInstanceService
}

func NewInstanceHandler(svc *service.InstanceService, scaleSvc *service.ScaleService, userSvc *service.UserService, grafanaInstanceSvc *service.GrafanaInstanceService) *InstanceHandler {
	return &InstanceHandler{svc: svc, scaleSvc: scaleSvc, userSvc: userSvc, grafanaInstanceSvc: grafanaInstanceSvc}
}

type instanceResp struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	ClusterID         *uuid.UUID `json:"cluster_id,omitempty"`
	InstanceName      string     `json:"instance_name"`
	InstanceType      string     `json:"instance_type"`
	TemplateType      string     `json:"template_type"`
	ReleaseName       string     `json:"release_name"`
	Namespace         string     `json:"namespace"`
	Spec              string     `json:"spec"`
	Status            string     `json:"status"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id,omitempty"`
	URL               string     `json:"url"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func toInstanceResp(i *model.Instance) instanceResp {
	return instanceResp{
		ID:                i.ID,
		TenantID:          i.TenantID,
		ClusterID:         i.ClusterID,
		InstanceName:      i.InstanceName,
		InstanceType:      i.InstanceType,
		TemplateType:      i.TemplateType,
		ReleaseName:       i.ReleaseName,
		Namespace:         i.Namespace,
		Spec:              i.Spec,
		Status:            i.Status,
		GrafanaInstanceID: i.GrafanaInstanceID,
		URL:               i.URL,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

type createInstanceBody struct {
	TenantID          *uuid.UUID `json:"tenant_id"`
	ClusterID         *uuid.UUID `json:"cluster_id"`
	InstanceName      string     `json:"instance_name" binding:"required"`
	InstanceType      string     `json:"instance_type" binding:"required"`
	TemplateType      string     `json:"template_type"`
	Spec              string     `json:"spec"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id"`
}

// List GET /api/v1/instances
func (h *InstanceHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	var tenantID *uuid.UUID
	if !isAdmin(c) {
		u, ok := currentUser(c, h.userSvc)
		if !ok {
			return
		}
		if u.TenantID == nil {
			if s := c.Query("tenant_id"); s != "" {
				id, err := uuid.Parse(s)
				if err != nil {
					response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid tenant_id")
					return
				}
				allowed, err := h.userSvc.CanAccessTenant(c.Request.Context(), u.ID, id, "read")
				if err != nil || !allowed {
					response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
					return
				}
				tenantID = &id
			} else {
				response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
				return
			}
		} else {
			tenantID = u.TenantID
		}
	} else if s := c.Query("tenant_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid tenant_id")
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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	inst, err := h.svc.Create(c.Request.Context(), &service.CreateInstanceRequest{
		TenantID:          body.TenantID,
		ClusterID:         body.ClusterID,
		InstanceName:      body.InstanceName,
		InstanceType:      body.InstanceType,
		TemplateType:      body.TemplateType,
		Spec:              body.Spec,
		GrafanaInstanceID: body.GrafanaInstanceID,
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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	inst, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, inst.TenantID) {
		return
	}
	response.JSON(c, toInstanceResp(inst))
}

type updateInstanceBody struct {
	InstanceName      string     `json:"instance_name"`
	Spec              string     `json:"spec"`
	Status            string     `json:"status"`
	TenantID          *uuid.UUID `json:"tenant_id"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id"`
}

// Update PUT /api/v1/instances/:id
func (h *InstanceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body updateInstanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	inst, err := h.svc.Update(c.Request.Context(), id, &service.UpdateInstanceRequest{
		InstanceName:      body.InstanceName,
		Spec:              body.Spec,
		Status:            body.Status,
		TenantID:          body.TenantID,
		GrafanaInstanceID: body.GrafanaInstanceID,
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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body scaleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	operator := ""
	if u, ok := currentUser(c, h.userSvc); ok {
		operator = u.Username
	}
	if err := h.scaleSvc.Scale(c.Request.Context(), id, &service.ScaleRequest{
		ScaleType: body.ScaleType,
		Replicas:  body.Replicas,
		CPU:       body.CPU,
		Memory:    body.Memory,
		Storage:   body.Storage,
		Operator:  operator,
	}); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListScaleEvents GET /api/v1/instances/:id/scale-events
func (h *InstanceHandler) ListScaleEvents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	// 先拉 instance 做租户校验，防止 IDOR：非 admin 只能看自己租户实例的伸缩历史。
	inst, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, inst.TenantID) {
		return
	}
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	f := repository.ScaleEventListFilter{InstanceID: &id}
	f.ScaleType = c.Query("scale_type")
	f.Status = c.Query("status")
	list, total, err := h.scaleSvc.ListScaleEvents(c.Request.Context(), f, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// Rebuild POST /api/v1/instances/:id/rebuild
func (h *InstanceHandler) Rebuild(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.Rebuild(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// Upgrade POST /api/v1/instances/:id/upgrade
func (h *InstanceHandler) Upgrade(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if err := h.svc.Upgrade(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// Metrics GET /api/v1/instances/:id/metrics
func (h *InstanceHandler) Metrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	inst, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, inst.TenantID) {
		return
	}

	m, err := h.svc.GetMetrics(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// LoginGrafana POST /api/v1/instances/:id/login
func (h *InstanceHandler) LoginGrafana(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	inst, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}

	// 解析目标 Grafana 实例 ID。
	var grafanaInstanceID uuid.UUID

	if inst.GrafanaInstanceID != nil {
		grafanaInstanceID = *inst.GrafanaInstanceID
	} else {
		grafanaInstances, _, err := h.grafanaInstanceSvc.List(c.Request.Context(), "platform", nil, 1, 1)
		if err != nil || len(grafanaInstances) == 0 {
			response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeNotFound, "no grafana instance configured")
			return
		}
		grafanaInstanceID = grafanaInstances[0].ID
	}

	// Set cookie so the reverse proxy knows which Grafana instance to target.
	c.SetCookie("grafana_proxy_instance", grafanaInstanceID.String(), 86400, "/api/v1/grafana/proxy", "", false, true)
	response.JSON(c, gin.H{
		"proxyUrl": "/api/v1/grafana/proxy/",
	})
}

func (h *InstanceHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeInstanceNotFound, err.Error())
	case errors.Is(err, service.ErrScaleInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeScaleInstanceNotFound, err.Error())
	case errors.Is(err, service.ErrTenantNotFoundForInstance):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeTenantNotFoundForInstance, err.Error())
	case errors.Is(err, service.ErrInstanceNameRequired),
		errors.Is(err, service.ErrInvalidInstanceType),
		errors.Is(err, service.ErrInvalidInstanceStatus),
		errors.Is(err, service.ErrInstanceNotReady):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	case errors.Is(err, service.ErrInstanceHasInstallations),
		errors.Is(err, service.ErrInstanceBusy):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeConflict, err.Error())
	case errors.Is(err, service.ErrInvalidScaleType),
		errors.Is(err, service.ErrScaleNotSupported),
		errors.Is(err, service.ErrScaleManagedByPlatform),
		errors.Is(err, service.ErrScaleTypeNotAllowed):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeInvalidPagination, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
