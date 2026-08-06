package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/logagent"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrBusinessClusterNotFound     = errors.New("business cluster not found")
	ErrBusinessClusterNameConflict = errors.New("business cluster name already exists")
	ErrVMOperatorRequired          = errors.New("vm operator crd is required on business cluster")
	ErrInvalidKubeconfig           = errors.New("invalid kubeconfig")
	ErrLogInstanceNotLinked        = errors.New("log instance not linked or not found")
	ErrInvalidCollectConfig        = errors.New("invalid collect config")
)

// BusinessClusterService 业务集群业务。
type BusinessClusterService struct {
	repo           *repository.BusinessClusterRepository
	instRepo       *repository.InstanceRepository
	logInstRepo    *repository.LogInstanceRepository
	tenantRepo     *repository.WorkspaceRepository
	zoneRepo       *repository.ZoneRepository
	vmCluster      *repository.VMClusterRepository
	logClusterRepo *repository.LogClusterRepository
	vmOperator     *vm.VMOperatorClient
	vectorAgent    *logagent.VectorAgentClient
	log            *zap.Logger
}

func NewBusinessClusterService(
	repo *repository.BusinessClusterRepository,
	instRepo *repository.InstanceRepository,
	logInstRepo *repository.LogInstanceRepository,
	tenantRepo *repository.WorkspaceRepository,
	zoneRepo *repository.ZoneRepository,
	vmCluster *repository.VMClusterRepository,
	logClusterRepo *repository.LogClusterRepository,
	vmOperator *vm.VMOperatorClient,
	vectorAgent *logagent.VectorAgentClient,
	log *zap.Logger,
) *BusinessClusterService {
	return &BusinessClusterService{
		repo: repo, instRepo: instRepo, logInstRepo: logInstRepo,
		tenantRepo: tenantRepo, zoneRepo: zoneRepo,
		vmCluster: vmCluster, logClusterRepo: logClusterRepo,
		vmOperator: vmOperator, vectorAgent: vectorAgent, log: log,
	}
}

// CreateBusinessClusterRequest 接入业务集群请求。
type CreateBusinessClusterRequest struct {
	InstanceID           string                     `json:"instance_id" binding:"required"`
	Name                 string                     `json:"name" binding:"required"`
	DisplayName          string                     `json:"display_name"`
	Kubeconfig           string                     `json:"kubeconfig"`
	KubeconfigPath       string                     `json:"kubeconfig_path"`
	Labels               map[string]string          `json:"labels"`
	MetricsCollectConfig *model.MetricsCollectConfig `json:"metrics_collect_config"`
	LogsCollectConfig    *model.LogsCollectConfig    `json:"logs_collect_config"`
}

// EnableLogsRequest 启用业务集群日志采集。
type EnableLogsRequest struct {
	LogInstanceID string `json:"log_instance_id" binding:"required"`
}

// CollectConfigView 采集配置读写视图（始终带默认值）。
type CollectConfigView struct {
	Metrics model.MetricsCollectConfig `json:"metrics"`
	Logs    model.LogsCollectConfig    `json:"logs"`
}

// UpdateCollectConfigRequest 更新采集配置。
type UpdateCollectConfigRequest struct {
	Metrics *model.MetricsCollectConfig `json:"metrics"`
	Logs    *model.LogsCollectConfig    `json:"logs"`
}

func (s *BusinessClusterService) resolveK8sClient(req *CreateBusinessClusterRequest) (*k8s.Client, error) {
	if strings.TrimSpace(req.Kubeconfig) != "" {
		return k8s.NewClientFromKubeconfigContent(req.Kubeconfig)
	}
	path := strings.TrimSpace(req.KubeconfigPath)
	if path == "" {
		return nil, ErrInvalidKubeconfig
	}
	cli, err := k8s.NewClient(path, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKubeconfig, err)
	}
	return cli, nil
}

func (s *BusinessClusterService) k8sClientFromCluster(m *model.BusinessCluster) (*k8s.Client, error) {
	return s.resolveK8sClient(&CreateBusinessClusterRequest{
		Kubeconfig: m.Kubeconfig, KubeconfigPath: m.KubeconfigPath,
	})
}

func (s *BusinessClusterService) agentName(m *model.BusinessCluster) string {
	return "ops-" + strings.ReplaceAll(m.ID.String(), "-", "")
}

// Create 接入业务集群并下发 VMAgent。
func (s *BusinessClusterService) Create(ctx context.Context, tenantID uuid.UUID, req *CreateBusinessClusterRequest) (*model.BusinessCluster, error) {
	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil || inst.TenantID != tenantID {
		return nil, ErrInstanceNotFound
	}
	if inst.ZoneID == nil {
		return nil, ErrZoneSharedNotReady
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil || strings.TrimSpace(tenant.VMInsertURL) == "" {
		return nil, ErrWorkspaceProvisionFailed
	}

	pool, err := s.vmCluster.GetActiveSharedByZone(ctx, *inst.ZoneID)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, ErrZoneSharedNotReady
	}

	k8sCli, err := s.resolveK8sClient(req)
	if err != nil {
		return nil, err
	}
	if !vm.HasVMAgentCRD(ctx, k8sCli) {
		return nil, ErrVMOperatorRequired
	}

	zoneSlug := ""
	if s.zoneRepo != nil {
		z, zerr := s.zoneRepo.GetByID(ctx, *inst.ZoneID)
		if zerr == nil && z != nil {
			zoneSlug = z.Slug
		}
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	metricsCfg := model.DefaultMetricsCollectConfig()
	if req.MetricsCollectConfig != nil {
		metricsCfg = model.ParseMetricsCollectConfig(model.MarshalCollectConfig(*req.MetricsCollectConfig))
	}
	logsCfg := model.DefaultLogsCollectConfig()
	if req.LogsCollectConfig != nil {
		logsCfg = model.ParseLogsCollectConfig(model.MarshalCollectConfig(*req.LogsCollectConfig))
	}
	m := &model.BusinessCluster{
		TenantID:             tenantID,
		InstanceID:           instanceID,
		Name:                 req.Name,
		DisplayName:          req.DisplayName,
		Kubeconfig:           req.Kubeconfig,
		KubeconfigPath:       req.KubeconfigPath,
		AgentStatus:          "deploying",
		LogAgentStatus:       "pending",
		Labels:               string(labelsJSON),
		MetricsCollectConfig: model.MarshalCollectConfig(metricsCfg),
		LogsCollectConfig:    model.MarshalCollectConfig(logsCfg),
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	agentSpec := vm.AgentSpec{
		Name:           s.agentName(m),
		Namespace:      "vmagent",
		RemoteWriteURL: tenant.VMInsertURL,
		BasicAuthUser:  tenant.VMUserID,
		BasicAuthPass:  tenant.VMUserKey,
		TenantID:       tenantID.String(),
		ZoneSlug:       zoneSlug,
		WorkspaceID:    inst.ID.String(),
		ClusterName:    m.Name,
		Collect:        metricsCfg,
	}

	if s.vmOperator != nil {
		if err := s.vmOperator.ApplyVMAgent(ctx, k8sCli, agentSpec); err != nil {
			m.AgentStatus = "failed"
			_ = s.repo.Update(ctx, m)
			return nil, fmt.Errorf("apply vmagent: %w", err)
		}
	}

	m.AgentStatus = "active"
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}

	s.log.Info("business_cluster_created",
		zap.String("id", m.ID.String()),
		zap.String("name", m.Name),
		zap.String("instance_id", instanceID.String()),
	)
	return m, nil
}

// EnableLogs 下发 Vector Agent（采集 → Kafka）。
func (s *BusinessClusterService) EnableLogs(ctx context.Context, tenantID, clusterID uuid.UUID, req *EnableLogsRequest) (*model.BusinessCluster, error) {
	m, err := s.repo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.TenantID != tenantID {
		return nil, ErrBusinessClusterNotFound
	}

	logInstID, err := uuid.Parse(strings.TrimSpace(req.LogInstanceID))
	if err != nil {
		return nil, ErrLogInstanceNotLinked
	}
	logInst, err := s.logInstRepo.GetByID(ctx, logInstID)
	if err != nil {
		return nil, err
	}
	if logInst == nil || logInst.TenantID != tenantID {
		return nil, ErrLogInstanceNotLinked
	}
	if logInst.ZoneID == nil {
		return nil, ErrZoneLogsNotReady
	}

	logPool, err := s.logClusterRepo.GetActiveByZone(ctx, *logInst.ZoneID)
	if err != nil {
		return nil, err
	}
	if logPool == nil {
		return nil, ErrZoneLogsNotReady
	}

	k8sCli, err := s.k8sClientFromCluster(m)
	if err != nil {
		return nil, err
	}

	zoneSlug := ""
	if s.zoneRepo != nil {
		z, _ := s.zoneRepo.GetByID(ctx, *logInst.ZoneID)
		if z != nil {
			zoneSlug = z.Slug
		}
	}

	m.LogAgentStatus = "deploying"
	m.LogInstanceID = &logInstID
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}

	agentName := "log-" + strings.ReplaceAll(m.ID.String(), "-", "")
	spec := logagent.AgentSpec{
		Name:         agentName,
		Namespace:    "vector",
		KafkaBrokers: logPool.KafkaBrokers,
		KafkaTopic:   logPool.KafkaTopic,
		TenantID:     tenantID.String(),
		ZoneSlug:     zoneSlug,
		WorkspaceID:  logInstID.String(),
		ClusterName:  m.Name,
		Collect:      model.ParseLogsCollectConfig(m.LogsCollectConfig),
	}
	if s.vectorAgent != nil {
		if err := s.vectorAgent.Apply(ctx, k8sCli, spec); err != nil {
			m.LogAgentStatus = "failed"
			_ = s.repo.Update(ctx, m)
			return nil, fmt.Errorf("apply vector agent: %w", err)
		}
	}

	m.LogAgentStatus = "active"
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	s.log.Info("business_cluster_logs_enabled", zap.String("id", m.ID.String()))
	return m, nil
}

// DisableLogs 移除 Vector Agent。
func (s *BusinessClusterService) DisableLogs(ctx context.Context, tenantID, clusterID uuid.UUID, force bool) (*model.BusinessCluster, error) {
	m, err := s.repo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.TenantID != tenantID {
		return nil, ErrBusinessClusterNotFound
	}

	if s.vectorAgent != nil {
		k8sCli, kerr := s.k8sClientFromCluster(m)
		if kerr == nil {
			agentName := "log-" + strings.ReplaceAll(m.ID.String(), "-", "")
			if derr := s.vectorAgent.Delete(ctx, k8sCli, agentName, "vector"); derr != nil && !force {
				return nil, fmt.Errorf("delete vector agent: %w", derr)
			} else if derr != nil {
				s.log.Warn("business_cluster_force_delete_vector_failed", zap.Error(derr))
			}
		} else if !force {
			return nil, kerr
		}
	}

	m.LogAgentStatus = "off"
	m.LogInstanceID = nil
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
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

	if s.vmOperator != nil {
		k8sCli, kerr := s.k8sClientFromCluster(m)
		if kerr == nil {
			if derr := s.vmOperator.DeleteVMAgent(ctx, k8sCli, s.agentName(m), "vmagent"); derr != nil && !force {
				return fmt.Errorf("delete vmagent: %w", derr)
			} else if derr != nil {
				s.log.Warn("business_cluster_force_delete_vmagent_failed", zap.Error(derr))
			}
		} else if !force {
			return kerr
		}
	}

	if s.vectorAgent != nil && m.LogAgentStatus == "active" {
		k8sCli, kerr := s.k8sClientFromCluster(m)
		if kerr == nil {
			agentName := "log-" + strings.ReplaceAll(m.ID.String(), "-", "")
			if derr := s.vectorAgent.Delete(ctx, k8sCli, agentName, "vector"); derr != nil && !force {
				return fmt.Errorf("delete vector agent: %w", derr)
			}
		} else if !force {
			return kerr
		}
	}

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

// GetCollectConfig 读取业务集群采集配置（带默认值）。
func (s *BusinessClusterService) GetCollectConfig(ctx context.Context, id uuid.UUID) (*CollectConfigView, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CollectConfigView{
		Metrics: model.ParseMetricsCollectConfig(m.MetricsCollectConfig),
		Logs:    model.ParseLogsCollectConfig(m.LogsCollectConfig),
	}, nil
}

// UpdateCollectConfig 更新采集配置并重新下发已启用的 Agent。
func (s *BusinessClusterService) UpdateCollectConfig(ctx context.Context, tenantID, clusterID uuid.UUID, req *UpdateCollectConfigRequest) (*CollectConfigView, error) {
	if req == nil || (req.Metrics == nil && req.Logs == nil) {
		return nil, ErrInvalidCollectConfig
	}
	m, err := s.repo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.TenantID != tenantID {
		return nil, ErrBusinessClusterNotFound
	}

	if req.Metrics != nil {
		cfg := model.ParseMetricsCollectConfig(model.MarshalCollectConfig(*req.Metrics))
		if err := validateMetricsCollectConfig(cfg); err != nil {
			return nil, err
		}
		m.MetricsCollectConfig = model.MarshalCollectConfig(cfg)
	}
	if req.Logs != nil {
		cfg := model.ParseLogsCollectConfig(model.MarshalCollectConfig(*req.Logs))
		m.LogsCollectConfig = model.MarshalCollectConfig(cfg)
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}

	// 指标 Agent 已运行则按新配置重放。
	if m.AgentStatus == "active" || m.AgentStatus == "failed" {
		if err := s.reapplyMetricsAgent(ctx, m); err != nil {
			m.AgentStatus = "failed"
			_ = s.repo.Update(ctx, m)
			return nil, fmt.Errorf("reapply vmagent: %w", err)
		}
		m.AgentStatus = "active"
		_ = s.repo.Update(ctx, m)
	}
	// 日志 Agent 已运行则按新配置重放。
	if m.LogAgentStatus == "active" || m.LogAgentStatus == "failed" {
		if err := s.reapplyLogsAgent(ctx, m); err != nil {
			m.LogAgentStatus = "failed"
			_ = s.repo.Update(ctx, m)
			return nil, fmt.Errorf("reapply vector agent: %w", err)
		}
		m.LogAgentStatus = "active"
		_ = s.repo.Update(ctx, m)
	}

	return &CollectConfigView{
		Metrics: model.ParseMetricsCollectConfig(m.MetricsCollectConfig),
		Logs:    model.ParseLogsCollectConfig(m.LogsCollectConfig),
	}, nil
}

func validateMetricsCollectConfig(cfg model.MetricsCollectConfig) error {
	for _, v := range []string{cfg.ScrapeInterval, cfg.ScrapeTimeout} {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if len(v) < 2 {
			return ErrInvalidCollectConfig
		}
		unit := v[len(v)-1]
		if unit != 's' && unit != 'm' && unit != 'h' {
			return ErrInvalidCollectConfig
		}
	}
	return nil
}

func (s *BusinessClusterService) reapplyMetricsAgent(ctx context.Context, m *model.BusinessCluster) error {
	inst, err := s.instRepo.GetByID(ctx, m.InstanceID)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstanceNotFound
	}
	tenant, err := s.tenantRepo.GetByID(ctx, m.TenantID)
	if err != nil {
		return err
	}
	if tenant == nil || strings.TrimSpace(tenant.VMInsertURL) == "" {
		return ErrWorkspaceProvisionFailed
	}
	k8sCli, err := s.k8sClientFromCluster(m)
	if err != nil {
		return err
	}
	zoneSlug := ""
	if inst.ZoneID != nil && s.zoneRepo != nil {
		if z, zerr := s.zoneRepo.GetByID(ctx, *inst.ZoneID); zerr == nil && z != nil {
			zoneSlug = z.Slug
		}
	}
	if s.vmOperator == nil {
		return nil
	}
	return s.vmOperator.ApplyVMAgent(ctx, k8sCli, vm.AgentSpec{
		Name:           s.agentName(m),
		Namespace:      "vmagent",
		RemoteWriteURL: tenant.VMInsertURL,
		BasicAuthUser:  tenant.VMUserID,
		BasicAuthPass:  tenant.VMUserKey,
		TenantID:       m.TenantID.String(),
		ZoneSlug:       zoneSlug,
		WorkspaceID:    inst.ID.String(),
		ClusterName:    m.Name,
		Collect:        model.ParseMetricsCollectConfig(m.MetricsCollectConfig),
	})
}

func (s *BusinessClusterService) reapplyLogsAgent(ctx context.Context, m *model.BusinessCluster) error {
	if m.LogInstanceID == nil {
		return ErrLogInstanceNotLinked
	}
	logInst, err := s.logInstRepo.GetByID(ctx, *m.LogInstanceID)
	if err != nil {
		return err
	}
	if logInst == nil || logInst.ZoneID == nil {
		return ErrLogInstanceNotLinked
	}
	logPool, err := s.logClusterRepo.GetActiveByZone(ctx, *logInst.ZoneID)
	if err != nil {
		return err
	}
	if logPool == nil {
		return ErrZoneLogsNotReady
	}
	k8sCli, err := s.k8sClientFromCluster(m)
	if err != nil {
		return err
	}
	zoneSlug := ""
	if s.zoneRepo != nil {
		if z, _ := s.zoneRepo.GetByID(ctx, *logInst.ZoneID); z != nil {
			zoneSlug = z.Slug
		}
	}
	if s.vectorAgent == nil {
		return nil
	}
	agentName := "log-" + strings.ReplaceAll(m.ID.String(), "-", "")
	return s.vectorAgent.Apply(ctx, k8sCli, logagent.AgentSpec{
		Name:         agentName,
		Namespace:    "vector",
		KafkaBrokers: logPool.KafkaBrokers,
		KafkaTopic:   logPool.KafkaTopic,
		TenantID:     m.TenantID.String(),
		ZoneSlug:     zoneSlug,
		WorkspaceID:  m.LogInstanceID.String(),
		ClusterName:  m.Name,
		Collect:      model.ParseLogsCollectConfig(m.LogsCollectConfig),
	})
}
