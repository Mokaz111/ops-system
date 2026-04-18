package service

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 日志实例相关业务错误。
var (
	ErrLogInstanceNotFound = errors.New("log instance not found")
	ErrLogInstanceName     = errors.New("instance_name required")
)

// LogInstanceService 日志实例业务。
// 当前（M1）仅完成元数据 CRUD；实际 Helm/VLogs CR 下发在 M4 落地。
type LogInstanceService struct {
	repo *repository.LogInstanceRepository
}

func NewLogInstanceService(repo *repository.LogInstanceRepository) *LogInstanceService {
	return &LogInstanceService{repo: repo}
}

// CreateLogInstanceRequest 创建日志实例请求。
type CreateLogInstanceRequest struct {
	TenantID      uuid.UUID `json:"tenant_id" binding:"required"`
	InstanceName  string    `json:"instance_name" binding:"required"`
	Namespace     string    `json:"namespace"`
	ReleaseName   string    `json:"release_name"`
	RetentionDays int       `json:"retention_days"`
	Spec          string    `json:"spec"`
}

// Create 创建。
func (s *LogInstanceService) Create(ctx context.Context, req *CreateLogInstanceRequest) (*model.LogInstance, error) {
	if req.InstanceName == "" {
		return nil, ErrLogInstanceName
	}
	m := &model.LogInstance{
		TenantID:      req.TenantID,
		InstanceName:  req.InstanceName,
		Namespace:     req.Namespace,
		ReleaseName:   req.ReleaseName,
		RetentionDays: req.RetentionDays,
		Spec:          req.Spec,
		Status:        "creating",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询。
func (s *LogInstanceService) Get(ctx context.Context, id uuid.UUID) (*model.LogInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrLogInstanceNotFound
	}
	return m, nil
}

// UpdateLogInstanceRequest 更新请求。
type UpdateLogInstanceRequest struct {
	InstanceName  string `json:"instance_name"`
	RetentionDays int    `json:"retention_days"`
	Spec          string `json:"spec"`
	Status        string `json:"status"`
}

// Update 更新。
func (s *LogInstanceService) Update(ctx context.Context, id uuid.UUID, req *UpdateLogInstanceRequest) (*model.LogInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrLogInstanceNotFound
	}
	if req.InstanceName != "" {
		m.InstanceName = req.InstanceName
	}
	if req.RetentionDays > 0 {
		m.RetentionDays = req.RetentionDays
	}
	if req.Spec != "" {
		m.Spec = req.Spec
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
func (s *LogInstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrLogInstanceNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *LogInstanceService) List(ctx context.Context, tenantID *uuid.UUID, keyword string, page, pageSize int) ([]model.LogInstance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.LogInstanceListFilter{
		TenantID: tenantID,
		Keyword:  keyword,
		Offset:   offset,
		Limit:    pageSize,
	})
}
