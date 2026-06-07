package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrZoneNotFound       = errors.New("zone not found")
	ErrZoneSlugConflict   = errors.New("zone slug already exists")
	ErrZoneHasInstances   = errors.New("zone has active instances")
	ErrZoneOffline        = errors.New("zone is offline")
	ErrZoneCapacityExhausted = errors.New("zone capacity exhausted")
)

// ZoneService 可用区业务。
type ZoneService struct {
	repo      *repository.ZoneRepository
	instRepo  *repository.InstanceRepository
	log       *zap.Logger
}

func NewZoneService(repo *repository.ZoneRepository, instRepo *repository.InstanceRepository, log *zap.Logger) *ZoneService {
	return &ZoneService{repo: repo, instRepo: instRepo, log: log}
}

// CreateZoneRequest 创建 Zone 请求。
type CreateZoneRequest struct {
	Slug        string            `json:"slug" binding:"required"`
	DisplayName string            `json:"display_name" binding:"required"`
	Description string            `json:"description"`
	ClusterID   string            `json:"cluster_id" binding:"required"`
	Endpoint    string            `json:"endpoint"`
	Labels      map[string]string `json:"labels"`
	Capacity    *ZoneCapacity     `json:"capacity"`
}

// ZoneCapacity Zone 容量配置。
type ZoneCapacity struct {
	MaxInstances int `json:"max_instances"`
	MaxStorage   string `json:"max_storage"`
}

// Create 创建可用区。
func (s *ZoneService) Create(ctx context.Context, req *CreateZoneRequest) (*model.Zone, error) {
	clusterID, err := uuid.Parse(req.ClusterID)
	if err != nil {
		return nil, ErrClusterInvalid
	}

	existing, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrZoneSlugConflict
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	capacityJSON, _ := json.Marshal(req.Capacity)

	m := &model.Zone{
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Description: req.Description,
		ClusterID:   clusterID,
		Endpoint:    req.Endpoint,
		Labels:      string(labelsJSON),
		Capacity:    string(capacityJSON),
		Status:      "creating",
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	// TODO: verify K8s cluster connectivity and update status to active
	m.Status = "active"
	if err := s.repo.Update(ctx, m); err != nil {
		s.log.Warn("zone_status_update_failed", zap.String("zone_id", m.ID.String()), zap.Error(err))
	}

	s.log.Info("zone_created", zap.String("slug", m.Slug), zap.String("id", m.ID.String()))
	return m, nil
}

// UpdateZoneRequest 更新 Zone 请求。
type UpdateZoneRequest struct {
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Endpoint    string            `json:"endpoint"`
	Labels      map[string]string `json:"labels"`
	Capacity    *ZoneCapacity     `json:"capacity"`
	Status      string            `json:"status"`
}

// Update 更新可用区。
func (s *ZoneService) Update(ctx context.Context, id uuid.UUID, req *UpdateZoneRequest) (*model.Zone, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrZoneNotFound
	}
	if req.DisplayName != "" {
		m.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.Endpoint != "" {
		m.Endpoint = req.Endpoint
	}
	if req.Labels != nil {
		b, _ := json.Marshal(req.Labels)
		m.Labels = string(b)
	}
	if req.Capacity != nil {
		b, _ := json.Marshal(req.Capacity)
		m.Capacity = string(b)
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询可用区。
func (s *ZoneService) Get(ctx context.Context, id uuid.UUID) (*model.Zone, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrZoneNotFound
	}
	return m, nil
}

// GetZoneStats 获取 Zone 容量统计。
type ZoneStats struct {
	ZoneID              string `json:"zone_id"`
	TotalInstances      int64  `json:"total_instances"`
	SharedInstances     int64  `json:"shared_instances"`
	DedicatedInstances  int64  `json:"dedicated_instances"`
}

// GetStats 获取 Zone 容量统计。
func (s *ZoneService) GetStats(ctx context.Context, id uuid.UUID) (*ZoneStats, error) {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountActiveInstances(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ZoneStats{
		ZoneID:         id.String(),
		TotalInstances: total,
	}, nil
}

// Delete 删除可用区。
func (s *ZoneService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrZoneNotFound
	}

	count, err := s.repo.CountActiveInstances(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrZoneHasInstances
	}

	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *ZoneService) List(ctx context.Context, status string, page, pageSize int) ([]model.Zone, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.repo.List(ctx, repository.ZoneListFilter{
		Status: status,
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
	})
}

// CapacityCheck 检查 Zone 容量。
func (s *ZoneService) CapacityCheck(ctx context.Context, zoneID uuid.UUID) error {
	z, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return err
	}
	if z == nil {
		return ErrZoneNotFound
	}
	if z.Status == "offline" {
		return ErrZoneOffline
	}

	var cap ZoneCapacity
	if strings.TrimSpace(z.Capacity) != "" && z.Capacity != "{}" {
		if err := json.Unmarshal([]byte(z.Capacity), &cap); err != nil {
			return nil // unparseable capacity = no limit
		}
	}
	if cap.MaxInstances <= 0 {
		return nil
	}

	count, err := s.repo.CountActiveInstances(ctx, zoneID)
	if err != nil {
		return err
	}
	if count >= int64(cap.MaxInstances) {
		return ErrZoneCapacityExhausted
	}
	return nil
}
