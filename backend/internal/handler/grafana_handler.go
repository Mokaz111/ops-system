package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GrafanaHandler Grafana 管理 HTTP 端点（实例中心架构，hostId 在路径中）。
type GrafanaHandler struct {
	svc *service.GrafanaService
	log *zap.Logger
}

func NewGrafanaHandler(svc *service.GrafanaService, log *zap.Logger) *GrafanaHandler {
	return &GrafanaHandler{svc: svc, log: log}
}

// resolveSvc 根据路径参数 hostId 返回对应 Grafana 实例的 Service。
func (h *GrafanaHandler) resolveSvc(c *gin.Context) (*service.GrafanaService, error) {
	raw := c.Param("hostId")
	if raw == "" {
		return h.svc, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return h.svc.ForHost(c.Request.Context(), &id)
}

// ── Organization endpoints ──

// ListOrgs GET /api/v1/grafana/instances/:hostId/orgs
func (h *GrafanaHandler) ListOrgs(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	orgs, err := svc.ListOrgs(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, orgs)
}

type createOrgBody struct {
	Name string `json:"name" binding:"required"`
}

// CreateOrg POST /api/v1/grafana/instances/:hostId/orgs
func (h *GrafanaHandler) CreateOrg(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body createOrgBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	orgID, err := svc.CreateOrg(c.Request.Context(), body.Name)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"org_id": orgID, "name": body.Name})
}

// DeleteOrg DELETE /api/v1/grafana/instances/:hostId/orgs/:orgId
func (h *GrafanaHandler) DeleteOrg(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	if err := svc.DeleteOrg(c.Request.Context(), orgID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Org users ──

// ListOrgUsers GET /api/v1/grafana/instances/:hostId/orgs/:orgId/users
func (h *GrafanaHandler) ListOrgUsers(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	users, err := svc.ListOrgUsers(c.Request.Context(), orgID)
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

// AddOrgUser POST /api/v1/grafana/instances/:hostId/orgs/:orgId/users
func (h *GrafanaHandler) AddOrgUser(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body addOrgUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	if err := svc.AddOrgUser(c.Request.Context(), orgID, body.LoginOrEmail, body.Role); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// RemoveOrgUser DELETE /api/v1/grafana/instances/:hostId/orgs/:orgId/users/:userId
func (h *GrafanaHandler) RemoveOrgUser(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid user id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	if err := svc.RemoveOrgUser(c.Request.Context(), orgID, userID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Datasources ──

// ListDatasources GET /api/v1/grafana/instances/:hostId/orgs/:orgId/datasources
func (h *GrafanaHandler) ListDatasources(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	dss, err := svc.ListDatasources(c.Request.Context(), orgID)
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

// CreateDatasource POST /api/v1/grafana/instances/:hostId/orgs/:orgId/datasources
func (h *GrafanaHandler) CreateDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body createDatasourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	access := body.Access
	if access == "" {
		access = "proxy"
	}
	if err := svc.CreateDatasource(c.Request.Context(), orgID, &service.CreateDatasourceRequest{
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

type updateDatasourceBody struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Access    string `json:"access"`
	IsDefault bool   `json:"is_default"`
}

// UpdateDatasource PUT /api/v1/grafana/instances/:hostId/orgs/:orgId/datasources/:dsId
func (h *GrafanaHandler) UpdateDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	dsID, err := strconv.ParseInt(c.Param("dsId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid datasource id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body updateDatasourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	access := body.Access
	if access == "" {
		access = "proxy"
	}
	if err := svc.UpdateDatasource(c.Request.Context(), orgID, dsID, &service.UpdateDatasourceRequest{
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

// DeleteDatasource DELETE /api/v1/grafana/instances/:hostId/orgs/:orgId/datasources/:dsId
func (h *GrafanaHandler) DeleteDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	dsID, err := strconv.ParseInt(c.Param("dsId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid datasource id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	if err := svc.DeleteDatasource(c.Request.Context(), orgID, dsID); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

type testDatasourceBody struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	URL    string `json:"url" binding:"required"`
	Access string `json:"access"`
}

// TestDatasource POST /api/v1/grafana/instances/:hostId/orgs/:orgId/datasources/test
func (h *GrafanaHandler) TestDatasource(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body testDatasourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
		return
	}
	access := body.Access
	if access == "" {
		access = "proxy"
	}
	result, err := svc.TestDatasource(c.Request.Context(), orgID, map[string]any{
		"name":   body.Name,
		"type":   body.Type,
		"url":    body.URL,
		"access": access,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, result)
}

// ── Dashboards ──

// ImportDashboard POST /api/v1/grafana/instances/:hostId/orgs/:orgId/dashboards/import
func (h *GrafanaHandler) ImportDashboard(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "empty dashboard json")
		return
	}
	if err := svc.ImportDashboard(c.Request.Context(), orgID, body); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListDashboards GET /api/v1/grafana/instances/:hostId/orgs/:orgId/dashboards
func (h *GrafanaHandler) ListDashboards(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	dashboards, err := svc.ListDashboards(c.Request.Context(), orgID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, dashboards)
}

// GetDashboard GET /api/v1/grafana/instances/:hostId/orgs/:orgId/dashboards/:uid
func (h *GrafanaHandler) GetDashboard(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	dashboard, err := svc.GetDashboard(c.Request.Context(), orgID, c.Param("uid"))
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, dashboard)
}

// DeleteDashboard DELETE /api/v1/grafana/instances/:hostId/orgs/:orgId/dashboards/:uid
func (h *GrafanaHandler) DeleteDashboard(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("orgId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid org id")
		return
	}
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	if err := svc.DeleteDashboard(c.Request.Context(), orgID, c.Param("uid")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Plugins ──

// ListPlugins GET /api/v1/grafana/instances/:hostId/plugins
func (h *GrafanaHandler) ListPlugins(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	plugins, err := svc.ListPlugins(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, plugins)
}

type installPluginBody struct {
	Version string `json:"version"`
}

// InstallPlugin POST /api/v1/grafana/instances/:hostId/plugins/:pluginId/install
func (h *GrafanaHandler) InstallPlugin(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	var body installPluginBody
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, response.TranslateBindingError(err))
			return
		}
	}
	if err := svc.InstallPlugin(c.Request.Context(), c.Param("pluginId"), body.Version); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// UninstallPlugin DELETE /api/v1/grafana/instances/:hostId/plugins/:pluginId
func (h *GrafanaHandler) UninstallPlugin(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	if err := svc.UninstallPlugin(c.Request.Context(), c.Param("pluginId")); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ── Health & Admin ──

// HealthCheck GET /api/v1/grafana/instances/:hostId/health
func (h *GrafanaHandler) HealthCheck(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	status, err := svc.HealthCheck(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, status)
}

// AdminStats GET /api/v1/grafana/instances/:hostId/admin/stats
func (h *GrafanaHandler) AdminStats(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	stats, err := svc.AdminStats(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, stats)
}

// AdminSettings GET /api/v1/grafana/instances/:hostId/admin/settings
func (h *GrafanaHandler) AdminSettings(c *gin.Context) {
	svc, err := h.resolveSvc(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid hostId")
		return
	}
	settings, err := svc.AdminSettings(c.Request.Context())
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, settings)
}

// ── Error handling ──

func (h *GrafanaHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrGrafanaDisabled):
		response.Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, response.ErrCodeGrafanaDisabled, err.Error())
	case errors.Is(err, service.ErrGrafanaOrgNameRequired):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeOrgNameRequired, err.Error())
	case errors.Is(err, service.ErrTenantNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, response.ErrCodeTenantNotFound, err.Error())
	default:
		h.log.Error("grafana_handler_error", zap.Error(err))
		response.Error(c, http.StatusBadGateway, http.StatusBadGateway, response.ErrCodeBadGateway, err.Error())
	}
}
