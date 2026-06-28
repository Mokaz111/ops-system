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

// TenantHandler 工作空间 HTTP（路由保持 /tenants 用于 API 兼容）。
type TenantHandler struct {
	svc     *service.WorkspaceService
	userSvc *service.UserService
}

func NewTenantHandler(svc *service.WorkspaceService, userSvc *service.UserService) *TenantHandler {
	return &TenantHandler{svc: svc, userSvc: userSvc}
}

type tenantResp struct {
	ID                uuid.UUID  `json:"id"`
	WorkspaceName     string     `json:"workspace_name"`
	VMUserID          string     `json:"vmuser_id"`
	VMUserKey         string     `json:"vmuser_key,omitempty"`
	TemplateType      string     `json:"template_type"`
	QuotaConfig       string     `json:"quota_config"`
	IsolationLevel    string     `json:"isolation_level,omitempty"`
	VMNamespace       string     `json:"vm_namespace,omitempty"`
	VMSelectURL       string     `json:"vm_select_url,omitempty"`
	VMInsertURL       string     `json:"vm_insert_url,omitempty"`
	Status            string     `json:"status"`
	N9ETeamID         int64      `json:"n9e_team_id"`
	GrafanaOrgID      int64      `json:"grafana_org_id"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	InsertURL         string     `json:"insert_url,omitempty"`
}

func (h *TenantHandler) toTenantResp(w *model.Workspace, withKey bool) tenantResp {
	r := tenantResp{
		ID:                w.ID,
		WorkspaceName:     w.WorkspaceName,
		VMUserID:          w.VMUserID,
		TemplateType:      w.TemplateType,
		QuotaConfig:       w.QuotaConfig,
		IsolationLevel:    w.IsolationLevel,
		VMNamespace:       w.VMNamespace,
		VMSelectURL:       w.VMSelectURL,
		VMInsertURL:       w.VMInsertURL,
		Status:            w.Status,
		N9ETeamID:         w.N9ETeamID,
		GrafanaOrgID:      w.GrafanaOrgID,
		GrafanaInstanceID: w.GrafanaInstanceID,
		CreatedAt:         w.CreatedAt,
		UpdatedAt:         w.UpdatedAt,
		InsertURL:         h.svc.InsertURL(w.VMUserID),
	}
	if withKey {
		r.VMUserKey = w.VMUserKey
	}
	return r
}

type createTenantBody struct {
	WorkspaceName     string     `json:"workspace_name" binding:"required"`
	TemplateType      string     `json:"template_type" binding:"required"`
	QuotaConfig       string     `json:"quota_config"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id"`
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
		if u.WorkspaceID == nil {
			raw := c.Query("workspace_id")
			if raw == "" {
				response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
				return
			}
			id, err := uuid.Parse(raw)
			if err != nil {
				response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid workspace_id")
				return
			}
			allowed, err := h.userSvc.CanAccessWorkspace(c.Request.Context(), u.ID, id, "read")
			if err != nil || !allowed {
				response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
				return
			}
			w, err := h.svc.Get(c.Request.Context(), id)
			if err != nil {
				h.handleErr(c, err)
				return
			}
			response.JSON(c, gin.H{
				"items":     []tenantResp{h.toTenantResp(w, false)},
				"total":     1,
				"page":      page,
				"page_size": ps,
			})
			return
		}
		w, err := h.svc.Get(c.Request.Context(), *u.WorkspaceID)
		if err != nil {
			h.handleErr(c, err)
			return
		}
		response.JSON(c, gin.H{
			"items":     []tenantResp{h.toTenantResp(w, false)},
			"total":     1,
			"page":      page,
			"page_size": ps,
		})
		return
	}

	templateType := c.Query("template_type")
	status := c.Query("status")
	keyword := c.Query("keyword")

	list, total, err := h.svc.List(c.Request.Context(), page, ps, templateType, status, keyword)
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
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	w, err := h.svc.Create(c.Request.Context(), &service.CreateWorkspaceRequest{
		WorkspaceName:     body.WorkspaceName,
		TemplateType:      body.TemplateType,
		QuotaConfig:       body.QuotaConfig,
		GrafanaInstanceID: body.GrafanaInstanceID,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(w, true))
}

// Get GET /api/v1/tenants/:id
func (h *TenantHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if !isAdmin(c) {
		caller, ok := userIDFromContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
			return
		}
		allowed, err := h.userSvc.CanAccessWorkspace(c.Request.Context(), caller, id, "read")
		if err != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
			return
		}
	}

	w, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(w, false))
}

type updateTenantBody struct {
	WorkspaceName     string     `json:"workspace_name"`
	TemplateType      string     `json:"template_type"`
	QuotaConfig       string     `json:"quota_config"`
	Status            string     `json:"status"`
	GrafanaInstanceID *uuid.UUID `json:"grafana_instance_id"`
}

// Update PUT /api/v1/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	var body updateTenantBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	w, err := h.svc.Update(c.Request.Context(), id, &service.UpdateWorkspaceRequest{
		WorkspaceName:     body.WorkspaceName,
		TemplateType:      body.TemplateType,
		QuotaConfig:       body.QuotaConfig,
		Status:            body.Status,
		GrafanaInstanceID: body.GrafanaInstanceID,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, h.toTenantResp(w, false))
}

// Delete DELETE /api/v1/tenants/:id
func (h *TenantHandler) Delete(c *gin.Context) {
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

// Metrics GET /api/v1/tenants/:id/metrics
func (h *TenantHandler) Metrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid id")
		return
	}
	if !isAdmin(c) {
		caller, ok := userIDFromContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, http.StatusUnauthorized, response.ErrCodeUnauthorized, "unauthorized")
			return
		}
		allowed, err := h.userSvc.CanAccessWorkspace(c.Request.Context(), caller, id, "read")
		if err != nil || !allowed {
			response.Error(c, http.StatusForbidden, http.StatusForbidden, response.ErrCodeForbidden, "forbidden")
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
	case errors.Is(err, service.ErrWorkspaceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeTenantNotFound, err.Error())
	case errors.Is(err, service.ErrWorkspaceSlugConflict):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeConflict, err.Error())
	case errors.Is(err, service.ErrWorkspaceHasInstances):
		response.Error(c, http.StatusConflict, http.StatusConflict, response.ErrCodeTenantHasInstances, err.Error())
	case errors.Is(err, service.ErrInvalidTemplateType),
		errors.Is(err, service.ErrQuotaConfigNotJSON),
		errors.Is(err, service.ErrWorkspaceNameRequired):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
	case errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeInvalidPagination, err.Error())
	case errors.Is(err, service.ErrWorkspaceProvisionFailed),
		errors.Is(err, service.ErrWorkspaceDeprovisionFailed):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, response.ErrCodeTenantProvisionFailed, "workspace orchestration failed, please retry")
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, response.ErrCodeInternal, "internal server error")
	}
}
