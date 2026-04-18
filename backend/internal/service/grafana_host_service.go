package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// GrafanaHost 相关业务错误。
var (
	ErrGrafanaHostNotFound = errors.New("grafana host not found")
)

// GrafanaHostService Grafana 主机注册业务。
// M1 仅完成注册 CRUD，健康检查与下发调用在 M5 接入。
type GrafanaHostService struct {
	repo *repository.GrafanaHostRepository
}

func NewGrafanaHostService(repo *repository.GrafanaHostRepository) *GrafanaHostService {
	return &GrafanaHostService{repo: repo}
}

// CreateGrafanaHostRequest 创建。
type CreateGrafanaHostRequest struct {
	Name        string     `json:"name" binding:"required"`
	Scope       string     `json:"scope" binding:"required"` // platform / tenant
	TenantID    *uuid.UUID `json:"tenant_id"`
	URL         string     `json:"url" binding:"required"`
	AdminUser   string     `json:"admin_user"`
	AdminToken  string     `json:"admin_token"`
}

// Create 注册 Grafana 主机。
// 注：M1 暂不做 token 加密，直接落库；M5 替换为 KMS/AES。
func (s *GrafanaHostService) Create(ctx context.Context, req *CreateGrafanaHostRequest) (*model.GrafanaHost, error) {
	if req.Name == "" || req.URL == "" {
		return nil, errors.New("name and url required")
	}
	if req.Scope != "platform" && req.Scope != "tenant" {
		return nil, errors.New("scope must be platform or tenant")
	}
	if req.Scope == "tenant" && req.TenantID == nil {
		return nil, errors.New("tenant_id required for tenant-scoped grafana host")
	}
	m := &model.GrafanaHost{
		Name:          req.Name,
		Scope:         req.Scope,
		TenantID:      req.TenantID,
		URL:           req.URL,
		AdminUser:     req.AdminUser,
		AdminTokenEnc: req.AdminToken,
		Status:        "active",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询。
func (s *GrafanaHostService) Get(ctx context.Context, id uuid.UUID) (*model.GrafanaHost, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrGrafanaHostNotFound
	}
	return m, nil
}

// UpdateGrafanaHostRequest 更新。
type UpdateGrafanaHostRequest struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	AdminUser  string `json:"admin_user"`
	AdminToken string `json:"admin_token"`
	Status     string `json:"status"`
}

// Update 更新。
func (s *GrafanaHostService) Update(ctx context.Context, id uuid.UUID, req *UpdateGrafanaHostRequest) (*model.GrafanaHost, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrGrafanaHostNotFound
	}
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.URL != "" {
		m.URL = req.URL
	}
	if req.AdminUser != "" {
		m.AdminUser = req.AdminUser
	}
	if req.AdminToken != "" {
		m.AdminTokenEnc = req.AdminToken
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete 删除。
func (s *GrafanaHostService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrGrafanaHostNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *GrafanaHostService) List(ctx context.Context, scope string, tenantID *uuid.UUID, page, pageSize int) ([]model.GrafanaHost, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.GrafanaHostListFilter{
		Scope:    scope,
		TenantID: tenantID,
		Offset:   offset,
		Limit:    pageSize,
	})
}
