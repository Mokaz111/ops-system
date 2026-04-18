package service

import (
	"context"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInstanceNotFound         = errors.New("instance not found")
	ErrInstanceNameRequired     = errors.New("instance_name required")
	ErrInvalidInstanceType      = errors.New("invalid instance_type")
	ErrTenantNotFoundForInstance = errors.New("tenant not found for instance")
)

var allowedInstanceTypes = map[string]struct{}{
	"metrics": {},
	"logs":    {},
	"visual":  {},
	"alert":   {},
}

// CreateInstanceRequest 创建实例请求。
type CreateInstanceRequest struct {
	TenantID     uuid.UUID
	ClusterID    *uuid.UUID
	InstanceName string
	InstanceType string
	TemplateType string
	Spec         string
}

// UpdateInstanceRequest 更新实例请求。
type UpdateInstanceRequest struct {
	InstanceName string
	Spec         string
	Status       string
}

// InstanceService 实例生命周期管理。
type InstanceService struct {
	inst   *repository.InstanceRepository
	tenant *repository.TenantRepository
	orch   *OrchestratorService
	log    *zap.Logger
}

func NewInstanceService(
	inst *repository.InstanceRepository,
	tenant *repository.TenantRepository,
	orch *OrchestratorService,
	log *zap.Logger,
) *InstanceService {
	return &InstanceService{inst: inst, tenant: tenant, orch: orch, log: log}
}

// Create 创建实例：校验租户、创建记录、编排部署。
func (s *InstanceService) Create(ctx context.Context, req *CreateInstanceRequest) (*model.Instance, error) {
	if req.InstanceName == "" {
		return nil, ErrInstanceNameRequired
	}
	if !allowedInstanceType(req.InstanceType) {
		return nil, ErrInvalidInstanceType
	}

	t, err := s.tenant.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFoundForInstance
	}

	inst := &model.Instance{
		TenantID:     req.TenantID,
		ClusterID:    req.ClusterID,
		InstanceName: strings.TrimSpace(req.InstanceName),
		InstanceType: req.InstanceType,
		TemplateType: req.TemplateType,
		Spec:         defaultJSONB(req.Spec),
		Status:       "creating",
	}
	if err := s.inst.Create(ctx, inst); err != nil {
		return nil, err
	}

	if s.orch != nil {
		if s.log != nil {
			s.log.Info("instance_deploy_placeholder",
				zap.String("instance_id", inst.ID.String()),
				zap.String("note", "instance deploy not yet implemented"))
		}
	}

	return inst, nil
}

func defaultJSONB(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "{}"
	}
	return s
}

func allowedInstanceType(s string) bool {
	_, ok := allowedInstanceTypes[s]
	return ok
}

// Get 详情。
func (s *InstanceService) Get(ctx context.Context, id uuid.UUID) (*model.Instance, error) {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}
	return inst, nil
}

// List 分页筛选。
func (s *InstanceService) List(ctx context.Context, page, pageSize int, tenantID *uuid.UUID, instanceType, status, keyword string) ([]model.Instance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.inst.List(ctx, repository.InstanceListFilter{
		TenantID:     tenantID,
		InstanceType: instanceType,
		Status:       status,
		Keyword:      keyword,
		Offset:       offset,
		Limit:        pageSize,
	})
}

// Update 更新实例。
func (s *InstanceService) Update(ctx context.Context, id uuid.UUID, req *UpdateInstanceRequest) (*model.Instance, error) {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}

	if req.InstanceName != "" {
		inst.InstanceName = strings.TrimSpace(req.InstanceName)
	}
	if req.Spec != "" {
		inst.Spec = req.Spec
	}
	if req.Status != "" {
		inst.Status = req.Status
	}

	if err := s.inst.Update(ctx, inst); err != nil {
		return nil, err
	}
	return inst, nil
}

// Delete 删除实例（先卸载编排、再软删除）。
func (s *InstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstanceNotFound
	}

	if s.orch != nil {
		if s.log != nil {
			s.log.Info("instance_undeploy_placeholder",
				zap.String("instance_id", inst.ID.String()),
				zap.String("note", "instance undeploy not yet implemented"))
		}
	}

	return s.inst.Delete(ctx, id)
}

// InstanceMetrics 实例资源指标占位。
type InstanceMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	Note               string  `json:"note"`
}

// GetMetrics 占位指标。
func (s *InstanceService) GetMetrics(ctx context.Context, id uuid.UUID) (*InstanceMetrics, error) {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}
	return &InstanceMetrics{
		Note: "placeholder; connect cluster metrics in later phase",
	}, nil
}
