package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/k8s"
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
	ErrZoneClusterNotReady   = errors.New("zone cluster is not ready")
)

// ZoneInitSharedRequest 可用区初始化请求。
type ZoneInitSharedRequest struct {
	DryRun      bool                   `json:"dry_run"`
	Namespace   string                 `json:"namespace"`    // 可选，默认 monitoring-{slug}
	ReleaseName string                 `json:"release_name"` // 可选，默认 vm-shared-stack
	Values      map[string]interface{} `json:"values"`       // 可选的 Helm values 覆盖
}

// ZoneInitSharedPlan 可用区初始化计划（复用 PlatformBootstrapService 的字段语义）。
type ZoneInitSharedPlan struct {
	DryRun      bool                   `json:"dry_run"`
	ZoneID      string                 `json:"zone_id"`
	ZoneSlug    string                 `json:"zone_slug"`
	ClusterID   string                 `json:"cluster_id"`
	Namespace   string                 `json:"namespace"`
	ReleaseName string                 `json:"release_name"`
	Chart       string                 `json:"chart"`
	Action      string                 `json:"action"`
	Values      map[string]interface{} `json:"values"`
}

// ZoneService 可用区业务。
type ZoneService struct {
	repo        *repository.ZoneRepository
	instRepo    *repository.InstanceRepository
	clusterRepo *repository.ClusterRepository
	clientCache *k8s.ClusterClientCache
	log         *zap.Logger
}

func NewZoneService(
	repo *repository.ZoneRepository,
	instRepo *repository.InstanceRepository,
	clusterRepo *repository.ClusterRepository,
	clientCache *k8s.ClusterClientCache,
	log *zap.Logger,
) *ZoneService {
	return &ZoneService{
		repo:        repo,
		instRepo:    instRepo,
		clusterRepo: clusterRepo,
		clientCache: clientCache,
		log:         log,
	}
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

// resolveClusterClients 解析 Zone 绑定的集群，构造 Helm 和 K8s client。
// 复用 router.go 中 k8sResolver 的指纹 + ClusterClientCache 模式。
func (s *ZoneService) resolveClusterClients(
	ctx context.Context,
	clusterID uuid.UUID,
) (*helm.Client, *k8s.Client, error) {
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, nil, fmt.Errorf("zone cluster lookup: %w", err)
	}
	if cluster == nil || cluster.Status != "active" {
		return nil, nil, ErrZoneClusterNotReady
	}

	fp := fmt.Sprintf("%v|%s|%s", cluster.InCluster, cluster.KubeconfigPath, cluster.Kubeconfig)

	k8sCli := s.clientCache.Get(clusterID, fp)
	if k8sCli == nil {
		kubeconfigPath := strings.TrimSpace(cluster.KubeconfigPath)
		inCluster := cluster.InCluster
		if !inCluster && kubeconfigPath == "" {
			return nil, nil, ErrZoneClusterNotReady
		}
		if kubeconfigPath == "" && !inCluster {
			// inline kubeconfig 没有落盘，跳过并提示。
			if strings.TrimSpace(cluster.Kubeconfig) != "" {
				s.log.Warn("zone_cluster_inline_kubeconfig_ignored",
					zap.String("cluster_id", clusterID.String()),
					zap.String("cluster_name", cluster.Name),
				)
			}
			return nil, nil, ErrZoneClusterNotReady
		}
		k8sCli, err = k8s.NewClient(kubeconfigPath, inCluster)
		if err != nil {
			return nil, nil, fmt.Errorf("zone k8s client: %w", err)
		}
		s.clientCache.Put(clusterID, fp, k8sCli)
	}

	// Helm client 按需构造（仅存储 settings，足够轻量，不缓存）。
	helmKubeconfig := strings.TrimSpace(cluster.KubeconfigPath)
	if cluster.InCluster {
		helmKubeconfig = ""
	}
	helmCli, err := helm.NewClient(helmKubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("zone helm client: %w", err)
	}

	return helmCli, k8sCli, nil
}

// InitShared 在可用区绑定的集群中初始化共享 VM 监控栈。
func (s *ZoneService) InitShared(ctx context.Context, zoneID uuid.UUID, req *ZoneInitSharedRequest) (*ZoneInitSharedPlan, error) {
	z, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}
	if z.Status == "offline" {
		return nil, ErrZoneOffline
	}

	helmCli, k8sCli, err := s.resolveClusterClients(ctx, z.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = "monitoring-" + z.Slug
	}
	release := strings.TrimSpace(req.ReleaseName)
	if release == "" {
		release = "vm-shared-stack"
	}

	bootstrapSvc := NewPlatformBootstrapService(helmCli, k8sCli)
	plan, err := bootstrapSvc.InitSharedVMStack(ctx, &InitSharedClusterRequest{
		DryRun:      req.DryRun,
		Namespace:   ns,
		ReleaseName: release,
		Values:      req.Values,
	})
	if err != nil {
		return nil, err
	}

	return &ZoneInitSharedPlan{
		DryRun:      plan.DryRun,
		ZoneID:      zoneID.String(),
		ZoneSlug:    z.Slug,
		ClusterID:   z.ClusterID.String(),
		Namespace:   plan.Namespace,
		ReleaseName: plan.ReleaseName,
		Chart:       plan.Chart,
		Action:      plan.Action,
		Values:      plan.Values,
	}, nil
}

// InitGrafana 在可用区绑定的集群中初始化 Grafana。
// 复用 vm/victoria-metrics-k8s-stack chart，仅启用 Grafana 组件并禁用 VM 组件。
func (s *ZoneService) InitGrafana(ctx context.Context, zoneID uuid.UUID, req *ZoneInitSharedRequest) (*ZoneInitSharedPlan, error) {
	z, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}
	if z.Status == "offline" {
		return nil, ErrZoneOffline
	}

	helmCli, k8sCli, err := s.resolveClusterClients(ctx, z.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := "monitoring-" + z.Slug
	release := "grafana-" + z.Slug
	chart := "vm/victoria-metrics-k8s-stack"
	values := map[string]interface{}{
		"grafana": map[string]interface{}{
			"enabled": true,
		},
		"defaultRules":             map[string]interface{}{"enabled": false},
		"vmsingle":                 map[string]interface{}{"enabled": false},
		"vmcluster":                map[string]interface{}{"enabled": false},
		"vmagent":                  map[string]interface{}{"enabled": false},
		"kube-state-metrics":       map[string]interface{}{"enabled": false},
		"prometheus-node-exporter": map[string]interface{}{"enabled": false},
		"alertmanager":             map[string]interface{}{"enabled": false},
	}

	plan := &ZoneInitSharedPlan{
		DryRun:      req.DryRun,
		ZoneID:      zoneID.String(),
		ZoneSlug:    z.Slug,
		ClusterID:   z.ClusterID.String(),
		Namespace:   ns,
		ReleaseName: release,
		Chart:       chart,
		Action:      "install_or_upgrade",
		Values:      values,
	}
	if req.DryRun {
		return plan, nil
	}
	if helmCli == nil {
		return nil, ErrHelmOperatorNotConfigured
	}
	if err := helmCli.InstallOrUpgrade(ctx, release, chart, ns, values); err != nil {
		return nil, err
	}
	if k8sCli != nil {
		k8sCli.InvalidateMapperCache()
	}
	return plan, nil
}
