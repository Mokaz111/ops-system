package service

import (
	"context"
	"encoding/json"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrBusinessClusterNotFound = errors.New("business cluster not found")
	ErrBusinessClusterNameConflict = errors.New("business cluster name already exists")
)

// BusinessClusterService 业务集群业务。
type BusinessClusterService struct {
	repo     *repository.BusinessClusterRepository
	instRepo *repository.InstanceRepository
	log      *zap.Logger
}

func NewBusinessClusterService(repo *repository.BusinessClusterRepository, instRepo *repository.InstanceRepository, log *zap.Logger) *BusinessClusterService {
	return &BusinessClusterService{repo: repo, instRepo: instRepo, log: log}
}

// CreateBusinessClusterRequest 接入业务集群请求。
type CreateBusinessClusterRequest struct {
	InstanceID     string            `json:"instance_id" binding:"required"`
	Name           string            `json:"name" binding:"required"`
	DisplayName    string            `json:"display_name"`
	Kubeconfig     string            `json:"kubeconfig"`
	KubeconfigPath string            `json:"kubeconfig_path"`
	Labels         map[string]string `json:"labels"`
}

// Create 接入业务集群。
func (s *BusinessClusterService) Create(ctx context.Context, tenantID uuid.UUID, req *CreateBusinessClusterRequest) (*model.BusinessCluster, error) {
	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}

	labelsJSON, _ := json.Marshal(req.Labels)

	m := &model.BusinessCluster{
		TenantID:       tenantID,
		InstanceID:     instanceID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Kubeconfig:     req.Kubeconfig,
		KubeconfigPath: req.KubeconfigPath,
		AgentStatus:    "pending",
		Labels:         string(labelsJSON),
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	// TODO: validate kubeconfig connectivity + VM Operator CRD
	// TODO: build and apply VMAgent CR

	s.log.Info("business_cluster_created",
		zap.String("id", m.ID.String()),
		zap.String("name", m.Name),
		zap.String("instance_id", instanceID.String()),
	)
	return m, nil
}

// Get 查询。
func (s *BusinessClusterService) Get(ctx context.Context, id uuid.UUID) (*model.BusinessCluster, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrBusinessClusterNotFound
	}
	return m, nil
}

// Delete 移除业务集群。
func (s *BusinessClusterService) Delete(ctx context.Context, id uuid.UUID, force bool) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrBusinessClusterNotFound
	}

	// TODO: delete VMAgent CR from business cluster
	// on failure & force=false: return error
	// on failure & force=true: log warning and continue

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("business_cluster_removed", zap.String("id", id.String()), zap.String("name", m.Name))
	return nil
}

// List 列表。
func (s *BusinessClusterService) List(ctx context.Context, tenantID, instanceID string, page, pageSize int) ([]model.BusinessCluster, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.repo.List(ctx, repository.BusinessClusterListFilter{
		TenantID:   tenantID,
		InstanceID: instanceID,
		Offset:     (page - 1) * pageSize,
		Limit:      pageSize,
	})
}
