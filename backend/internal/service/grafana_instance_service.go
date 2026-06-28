package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// GrafanaInstance 相关业务错误。
var (
	ErrGrafanaInstanceNotFound = errors.New("grafana instance not found")
)

// GrafanaInstanceService Grafana 纳管实例注册业务。
type GrafanaInstanceService struct {
	repo *repository.GrafanaInstanceRepository
}

func NewGrafanaInstanceService(repo *repository.GrafanaInstanceRepository) *GrafanaInstanceService {
	return &GrafanaInstanceService{repo: repo}
}

// CreateGrafanaInstanceRequest 创建。
type CreateGrafanaInstanceRequest struct {
	Name          string     `json:"name" binding:"required"`
	Source        string     `json:"source" binding:"required"` // platform / external
	ZoneID        *uuid.UUID `json:"zone_id"`
	URL           string     `json:"url" binding:"required"`
	AdminUser     string     `json:"admin_user"`
	AdminPassword string     `json:"admin_password"`
	AdminToken    string     `json:"admin_token"`
}

// Create 注册 Grafana 纳管实例。
func (s *GrafanaInstanceService) Create(ctx context.Context, req *CreateGrafanaInstanceRequest) (*model.GrafanaInstance, error) {
	if req.Name == "" || req.URL == "" {
		return nil, errors.New("name and url required")
	}
	if req.Source != "platform" && req.Source != "external" {
		return nil, errors.New("source must be platform or external")
	}
	m := &model.GrafanaInstance{
		Name:          req.Name,
		Source:        req.Source,
		ZoneID:        req.ZoneID,
		URL:           req.URL,
		AdminUser:     req.AdminUser,
		AdminPassword: req.AdminPassword,
		AdminTokenEnc: req.AdminToken,
		Status:        "active",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询。
func (s *GrafanaInstanceService) Get(ctx context.Context, id uuid.UUID) (*model.GrafanaInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrGrafanaInstanceNotFound
	}
	return m, nil
}

// UpdateGrafanaInstanceRequest 更新。
type UpdateGrafanaInstanceRequest struct {
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	ZoneID        *uuid.UUID `json:"zone_id"`
	AdminUser     string     `json:"admin_user"`
	AdminPassword string     `json:"admin_password"`
	AdminToken    string     `json:"admin_token"`
	Status        string     `json:"status"`
}

// Update 更新。
func (s *GrafanaInstanceService) Update(ctx context.Context, id uuid.UUID, req *UpdateGrafanaInstanceRequest) (*model.GrafanaInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrGrafanaInstanceNotFound
	}
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.URL != "" {
		m.URL = req.URL
	}
	if req.ZoneID != nil {
		m.ZoneID = req.ZoneID
	}
	if req.AdminUser != "" {
		m.AdminUser = req.AdminUser
	}
	if req.AdminPassword != "" {
		m.AdminPassword = req.AdminPassword
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
func (s *GrafanaInstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrGrafanaInstanceNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *GrafanaInstanceService) List(ctx context.Context, source string, zoneID *uuid.UUID, page, pageSize int) ([]model.GrafanaInstance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.GrafanaInstanceListFilter{
		Source: source,
		ZoneID: zoneID,
		Offset: offset,
		Limit:  pageSize,
	})
}
