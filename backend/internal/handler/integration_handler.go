package handler

import (
	"errors"
	"net/http"

	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IntegrationHandler 接入中心 HTTP（模版 + 安装记录）。
type IntegrationHandler struct {
	tplSvc     *service.IntegrationTemplateService
	installSvc *service.IntegrationInstallationService
	userSvc    *service.UserService
}

func NewIntegrationHandler(
	tplSvc *service.IntegrationTemplateService,
	installSvc *service.IntegrationInstallationService,
	userSvc *service.UserService,
) *IntegrationHandler {
	return &IntegrationHandler{tplSvc: tplSvc, installSvc: installSvc, userSvc: userSvc}
}

// ListCategories GET /api/v1/integrations/categories
// M1 返回预置分类清单，M2 起由 DB 动态聚合。
func (h *IntegrationHandler) ListCategories(c *gin.Context) {
	response.JSON(c, []gin.H{
		{"key": "monitor", "label": "监控"},
		{"key": "db", "label": "数据库"},
		{"key": "middleware", "label": "中间件"},
		{"key": "infra", "label": "基础设施"},
		{"key": "log", "label": "日志"},
		{"key": "cloud", "label": "云产品"},
	})
}

// ListTemplates GET /api/v1/integrations/templates
func (h *IntegrationHandler) ListTemplates(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	list, total, err := h.tplSvc.List(c.Request.Context(),
		c.Query("category"),
		c.Query("component"),
		c.Query("keyword"),
		page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// GetTemplate GET /api/v1/integrations/templates/:id
func (h *IntegrationHandler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.tplSvc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// CreateTemplate POST /api/v1/integrations/templates (admin)
func (h *IntegrationHandler) CreateTemplate(c *gin.Context) {
	u, ok := currentUser(c, h.userSvc)
	if !ok {
		return
	}
	var body service.CreateIntegrationTemplateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.tplSvc.Create(c.Request.Context(), u.Username, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// UpdateTemplate PUT /api/v1/integrations/templates/:id (admin)
func (h *IntegrationHandler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.UpdateIntegrationTemplateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.tplSvc.Update(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// DeleteTemplate DELETE /api/v1/integrations/templates/:id (admin)
func (h *IntegrationHandler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.tplSvc.Delete(c.Request.Context(), id); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// ListVersions GET /api/v1/integrations/templates/:id/versions
func (h *IntegrationHandler) ListVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	list, err := h.tplSvc.ListVersions(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, list)
}

// DeleteVersion DELETE /api/v1/integrations/templates/:id/versions/:version (admin)
func (h *IntegrationHandler) DeleteVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	version := c.Param("version")
	if version == "" {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "version required")
		return
	}
	if err := h.tplSvc.DeleteVersion(c.Request.Context(), id, version); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

// CreateVersion POST /api/v1/integrations/templates/:id/versions (admin)
func (h *IntegrationHandler) CreateVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	var body service.CreateVersionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	v, err := h.tplSvc.CreateVersion(c.Request.Context(), id, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, v)
}

// InstallPlan POST /api/v1/integrations/install/plan
// 服务端 dry-run：只渲染不落库。
func (h *IntegrationHandler) InstallPlan(c *gin.Context) {
	var body service.InstallRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if !h.installPlanTenantGuard(c, &body) {
		return
	}
	plan, err := h.installSvc.Plan(c.Request.Context(), &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, plan)
}

// Install POST /api/v1/integrations/install
func (h *IntegrationHandler) Install(c *gin.Context) {
	u, ok := currentUser(c, h.userSvc)
	if !ok {
		return
	}
	var body service.InstallRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	// 非 admin 只能在自己的 tenant 下安装。
	if !assertTenantAccess(c, h.userSvc, body.TenantID) {
		return
	}
	m, err := h.installSvc.Install(c.Request.Context(), u.Username, &body)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, m)
}

// InstallPlan POST /api/v1/integrations/install/plan
// Plan 和 Install 共用 InstallRequest，同样要校验租户。
func (h *IntegrationHandler) installPlanTenantGuard(c *gin.Context, body *service.InstallRequest) bool {
	return assertTenantAccess(c, h.userSvc, body.TenantID)
}

// ListInstallations GET /api/v1/integrations/installations
func (h *IntegrationHandler) ListInstallations(c *gin.Context) {
	page, ps, ok := parsePageAndSize(c, 20)
	if !ok {
		return
	}
	scope, ok := resolveTenantScope(c, h.userSvc)
	if !ok {
		return
	}
	var f repository.IntegrationInstallationListFilter
	f.TenantID = scope
	if raw := c.Query("instance_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid instance_id")
			return
		}
		f.InstanceID = &id
	}
	if raw := c.Query("template_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid template_id")
			return
		}
		f.TemplateID = &id
	}
	f.Status = c.Query("status")
	list, total, err := h.installSvc.List(c.Request.Context(), f, page, ps)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, gin.H{"items": list, "total": total, "page": page, "page_size": ps})
}

// GetInstallation GET /api/v1/integrations/installations/:id
func (h *IntegrationHandler) GetInstallation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.installSvc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, m.TenantID) {
		return
	}
	response.JSON(c, m)
}

// ListInstallationRevisions GET /api/v1/integrations/installations/:id/revisions
func (h *IntegrationHandler) ListInstallationRevisions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	// 先按 id 加载 installation 做 tenant 校验，再返回 revisions。
	m, err := h.installSvc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, m.TenantID) {
		return
	}
	list, err := h.installSvc.ListRevisions(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, list)
}

// Uninstall DELETE /api/v1/integrations/installations/:id
func (h *IntegrationHandler) Uninstall(c *gin.Context) {
	u, ok := currentUser(c, h.userSvc)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := h.installSvc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	if !assertTenantAccess(c, h.userSvc, m.TenantID) {
		return
	}
	if err := h.installSvc.Uninstall(c.Request.Context(), id, u.Username); err != nil {
		h.handleErr(c, err)
		return
	}
	response.JSON(c, nil)
}

func (h *IntegrationHandler) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrIntegrationTemplateNotFound),
		errors.Is(err, service.ErrIntegrationVersionNotFound),
		errors.Is(err, service.ErrIntegrationInstallationNotFound),
		errors.Is(err, service.ErrIntegrationInstanceNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrIntegrationTemplateName),
		errors.Is(err, service.ErrIntegrationVersionExists),
		errors.Is(err, service.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrIntegrationTenantMismatch):
		response.Error(c, http.StatusForbidden, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrIntegrationVersionInUse),
		errors.Is(err, service.ErrIntegrationVersionLastOne):
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
