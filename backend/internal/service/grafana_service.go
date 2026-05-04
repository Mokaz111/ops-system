package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/config"
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
	OrgID  int64  `json:"org_id"`
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	Email  string `json:"email"`
	Role   string `json:"role"`
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

// GrafanaDashboard Grafana Dashboard 摘要。
type GrafanaDashboard struct {
	ID          int64    `json:"id"`
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	FolderID    int64    `json:"folder_id"`
	FolderTitle string   `json:"folder_title"`
}

// GrafanaPlugin Grafana 插件信息。
type GrafanaPlugin struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	Pinned  bool   `json:"pinned"`
}

// UpdateDatasourceRequest 更新数据源请求。
type UpdateDatasourceRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Access    string `json:"access"`
	IsDefault bool   `json:"is_default"`
}

// DatasourceTestResult 数据源连通性测试结果。
type DatasourceTestResult map[string]any

// TestDatasource 测试数据源连通性。
func (s *GrafanaService) TestDatasource(ctx context.Context, orgID int64, body map[string]any) (*DatasourceTestResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	out, err := s.client.TestDatasource(ctx, orgID, body)
	if err != nil {
		return nil, err
	}
	r := DatasourceTestResult(out)
	return &r, nil
}

// AdminSettings 获取 Grafana 服务器配置（需要 Basic Auth）。
func (s *GrafanaService) AdminSettings(ctx context.Context) (map[string]any, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.client.AdminSettings(ctx)
}

// GrafanaHealthStatus 健康检查结果。
type GrafanaHealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GrafanaStats 全局统计信息（来自 Admin API）。
type GrafanaStats struct {
	Users       int64 `json:"users"`
	Orgs        int64 `json:"orgs"`
	Dashboards  int64 `json:"dashboards"`
	Datasources int64 `json:"datasources"`
	ActiveUsers int64 `json:"active_users"`
}

// GrafanaService Grafana 管理（组织/用户/数据源/Dashboard）。
type GrafanaService struct {
	client       *grafana.Client
	instanceRepo *repository.GrafanaInstanceRepository
	tenantRepo   *repository.TenantRepository
	log          *zap.Logger
}

func NewGrafanaService(client *grafana.Client, instanceRepo *repository.GrafanaInstanceRepository, tenantRepo *repository.TenantRepository, log *zap.Logger) *GrafanaService {
	return &GrafanaService{client: client, instanceRepo: instanceRepo, tenantRepo: tenantRepo, log: log}
}

// ForInstance 根据 grafana_instance_id 返回对应 Grafana 实例的 Service；nil 返回自身。
func (s *GrafanaService) ForInstance(ctx context.Context, instanceID *uuid.UUID) (*GrafanaService, error) {
	if instanceID == nil || s.instanceRepo == nil {
		return s, nil
	}
	inst, err := s.instanceRepo.GetByID(ctx, *instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil || inst.Status != "active" || inst.URL == "" {
		return s, nil
	}
	resolved := grafana.NewClient(&config.GrafanaConfig{
		Enabled:       true,
		BaseURL:       inst.URL,
		APIKey:        inst.AdminTokenEnc,
		AdminUser:     inst.AdminUser,
		AdminPassword: inst.AdminPassword,
	}, s.log)
	return &GrafanaService{client: resolved, instanceRepo: s.instanceRepo, tenantRepo: s.tenantRepo, log: s.log}, nil
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

// ListDashboards 列出指定组织下的所有 Dashboard。
func (s *GrafanaService) ListDashboards(ctx context.Context, orgID int64) ([]GrafanaDashboard, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	items, err := s.client.ListDashboards(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]GrafanaDashboard, 0, len(items))
	for _, d := range items {
		out = append(out, GrafanaDashboard{
			ID:          d.ID,
			UID:         d.UID,
			Title:       d.Title,
			URL:         d.URL,
			Type:        d.Type,
			Tags:        d.Tags,
			FolderID:    d.FolderID,
			FolderTitle: d.FolderTitle,
		})
	}
	return out, nil
}

// GetDashboard 获取 Dashboard 完整 JSON。
func (s *GrafanaService) GetDashboard(ctx context.Context, orgID int64, uid string) (map[string]interface{}, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.client.GetDashboardByUID(ctx, orgID, uid)
}

// DeleteDashboard 删除指定 Dashboard。
func (s *GrafanaService) DeleteDashboard(ctx context.Context, orgID int64, uid string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.DeleteDashboardByUID(ctx, orgID, uid)
}

// ListPlugins 列出所有已安装插件。
func (s *GrafanaService) ListPlugins(ctx context.Context) ([]GrafanaPlugin, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	items, err := s.client.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GrafanaPlugin, 0, len(items))
	for _, p := range items {
		out = append(out, GrafanaPlugin{
			ID:      p.ID,
			Name:    p.Name,
			Type:    p.Type,
			Version: p.Version,
			Enabled: p.Enabled,
			Pinned:  p.Pinned,
		})
	}
	return out, nil
}

// InstallPlugin 安装 Grafana 插件。
func (s *GrafanaService) InstallPlugin(ctx context.Context, pluginID, version string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.InstallPlugin(ctx, pluginID, version)
}

// UninstallPlugin 卸载 Grafana 插件。
func (s *GrafanaService) UninstallPlugin(ctx context.Context, pluginID string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	return s.client.UninstallPlugin(ctx, pluginID)
}

// UpdateDatasource 更新数据源配置。
func (s *GrafanaService) UpdateDatasource(ctx context.Context, orgID int64, dsID int64, req *UpdateDatasourceRequest) error {
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
	return s.client.UpdateDatasource(ctx, orgID, dsID, body)
}

// AdminStats 获取 Grafana 全局统计（需要 admin_user + admin_password Basic Auth）。
func (s *GrafanaService) AdminStats(ctx context.Context) (*GrafanaStats, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	st, err := s.client.AdminStats(ctx)
	if err != nil {
		return nil, err
	}
	return &GrafanaStats{
		Users:       st.Users,
		Orgs:        st.Orgs,
		Dashboards:  st.Dashboards,
		Datasources: st.Datasources,
		ActiveUsers: st.ActiveUsers,
	}, nil
}

// HealthCheck 检查 Grafana 健康状态。
func (s *GrafanaService) HealthCheck(ctx context.Context) (*GrafanaHealthStatus, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	h, err := s.client.HealthCheck(ctx)
	if err != nil {
		return nil, err
	}
	return &GrafanaHealthStatus{
		Status:  h.Status,
		Message: h.Message,
	}, nil
}
