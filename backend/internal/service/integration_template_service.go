package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 接入中心相关业务错误。
var (
	ErrIntegrationTemplateNotFound      = errors.New("integration template not found")
	ErrIntegrationTemplateName          = errors.New("template name required")
	ErrIntegrationTemplateNameExists    = errors.New("template name already exists")
	ErrIntegrationTemplateInUse         = errors.New("integration template still referenced by active installations")
	ErrIntegrationVersionExists         = errors.New("template version already exists")
	ErrIntegrationVersionNotFound       = errors.New("template version not found")
	ErrIntegrationVersionInUse          = errors.New("template version still referenced by active installations")
	ErrIntegrationVersionLastOne        = errors.New("cannot delete the only remaining version")
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
		return nil, ErrIntegrationTemplateNameExists
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

// Delete 删除模版本体；被活跃安装引用时拒绝，通过时一并软删除其所有版本。
//
// 原实现只删 tpl 不动 version，导致：
//  1. 残留 template_versions 行仍可被 GetVersion 查到，但 tpl 已 soft-deleted；
//  2. 活跃安装如果还指向该 tpl 就会出现"孤儿"引用。
// 这里加上引用检查 + 事务化级联软删除，把"删除一次模板"做成真正的幂等一步。
func (s *IntegrationTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrIntegrationTemplateNotFound
	}
	if s.installationRepo != nil {
		cnt, cerr := s.installationRepo.CountActiveByTemplateID(ctx, id)
		if cerr != nil {
			return cerr
		}
		if cnt > 0 {
			return ErrIntegrationTemplateInUse
		}
	}
	return s.repo.DeleteTemplateTx(ctx, id)
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
	// latest_version 用语义化比较决定：只有"严格大于"当前 latest 才更新。
	// 这样按历史顺序回填旧版本（比如补一个 1.0.1-hotfix 到最新 1.2.0 的模板）不会把 latest 倒退。
	if tpl.LatestVersion == "" || compareVersions(req.Version, tpl.LatestVersion) > 0 {
		tpl.LatestVersion = req.Version
		if err := s.repo.Update(ctx, tpl); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// compareVersions 比较两个版本号字符串，返回 -1 / 0 / 1。
//
// 支持两种常见写法：
//   - 纯数字三段式 "1.2.3"（可带 "v" 前缀，可带 pre-release / build metadata 后缀）；
//   - 以上解析失败时退化为字典序（向后兼容 "m1-snapshot" 之类随意命名）。
// 我们没有引入第三方 semver 库，三段数字已足够覆盖约 95% 的实际场景。
func compareVersions(a, b string) int {
	aa, aok := parseSemverPrefix(a)
	bb, bok := parseSemverPrefix(b)
	if aok && bok {
		for i := 0; i < 3; i++ {
			switch {
			case aa[i] < bb[i]:
				return -1
			case aa[i] > bb[i]:
				return 1
			}
		}
		return strings.Compare(a, b)
	}
	return strings.Compare(a, b)
}

// parseSemverPrefix 提取 "v1.2.3" / "1.2.3" / "1.2.3-rc.1+meta" 前缀的 [major, minor, patch]。
// 任一段非数字即认为整体不可解析。
func parseSemverPrefix(v string) ([3]int, bool) {
	var out [3]int
	s := strings.TrimPrefix(strings.TrimSpace(v), "v")
	if idx := strings.IndexAny(s, "-+"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
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

	// 若被删版本就是 latest_version，则按语义化版本选出剩余版本里最大的那一个作为新的 latest。
	// 原实现随便挑"创建时间最新的那一条"作为兜底，若最近一次是回填的旧版本会把 latest 往回倒。
	updateLatest := tpl.LatestVersion == version
	newLatest := tpl.LatestVersion
	if updateLatest {
		newLatest = pickLatestExcluding(all, version)
	}

	return s.repo.DeleteVersionTx(ctx, templateID, version, updateLatest, newLatest)
}

// pickLatestExcluding 从版本列表里挑出 != exclude 的最大版本（语义化比较），兜底返回第一个。
func pickLatestExcluding(all []model.IntegrationTemplateVersion, exclude string) string {
	best := ""
	for _, v := range all {
		if v.Version == exclude {
			continue
		}
		if best == "" || compareVersions(v.Version, best) > 0 {
			best = v.Version
		}
	}
	return best
}
