package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrGrafanaDisabled        = errors.New("grafana is not enabled")
	ErrGrafanaOrgNameRequired = errors.New("org name required")
)

// GrafanaOrg Grafana 组织信息。
type GrafanaOrg struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GrafanaOrgUser Grafana 组织用户。
type GrafanaOrgUser struct {
	OrgID    int64  `json:"org_id"`
	UserID   int64  `json:"user_id"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// GrafanaDatasource Grafana 数据源。
type GrafanaDatasource struct {
	ID        int64  `json:"id"`
	OrgID     int64  `json:"org_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Access    string `json:"access"`
	IsDefault bool   `json:"is_default"`
}

// GrafanaService Grafana 管理（组织/用户/数据源/Dashboard）。
type GrafanaService struct {
	client    *grafana.Client
	tenantRepo *repository.TenantRepository
	log       *zap.Logger
}

func NewGrafanaService(client *grafana.Client, tenantRepo *repository.TenantRepository, log *zap.Logger) *GrafanaService {
	return &GrafanaService{client: client, tenantRepo: tenantRepo, log: log}
}

func (s *GrafanaService) ensureEnabled() error {
	if s.client == nil || !s.client.Enabled() {
		return ErrGrafanaDisabled
	}
	return nil
}

// CreateOrg 创建 Grafana 组织。
func (s *GrafanaService) CreateOrg(ctx context.Context, name string) (int64, error) {
	if err := s.ensureEnabled(); err != nil {
		return 0, err
	}
	if name == "" {
		return 0, ErrGrafanaOrgNameRequired
	}
	return s.client.CreateOrg(ctx, name)
}

// DeleteOrg 删除 Grafana 组织。
func (s *GrafanaService) DeleteOrg(ctx context.Context, orgID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.DeleteOrg(ctx, orgID)
}

// ListOrgs 列出所有 Grafana 组织。
func (s *GrafanaService) ListOrgs(ctx context.Context) ([]GrafanaOrg, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	var orgs []GrafanaOrg
	if err := s.client.DoJSON(ctx, "GET", "/api/orgs", nil, 0, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

// ListOrgUsers 列出组织内用户。
func (s *GrafanaService) ListOrgUsers(ctx context.Context, orgID int64) ([]GrafanaOrgUser, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	var users []GrafanaOrgUser
	if err := s.client.DoJSON(ctx, "GET", "/api/org/users", nil, orgID, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// AddOrgUser 添加用户到组织。
func (s *GrafanaService) AddOrgUser(ctx context.Context, orgID int64, loginOrEmail, role string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.AddOrgUser(ctx, orgID, &grafana.OrgUser{LoginOrEmail: loginOrEmail, Role: role})
}

// RemoveOrgUser 从组织移除用户。
func (s *GrafanaService) RemoveOrgUser(ctx context.Context, orgID, userID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.RemoveOrgUser(ctx, orgID, userID)
}

// ListDatasources 列出组织内数据源。
func (s *GrafanaService) ListDatasources(ctx context.Context, orgID int64) ([]GrafanaDatasource, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	var dss []GrafanaDatasource
	if err := s.client.DoJSON(ctx, "GET", "/api/datasources", nil, orgID, &dss); err != nil {
		return nil, err
	}
	return dss, nil
}

// CreateDatasourceRequest 创建数据源请求。
type CreateDatasourceRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Access    string `json:"access"`
	IsDefault bool   `json:"is_default"`
}

// CreateDatasource 在指定组织中创建数据源。
func (s *GrafanaService) CreateDatasource(ctx context.Context, orgID int64, req *CreateDatasourceRequest) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	body := map[string]any{
		"name":      req.Name,
		"type":      req.Type,
		"url":       req.URL,
		"access":    req.Access,
		"isDefault": req.IsDefault,
	}
	return s.client.DoJSON(ctx, "POST", "/api/datasources", body, orgID, nil)
}

// DeleteDatasource 删除数据源。
func (s *GrafanaService) DeleteDatasource(ctx context.Context, orgID, dsID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.DeleteDatasource(ctx, orgID, dsID)
}

// ImportDashboard 导入 Dashboard JSON 到指定组织。
func (s *GrafanaService) ImportDashboard(ctx context.Context, orgID int64, dashboardJSON []byte) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.ImportDashboardJSON(ctx, orgID, dashboardJSON)
}

// CreateOrgForTenant 为租户自动创建 Grafana 组织并配置默认数据源。
func (s *GrafanaService) CreateOrgForTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if err := s.ensureEnabled(); err != nil {
		return 0, err
	}
	t, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	if t == nil {
		return 0, ErrTenantNotFound
	}
	if err := s.client.SyncTenantOnCreate(ctx, t); err != nil {
		return 0, err
	}
	_ = s.tenantRepo.Update(ctx, t)
	return t.GrafanaOrgID, nil
}
