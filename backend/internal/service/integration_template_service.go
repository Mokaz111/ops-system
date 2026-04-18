package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 接入中心相关业务错误。
var (
	ErrIntegrationTemplateNotFound = errors.New("integration template not found")
	ErrIntegrationTemplateName     = errors.New("template name required")
	ErrIntegrationVersionExists    = errors.New("template version already exists")
	ErrIntegrationVersionNotFound  = errors.New("template version not found")
	ErrIntegrationVersionInUse     = errors.New("template version still referenced by active installations")
	ErrIntegrationVersionLastOne   = errors.New("cannot delete the only remaining version")
)

// IntegrationTemplateService 模版业务（M1 骨架，仅元数据 CRUD + 版本登记）。
type IntegrationTemplateService struct {
	repo             *repository.IntegrationTemplateRepository
	installationRepo *repository.IntegrationInstallationRepository
}

func NewIntegrationTemplateService(
	repo *repository.IntegrationTemplateRepository,
	installationRepo *repository.IntegrationInstallationRepository,
) *IntegrationTemplateService {
	return &IntegrationTemplateService{repo: repo, installationRepo: installationRepo}
}

// CreateIntegrationTemplateRequest 创建模版。
type CreateIntegrationTemplateRequest struct {
	Name        string   `json:"name" binding:"required"`
	DisplayName string   `json:"display_name"`
	Category    string   `json:"category"`
	Component   string   `json:"component"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Tags        []string `json:"tags"`
}

// Create 创建模版本体。
func (s *IntegrationTemplateService) Create(ctx context.Context, creator string, req *CreateIntegrationTemplateRequest) (*model.IntegrationTemplate, error) {
	if req.Name == "" {
		return nil, ErrIntegrationTemplateName
	}
	exist, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return nil, errors.New("template name already exists")
	}
	m := &model.IntegrationTemplate{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Category:    req.Category,
		Component:   req.Component,
		Description: req.Description,
		Icon:        req.Icon,
		Tags:        marshalJSONStringArray(req.Tags),
		Status:      "active",
		CreatedBy:   creator,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询模版。
func (s *IntegrationTemplateService) Get(ctx context.Context, id uuid.UUID) (*model.IntegrationTemplate, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrIntegrationTemplateNotFound
	}
	return m, nil
}

// UpdateIntegrationTemplateRequest 更新。
type UpdateIntegrationTemplateRequest struct {
	DisplayName string   `json:"display_name"`
	Category    string   `json:"category"`
	Component   string   `json:"component"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
}

// Update 更新模版。
func (s *IntegrationTemplateService) Update(ctx context.Context, id uuid.UUID, req *UpdateIntegrationTemplateRequest) (*model.IntegrationTemplate, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrIntegrationTemplateNotFound
	}
	if req.DisplayName != "" {
		m.DisplayName = req.DisplayName
	}
	if req.Category != "" {
		m.Category = req.Category
	}
	if req.Component != "" {
		m.Component = req.Component
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.Icon != "" {
		m.Icon = req.Icon
	}
	if req.Tags != nil {
		m.Tags = marshalJSONStringArray(req.Tags)
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete 删除模版。
func (s *IntegrationTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrIntegrationTemplateNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *IntegrationTemplateService) List(ctx context.Context, category, component, keyword string, page, pageSize int) ([]model.IntegrationTemplate, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.IntegrationTemplateListFilter{
		Category:  category,
		Component: component,
		Keyword:   keyword,
		Offset:    offset,
		Limit:     pageSize,
	})
}

// CreateVersionRequest 新增版本。
type CreateVersionRequest struct {
	Version       string `json:"version" binding:"required"`
	CollectorSpec string `json:"collector_spec"`
	AlertSpec     string `json:"alert_spec"`
	DashboardSpec string `json:"dashboard_spec"`
	Variables     string `json:"variables"`
	Changelog     string `json:"changelog"`
}

// CreateVersion 新增模版版本。
func (s *IntegrationTemplateService) CreateVersion(ctx context.Context, templateID uuid.UUID, req *CreateVersionRequest) (*model.IntegrationTemplateVersion, error) {
	tpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, ErrIntegrationTemplateNotFound
	}
	exist, err := s.repo.GetVersion(ctx, templateID, req.Version)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return nil, ErrIntegrationVersionExists
	}
	v := &model.IntegrationTemplateVersion{
		TemplateID:    templateID,
		Version:       req.Version,
		CollectorSpec: req.CollectorSpec,
		AlertSpec:     req.AlertSpec,
		DashboardSpec: req.DashboardSpec,
		Variables:     req.Variables,
		Changelog:     req.Changelog,
	}
	if err := s.repo.CreateVersion(ctx, v); err != nil {
		return nil, err
	}
	tpl.LatestVersion = req.Version
	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, err
	}
	return v, nil
}

// ListVersions 列出版本。
func (s *IntegrationTemplateService) ListVersions(ctx context.Context, templateID uuid.UUID) ([]model.IntegrationTemplateVersion, error) {
	tpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, ErrIntegrationTemplateNotFound
	}
	return s.repo.ListVersions(ctx, templateID)
}

// GetVersion 查询特定版本。
func (s *IntegrationTemplateService) GetVersion(ctx context.Context, templateID uuid.UUID, version string) (*model.IntegrationTemplateVersion, error) {
	v, err := s.repo.GetVersion(ctx, templateID, version)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrIntegrationVersionNotFound
	}
	return v, nil
}

// DeleteVersion 下架 / 删除一个模板版本。
//
// 规则：
//   - 被活跃安装记录引用（非 uninstalled / uninstall_failed）时拒绝；
//   - 不允许删除最后一个版本（避免 latest_version 悬空）；
//   - 如果删除的是 latest_version，自动把 latest_version 切到当前剩余版本中创建时间最新的那一个。
func (s *IntegrationTemplateService) DeleteVersion(ctx context.Context, templateID uuid.UUID, version string) error {
	tpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}
	if tpl == nil {
		return ErrIntegrationTemplateNotFound
	}
	v, err := s.repo.GetVersion(ctx, templateID, version)
	if err != nil {
		return err
	}
	if v == nil {
		return ErrIntegrationVersionNotFound
	}

	if s.installationRepo != nil {
		cnt, cerr := s.installationRepo.CountActiveByTemplateVersion(ctx, templateID, version)
		if cerr != nil {
			return cerr
		}
		if cnt > 0 {
			return ErrIntegrationVersionInUse
		}
	}

	all, err := s.repo.ListVersions(ctx, templateID)
	if err != nil {
		return err
	}
	if len(all) <= 1 {
		return ErrIntegrationVersionLastOne
	}

	if err := s.repo.DeleteVersion(ctx, templateID, version); err != nil {
		return err
	}

	if tpl.LatestVersion == version {
		for _, vv := range all {
			if vv.Version != version {
				tpl.LatestVersion = vv.Version
				break
			}
		}
		if err := s.repo.Update(ctx, tpl); err != nil {
			return err
		}
	}
	return nil
}
