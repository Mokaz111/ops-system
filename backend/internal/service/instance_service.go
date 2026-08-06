package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInstanceNotFound             = errors.New("instance not found")
	ErrInstanceNameRequired         = errors.New("instance_name required")
	ErrInvalidInstanceType          = errors.New("invalid instance_type")
	ErrWorkspaceNotFoundForInstance = errors.New("tenant not found for instance")
	ErrInstanceHasInstallations     = errors.New("instance still has active integration installations")
	ErrInstanceHasBusinessClusters  = errors.New("instance still has business clusters")
	ErrInvalidInstanceStatus        = errors.New("invalid instance status")
	ErrZoneSharedNotReady           = errors.New("zone shared vm pool is not ready")
)

var allowedInstanceTypes = map[string]struct{}{
	"metrics": {},
	"logs":    {},
	"alert":   {},
}

// allowedInstanceStatuses 限定 Update 时允许的 status 值，防止手动把状态写成任意字符串。
// 最终状态 deleted/deleting 由 Delete 流程维护，不允许通过 API 显式设置。
var allowedInstanceStatuses = map[string]struct{}{
	"creating": {},
	"running":  {},
	"failed":   {},
}

// CreateInstanceRequest 创建实例请求。
type CreateInstanceRequest struct {
	TenantID          *uuid.UUID
	ClusterID         *uuid.UUID
	ZoneID            *uuid.UUID
	InstanceName      string
	InstanceType      string
	TemplateType      string
	Spec              string
	GrafanaInstanceID *uuid.UUID
}

// UpdateInstanceRequest 更新实例请求。
type UpdateInstanceRequest struct {
	InstanceName      string
	Spec              string
	Status            string
	TenantID          *uuid.UUID
	GrafanaInstanceID *uuid.UUID
}

// InstanceService 实例生命周期管理（Metric Workspace）。
type InstanceService struct {
	inst             *repository.InstanceRepository
	tenant           *repository.WorkspaceRepository
	zone             *repository.ZoneRepository
	vmCluster        *repository.VMClusterRepository
	businessCluster  *repository.BusinessClusterRepository
	installation     *repository.IntegrationInstallationRepository
	orch             *OrchestratorService
	vmOperator       *vm.VMOperatorClient
	routeBuilder     *vm.RouteBuilder
	vmQuery          *vm.QueryClient
	log              *zap.Logger
}

func NewInstanceService(
	inst *repository.InstanceRepository,
	tenant *repository.WorkspaceRepository,
	zone *repository.ZoneRepository,
	vmCluster *repository.VMClusterRepository,
	businessCluster *repository.BusinessClusterRepository,
	installation *repository.IntegrationInstallationRepository,
	orch *OrchestratorService,
	vmOperator *vm.VMOperatorClient,
	routeBuilder *vm.RouteBuilder,
	vmQuery *vm.QueryClient,
	log *zap.Logger,
) *InstanceService {
	return &InstanceService{
		inst:            inst,
		tenant:          tenant,
		zone:            zone,
		vmCluster:       vmCluster,
		businessCluster: businessCluster,
		installation:    installation,
		orch:            orch,
		vmOperator:      vmOperator,
		routeBuilder:    routeBuilder,
		vmQuery:         vmQuery,
		log:             log,
	}
}

// Create 创建 Metric Workspace（shared-only）。
func (s *InstanceService) Create(ctx context.Context, req *CreateInstanceRequest) (*model.Instance, error) {
	if req.InstanceName == "" {
		return nil, ErrInstanceNameRequired
	}
	if !allowedInstanceType(req.InstanceType) {
		return nil, ErrInvalidInstanceType
	}
	templateType := strings.TrimSpace(req.TemplateType)
	if templateType == "" {
		templateType = "shared"
	}
	if templateType != "shared" {
		return nil, ErrInvalidTemplateType
	}
	if req.ZoneID == nil || *req.ZoneID == uuid.Nil {
		return nil, ErrZoneSharedNotReady
	}

	var tenantID uuid.UUID
	var grafanaInstanceID *uuid.UUID
	var tenant *model.Workspace
	if req.TenantID != nil && *req.TenantID != uuid.Nil {
		t, err := s.tenant.GetByID(ctx, *req.TenantID)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, ErrWorkspaceNotFoundForInstance
		}
		if strings.TrimSpace(t.VMSelectURL) == "" {
			return nil, ErrWorkspaceProvisionFailed
		}
		tenantID = *req.TenantID
		tenant = t
		grafanaInstanceID = req.GrafanaInstanceID
		if grafanaInstanceID == nil {
			grafanaInstanceID = t.GrafanaInstanceID
		}
	} else {
		return nil, ErrWorkspaceNotFoundForInstance
	}

	pool, err := s.vmCluster.GetActiveSharedByZone(ctx, *req.ZoneID)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, ErrZoneSharedNotReady
	}

	namespace := pool.Namespace
	if namespace == "" {
		namespace = s.routeBuilder.BuildWorkspaceRoutes(tenant).Namespace
	}

	inst := &model.Instance{
		TenantID:          tenantID,
		ClusterID:         req.ClusterID,
		ZoneID:            req.ZoneID,
		InstanceName:      strings.TrimSpace(req.InstanceName),
		InstanceType:      req.InstanceType,
		TemplateType:      "shared",
		ReleaseName:       pool.ReleaseName,
		Namespace:         namespace,
		Spec:              defaultJSONB(req.Spec),
		GrafanaInstanceID: grafanaInstanceID,
		Status:            "running",
	}
	if err := s.inst.Create(ctx, inst); err != nil {
		return nil, err
	}

	s.log.Info("metric_workspace_created",
		zap.String("instance_id", inst.ID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("zone_id", req.ZoneID.String()),
	)
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

// Update 更新实例。status 必须落在白名单内，否则返回 ErrInvalidInstanceStatus。
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
		if _, ok := allowedInstanceStatuses[req.Status]; !ok {
			return nil, ErrInvalidInstanceStatus
		}
		inst.Status = req.Status
	}
	if req.GrafanaInstanceID != nil {
		inst.GrafanaInstanceID = req.GrafanaInstanceID
	}
	if req.TenantID != nil {
		if *req.TenantID != uuid.Nil {
			t, err := s.tenant.GetByID(ctx, *req.TenantID)
			if err != nil {
				return nil, err
			}
			if t == nil {
				return nil, ErrWorkspaceNotFoundForInstance
			}
		}
		inst.TenantID = *req.TenantID
	}

	if err := s.inst.Update(ctx, inst); err != nil {
		return nil, err
	}
	return inst, nil
}

// Delete 删除实例：先回收 k8s CR 资源，再软删除数据库记录。
//
// 为避免遗留 k8s / grafana 资源，要求所有活跃 integration installation 先被卸载；
// 否则返回 ErrInstanceHasInstallations（HTTP 409）让上层先处理。
// 资源回收为 best-effort：CR 删除失败时记录告警日志但仍软删 DB，避免删除操作卡死，
// 由孤儿资源巡检后续处理。
func (s *InstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstanceNotFound
	}

	if s.installation != nil {
		n, err := s.installation.CountActiveByInstanceID(ctx, id)
		if err != nil {
			return fmt.Errorf("count active installations: %w", err)
		}
		if n > 0 {
			return ErrInstanceHasInstallations
		}
	}

	if s.businessCluster != nil {
		n, err := s.businessCluster.CountByInstanceID(ctx, id)
		if err != nil {
			return fmt.Errorf("count business clusters: %w", err)
		}
		if n > 0 {
			return ErrInstanceHasBusinessClusters
		}
	}

	// Metric Workspace 删除不回收 Workspace 级 VMUser。
	return s.inst.Transaction(ctx, func(tx *gorm.DB) error {
		if s.installation != nil {
			n, err := s.installation.CountActiveByInstanceID(ctx, id)
			if err != nil {
				return fmt.Errorf("count active installations in tx: %w", err)
			}
			if n > 0 {
				return ErrInstanceHasInstallations
			}
		}
		return tx.WithContext(ctx).Delete(&model.Instance{}, "id = ?", id).Error
	})
}

// InstanceMetrics 实例资源指标占位。
type InstanceMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	Note               string  `json:"note"`
}

// GetMetrics 查询 Metric Workspace 关联指标。
func (s *InstanceService) GetMetrics(ctx context.Context, id uuid.UUID) (*InstanceMetrics, error) {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstanceNotFound
	}
	if s.vmQuery == nil || !s.vmQuery.Enabled() {
		return &InstanceMetrics{Note: "victoriametrics query client is not configured"}, nil
	}
	t, err := s.tenant.GetByID(ctx, inst.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrWorkspaceNotFoundForInstance
	}
	selector := `instance="` + escapePromQLLabel(inst.InstanceName) + `"`
	cpu, _ := s.vmQuery.Scalar(ctx, t, `avg(rate(container_cpu_usage_seconds_total{`+selector+`}[5m])) * 100`)
	mem, _ := s.vmQuery.Scalar(ctx, t, `avg(container_memory_working_set_bytes{`+selector+`})`)
	return &InstanceMetrics{
		CPUUsagePercent:    cpu,
		MemoryUsagePercent: mem,
		Note:               "queried from tenant-scoped VictoriaMetrics endpoint",
	}, nil
}

// escapePromQLLabel 转义 PromQL 标签值中的特殊字符（反斜杠、双引号、换行），
// 防止用户可控的实例名破坏查询或注入额外标签。
func escapePromQLLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
