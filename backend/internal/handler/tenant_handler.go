package handler

import (
	"errors"
	"net/http"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantHandler 租户 HTTP。
type TenantHandler struct {
	svc     *service.TenantService
	userSvc *service.UserService
}

func NewTenantHandler(svc *service.TenantService, userSvc *service.UserService) *TenantHandler {
	return &TenantHandler{svc: svc, userSvc: userSvc}
}

type tenantResp struct {
	ID           uuid.UUID `json:"id"`
	TenantName   string    `json:"tenant_name"`
	DeptID       uuid.UUID `json:"dept_id"`
	VMUserID     string    `json:"vmuser_id"`
	VMUserKey    string    `json:"vmuser_key,omitempty"`
	TemplateType string    `json:"template_type"`
	QuotaConfig  string    `json:"quota_config"`
	Status       string    `json:"status"`
	N9ETeamID    int64     `json:"n9e_team_id"`
	GrafanaOrgID int64     `json:"grafana_org_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	InsertURL    string    `json:"insert_url,omitempty"`
}

func (h *TenantHandler) toTenantResp(t *model.Tenant, withKey bool) tenantResp {
	r := tenantResp{
		ID:           t.ID,
		TenantName:   t.TenantName,
		DeptID:       t.DeptID,
		VMUserID:     t.VMUserID,
		TemplateType: t.TemplateType,
		QuotaConfig:  t.QuotaConfig,
		Status:       t.Status,
		N9ETeamID:    t.N9ETeamID,
		GrafanaOrgID: t.GrafanaOrgID,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		InsertURL:    h.svc.InsertURL(t.VMUserID),
	}
	if withKey {
		r.VMUserKey = t.VMUserKey
	}
	return r
}

type createTenantBody struct {
	TenantName   string    `json:"tenant_name" binding:"required"`
	DeptID       uuid.UUID `json:"dept_id" binding:"required"`
	TemplateType string    `json:"template_type" binding:"required"`
	QuotaConfig  string    `json:"quota_config"`
}

// List GET /api/v1/tenants
func (h *TenantHandler) List(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}

	if !isAdmin(c) {
		u, ok := currentUser(c, h.userSvc)
		if !ok {
			return
		}
		if u.TenantID == nil {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return
		}
		t, err := h.svc.Get(c.Request.Context(), *u.TenantID)
		if err != nil {
			h.handleErr(c, err)
			return
		}
		response.JSON(c, gin.H{
			"items":     []tenantResp{h.toTenantResp(t, false)},
			"total":     1,
			"page":      page,
			"page_size": ps,
		})
		return
	}

	var deptID *uuid.UUID
	if s := c.Query("dept_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid dept_id")
			return
		}
		deptID = &id
	}
	templateType := c.Query("template_type")
	status := c.Query("status")
	keyword := c.Query("keyword")

	list, total, err := h.svc.List(c.Request.Context(), page, ps, deptID, templateType, status, keyword)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	items := make([]tenantResp, 0, len(list))
	for i := range list {
		items = append(items, h.toTenantResp(&list[i], false))
	}
	response.JSON(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": ps,
	})
}

// Create POST /api/v1/tenants
func (h *TenantHandler) Create(c *gin.Context) {
	var body createTenantBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.svc.Create(c.Request.Context(), &service.CreateTenantRequest{
		TenantName:   body.TenantName,
		DeptID:       body.DeptID,
		TemplateType: body.TemplateType,
		QuotaConfig:  body.QuotaConfig,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(t, true))
}

// Get GET /api/v1/tenants/:id
func (h *TenantHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if !isAdmin(c) {
		u, ok := currentUser(c, h.userSvc)
		if !ok {
			return
		}
		if u.TenantID == nil || *u.TenantID != id {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return
		}
	}

	t, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(t, false))
}

type updateTenantBody struct {
	TenantName   string `json:"tenant_name" binding:"required"`
	TemplateType string `json:"template_type"`
	QuotaConfig  string `json:"quota_config"`
	Status       string `json:"status"`
}

// Update PUT /api/v1/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body updateTenantBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.svc.Update(c.Request.Context(), id, &service.UpdateTenantRequest{
		TenantName:   body.TenantName,
		TemplateType: body.TemplateType,
		QuotaConfig:  body.QuotaConfig,
		Status:       body.Status,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(t, false))
}

// Delete DELETE /api/v1/tenants/:id
func (h *TenantHandler) Delete(c *gin.Context) {
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

// Metrics GET /api/v1/tenants/:id/metrics
func (h *TenantHandler) Metrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if !isAdmin(c) {
		u, ok := currentUser(c, h.userSvc)
		if !ok {
			return
		}
		if u.TenantID == nil || *u.TenantID != id {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, "forbidden")
			return
		}
	}

	m, err := h.svc.GetMetrics(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

func (h *TenantHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrTenantNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrDeptNotFound):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDeptHasTenant):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrTenantHasInstances):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidTemplateType),
		errors.Is(err, service.ErrQuotaConfigNotJSON),
		errors.Is(err, service.ErrTenantNameRequired):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrTenantProvisionFailed),
		errors.Is(err, service.ErrTenantDeprovisionFailed):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "tenant orchestration failed, please retry")
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
