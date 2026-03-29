package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// GrafanaHandler Grafana 管理 HTTP 端点。
type GrafanaHandler struct {
	svc *service.GrafanaService
}

func NewGrafanaHandler(svc *service.GrafanaService) *GrafanaHandler {
	return &GrafanaHandler{svc: svc}
}

// ListOrgs GET /api/v1/grafana/orgs
func (h *GrafanaHandler) ListOrgs(c *gin.Context) {
	orgs, err := h.svc.ListOrgs(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, orgs)
}

type createOrgBody struct {
	Name string `json:"name" binding:"required"`
}

// CreateOrg POST /api/v1/grafana/orgs
func (h *GrafanaHandler) CreateOrg(c *gin.Context) {
	var body createOrgBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	orgID, err := h.svc.CreateOrg(c.Request.Context(), body.Name)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"org_id": orgID, "name": body.Name})
}

// DeleteOrg DELETE /api/v1/grafana/orgs/:id
func (h *GrafanaHandler) DeleteOrg(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	if err := h.svc.DeleteOrg(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListOrgUsers GET /api/v1/grafana/orgs/:id/users
func (h *GrafanaHandler) ListOrgUsers(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	users, err := h.svc.ListOrgUsers(c.Request.Context(), orgID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, users)
}

type addOrgUserBody struct {
	LoginOrEmail string `json:"login_or_email" binding:"required"`
	Role         string `json:"role" binding:"required"`
}

// AddOrgUser POST /api/v1/grafana/orgs/:id/users
func (h *GrafanaHandler) AddOrgUser(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	var body addOrgUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.AddOrgUser(c.Request.Context(), orgID, body.LoginOrEmail, body.Role); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// RemoveOrgUser DELETE /api/v1/grafana/orgs/:id/users/:userId
func (h *GrafanaHandler) RemoveOrgUser(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.svc.RemoveOrgUser(c.Request.Context(), orgID, userID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListDatasources GET /api/v1/grafana/orgs/:id/datasources
func (h *GrafanaHandler) ListDatasources(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	dss, err := h.svc.ListDatasources(c.Request.Context(), orgID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, dss)
}

type createDatasourceBody struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"`
	URL       string `json:"url" binding:"required"`
	Access    string `json:"access"`
	IsDefault bool   `json:"is_default"`
}

// CreateDatasource POST /api/v1/grafana/orgs/:id/datasources
func (h *GrafanaHandler) CreateDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	var body createDatasourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	access := body.Access
	if access == "" {
		access = "proxy"
	}
	if err := h.svc.CreateDatasource(c.Request.Context(), orgID, &service.CreateDatasourceRequest{
		Name:      body.Name,
		Type:      body.Type,
		URL:       body.URL,
		Access:    access,
		IsDefault: body.IsDefault,
	}); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// DeleteDatasource DELETE /api/v1/grafana/orgs/:id/datasources/:dsId
func (h *GrafanaHandler) DeleteDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	dsID, err := strconv.ParseInt(c.Param("dsId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid datasource id")
		return
	}
	if err := h.svc.DeleteDatasource(c.Request.Context(), orgID, dsID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ImportDashboard POST /api/v1/grafana/orgs/:id/dashboards/import
func (h *GrafanaHandler) ImportDashboard(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid org id")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "empty dashboard json")
		return
	}
	if err := h.svc.ImportDashboard(c.Request.Context(), orgID, body); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *GrafanaHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrGrafanaDisabled):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, service.ErrGrafanaOrgNameRequired):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrTenantNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
	}
}
