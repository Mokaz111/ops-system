package service

import (
	"context"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// Cluster 相关业务错误。
var (
	ErrClusterNotFound = errors.New("cluster not found")
	ErrClusterInvalid  = errors.New("invalid cluster spec")
)

// ClusterService K8s 集群注册表业务；不负责真实 client 缓存，只管元数据。
// 真正的 k8s.Client 构造在 router 里按需 lazy build + 缓存。
type ClusterService struct {
	repo *repository.ClusterRepository
}

func NewClusterService(repo *repository.ClusterRepository) *ClusterService {
	return &ClusterService{repo: repo}
}

// CreateClusterRequest 创建。
type CreateClusterRequest struct {
	Name           string `json:"name" binding:"required"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	InCluster      bool   `json:"in_cluster"`
	Kubeconfig     string `json:"kubeconfig"`
	KubeconfigPath string `json:"kubeconfig_path"`
}

// Create 注册集群。
func (s *ClusterService) Create(ctx context.Context, req *CreateClusterRequest) (*model.Cluster, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrClusterInvalid
	}
	if !req.InCluster && strings.TrimSpace(req.Kubeconfig) == "" && strings.TrimSpace(req.KubeconfigPath) == "" {
		return nil, ErrClusterInvalid
	}
	m := &model.Cluster{
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		InCluster:      req.InCluster,
		Kubeconfig:     req.Kubeconfig,
		KubeconfigPath: req.KubeconfigPath,
		Status:         "active",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateClusterRequest 更新。
type UpdateClusterRequest struct {
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	InCluster      *bool  `json:"in_cluster"`
	Kubeconfig     string `json:"kubeconfig"`
	KubeconfigPath string `json:"kubeconfig_path"`
	Status         string `json:"status"`
}

// Update 更新。
func (s *ClusterService) Update(ctx context.Context, id uuid.UUID, req *UpdateClusterRequest) (*model.Cluster, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrClusterNotFound
	}
	if req.DisplayName != "" {
		m.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.InCluster != nil {
		m.InCluster = *req.InCluster
	}
	if req.Kubeconfig != "" {
		m.Kubeconfig = req.Kubeconfig
	}
	if req.KubeconfigPath != "" {
		m.KubeconfigPath = req.KubeconfigPath
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询。
func (s *ClusterService) Get(ctx context.Context, id uuid.UUID) (*model.Cluster, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrClusterNotFound
	}
	return m, nil
}

// Delete 删除。
func (s *ClusterService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrClusterNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *ClusterService) List(ctx context.Context, status string, page, pageSize int) ([]model.Cluster, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.repo.List(ctx, repository.ClusterListFilter{
		Status: status,
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
	})
}
