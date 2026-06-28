package service

import (
	"context"
	"encoding/json"
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
	ErrInstanceNotFound          = errors.New("instance not found")
	ErrInstanceNameRequired      = errors.New("instance_name required")
	ErrInvalidInstanceType       = errors.New("invalid instance_type")
	ErrWorkspaceNotFoundForInstance = errors.New("tenant not found for instance")
	ErrInstanceHasInstallations  = errors.New("instance still has active integration installations")
	ErrInvalidInstanceStatus     = errors.New("invalid instance status")
	ErrInstanceNotReady          = errors.New("instance is not in a ready state for this operation")
)

var allowedInstanceTypes = map[string]struct{}{
	"metrics": {},
	"logs":    {},
	"alert":   {},
}

// allowedInstanceStatuses 限定 Update 时允许的 status 值，防止手动把状态写成任意字符串
// 破坏 worker/ScaleService 的状态机。最终状态 deleted/deleting 由 Delete 流程维护，
// 不允许通过 API 显式设置。
var allowedInstanceStatuses = map[string]struct{}{
	"creating": {},
	"running":  {},
	"failed":   {},
	"scaling":  {},
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

// InstanceService 实例生命周期管理。
type InstanceService struct {
	inst         *repository.InstanceRepository
	tenant       *repository.WorkspaceRepository
	installation *repository.IntegrationInstallationRepository
	orch         *OrchestratorService
	vmOperator   *vm.VMOperatorClient
	routeBuilder *vm.RouteBuilder
	vmQuery      *vm.QueryClient
	log          *zap.Logger
}

func NewInstanceService(
	inst *repository.InstanceRepository,
	tenant *repository.WorkspaceRepository,
	installation *repository.IntegrationInstallationRepository,
	orch *OrchestratorService,
	vmOperator *vm.VMOperatorClient,
	routeBuilder *vm.RouteBuilder,
	vmQuery *vm.QueryClient,
	log *zap.Logger,
) *InstanceService {
	return &InstanceService{
		inst:         inst,
		tenant:       tenant,
		installation: installation,
		orch:         orch,
		vmOperator:   vmOperator,
		routeBuilder: routeBuilder,
		vmQuery:      vmQuery,
		log:          log,
	}
}

// Create 创建实例：校验租户、创建记录、编排部署。
func (s *InstanceService) Create(ctx context.Context, req *CreateInstanceRequest) (*model.Instance, error) {
	if req.InstanceName == "" {
		return nil, ErrInstanceNameRequired
	}
	if !allowedInstanceType(req.InstanceType) {
		return nil, ErrInvalidInstanceType
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
		tenantID = *req.TenantID
		tenant = t
		grafanaInstanceID = req.GrafanaInstanceID
		if grafanaInstanceID == nil {
			grafanaInstanceID = t.GrafanaInstanceID
		}
	} else {
		tenantID = uuid.Nil
		grafanaInstanceID = req.GrafanaInstanceID
	}

	inst := &model.Instance{
		TenantID:          tenantID,
		ClusterID:         req.ClusterID,
		ZoneID:            req.ZoneID,
		InstanceName:      strings.TrimSpace(req.InstanceName),
		InstanceType:      req.InstanceType,
		TemplateType:      req.TemplateType,
		ReleaseName:       "ops-" + strings.ReplaceAll(tenantID.String(), "-", ""),
		Namespace:         "ops-" + strings.ReplaceAll(tenantID.String(), "-", ""),
		Spec:              defaultJSONB(req.Spec),
		GrafanaInstanceID: grafanaInstanceID,
		Status:            "creating",
	}
	if err := s.inst.Create(ctx, inst); err != nil {
		return nil, err
	}

	// CR + Operator 模式部署
	if s.vmOperator != nil && s.vmOperator.Enabled() && tenant != nil {
		ns := inst.Namespace
		if ns == "" {
			ns = "ops-" + strings.ReplaceAll(tenantID.String(), "-", "")
		}
		if inst.TemplateType == "shared" {
			// 共享模式：在已有共享 VMCluster 上创建 VMUser + VMRoute
			routeSet := s.routeBuilder.BuildWorkspaceRoutes(tenant)
			if err := s.vmOperator.ApplyWorkspaceUser(ctx, tenant, routeSet); err != nil {
				s.log.Error("instance_vmuser_apply_failed",
					zap.String("instance_id", inst.ID.String()),
					zap.String("tenant_id", tenant.ID.String()),
					zap.Error(err))
				if err := s.markFailed(ctx, inst.ID, "apply vmuser"); err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("apply vmuser: %w", err)
			}
			s.log.Info("instance_vmuser_created",
				zap.String("instance_id", inst.ID.String()),
				zap.String("tenant_id", tenant.ID.String()))
		} else if inst.TemplateType == "dedicated_cluster" || inst.TemplateType == "dedicated_single" {
			// 独享集群：创建 VMCluster CR，Operator 自动编排组件
			clusterSpec := buildVMClusterSpecFromInstance(inst, ns)
			if err := s.vmOperator.ApplyVMCluster(ctx, clusterSpec); err != nil {
				s.log.Error("instance_vmcluster_apply_failed",
					zap.String("instance_id", inst.ID.String()),
					zap.String("tenant_id", tenant.ID.String()),
					zap.Error(err))
				if err := s.markFailed(ctx, inst.ID, "apply vmcluster"); err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("apply vmcluster: %w", err)
			}
			s.log.Info("instance_vmcluster_created",
				zap.String("instance_id", inst.ID.String()),
				zap.String("tenant_id", tenant.ID.String()))
		}
		// 部署成功后显式转入 running，不依赖 worker 的 InstanceStatusAutoAdvance。
		if err := s.setStatus(ctx, inst.ID, "running"); err != nil {
			return nil, fmt.Errorf("mark instance running: %w", err)
		}
		inst.Status = "running"
	} else if s.log != nil {
		s.log.Info("instance_deploy_skipped_operator_disabled",
			zap.String("instance_id", inst.ID.String()),
			zap.String("reason", "vm operator or tenant not available"))
		// 未启用 Operator 时（如平台级实例或纯记录场景），直接置 running。
		if err := s.setStatus(ctx, inst.ID, "running"); err != nil {
			return nil, fmt.Errorf("mark instance running: %w", err)
		}
		inst.Status = "running"
	}

	return inst, nil
}

// setStatus 在事务中将实例状态更新为目标值，错误上抛而非静默丢弃。
func (s *InstanceService) setStatus(ctx context.Context, id uuid.UUID, status string) error {
	if err := s.inst.UpdateStatus(ctx, id, status); err != nil {
		s.log.Error("instance_status_update_failed",
			zap.Stringer("instance_id", id),
			zap.String("status", status),
			zap.Error(err))
		return err
	}
	return nil
}

// markFailed 将实例标记为 failed；状态修正失败时上抛错误，避免留下 creating 僵尸记录。
func (s *InstanceService) markFailed(ctx context.Context, id uuid.UUID, reason string) error {
	if err := s.inst.UpdateStatus(ctx, id, "failed"); err != nil {
		s.log.Error("instance_mark_failed_failed",
			zap.Stringer("instance_id", id),
			zap.String("reason", reason),
			zap.Error(err))
		return fmt.Errorf("persist instance failed status (%s): %w", reason, err)
	}
	return nil
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

	// 回收 k8s CR 资源（实例级部署基于 VM Operator CR，非 Helm release）。
	s.reclaimInstanceResources(ctx, inst)

	// 活跃集成校验与软删在同一事务中，避免并发新增集成导致孤儿资源。
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

// reclaimInstanceResources 回收实例对应的 k8s CR 资源（best-effort）。
// shared 实例回收 VMUser；dedicated 实例回收 VMCluster CR。Helm release 属租户级，
// 由租户去配（OrchestratorService.DeleteWorkspace）负责，不在实例删除范围。
func (s *InstanceService) reclaimInstanceResources(ctx context.Context, inst *model.Instance) {
	if s.vmOperator == nil || !s.vmOperator.Enabled() {
		return
	}
	ns := inst.Namespace
	switch inst.TemplateType {
	case "shared":
		if inst.TenantID == uuid.Nil {
			return
		}
		t, err := s.tenant.GetByID(ctx, inst.TenantID)
		if err != nil || t == nil {
			s.log.Warn("instance_reclaim_tenant_missing",
				zap.Stringer("instance_id", inst.ID),
				zap.Error(err))
			return
		}
		if err := s.vmOperator.DeleteVMUser(ctx, t.VMUserID, ns); err != nil {
			s.log.Warn("instance_reclaim_vmuser_failed",
				zap.Stringer("instance_id", inst.ID),
				zap.String("vmuser_id", t.VMUserID),
				zap.Error(err))
		}
	case "dedicated_cluster", "dedicated_single":
		if err := s.vmOperator.DeleteVMCluster(ctx, inst.ReleaseName, ns); err != nil {
			s.log.Warn("instance_reclaim_vmcluster_failed",
				zap.Stringer("instance_id", inst.ID),
				zap.String("release", inst.ReleaseName),
				zap.Error(err))
		}
	}
}

// InstanceMetrics 实例资源指标占位。
type InstanceMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	Note               string  `json:"note"`
}

// Rebuild 重建实例：实例级操作，先回收该实例的 CR 资源再重新 Apply，
// 保留数据库记录与实例 ID。仅 running 状态允许重建。
func (s *InstanceService) Rebuild(ctx context.Context, id uuid.UUID) error {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstanceNotFound
	}
	if inst.Status != "running" {
		return ErrInstanceNotReady
	}

	if s.vmOperator == nil || !s.vmOperator.Enabled() {
		return nil
	}

	// 标记 creating，避免与其它操作并发。
	if err := s.setStatus(ctx, inst.ID, "creating"); err != nil {
		return err
	}

	// 先回收该实例自身资源（不影响同租户其它实例），再重新 Apply。
	s.reclaimInstanceResources(ctx, inst)
	if err := s.applyInstanceResources(ctx, inst); err != nil {
		if mErr := s.markFailed(ctx, inst.ID, "rebuild"); mErr != nil {
			return mErr
		}
		return err
	}

	if err := s.setStatus(ctx, inst.ID, "running"); err != nil {
		return err
	}
	return nil
}

// Upgrade 升级实例：实例级操作，对同一 CR 重新 Apply（幂等 upsert），
// 不删除命名空间、不影响同租户其它实例。仅 running 状态允许升级。
func (s *InstanceService) Upgrade(ctx context.Context, id uuid.UUID) error {
	inst, err := s.inst.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstanceNotFound
	}
	if inst.Status != "running" {
		return ErrInstanceNotReady
	}

	if s.vmOperator == nil || !s.vmOperator.Enabled() {
		return nil
	}

	if err := s.applyInstanceResources(ctx, inst); err != nil {
		return err
	}
	return nil
}

// applyInstanceResources 按模板重新 Apply 实例对应的 CR（实例级部署）。
func (s *InstanceService) applyInstanceResources(ctx context.Context, inst *model.Instance) error {
	if inst.TemplateType == "shared" {
		if inst.TenantID == uuid.Nil {
			return nil
		}
		t, err := s.tenant.GetByID(ctx, inst.TenantID)
		if err != nil {
			return fmt.Errorf("load tenant for rebuild/upgrade: %w", err)
		}
		if t == nil {
			return ErrWorkspaceNotFoundForInstance
		}
		routeSet := s.routeBuilder.BuildWorkspaceRoutes(t)
		if err := s.vmOperator.ApplyWorkspaceUser(ctx, t, routeSet); err != nil {
			return fmt.Errorf("apply vmuser: %w", err)
		}
		return nil
	}
	if inst.TemplateType == "dedicated_cluster" || inst.TemplateType == "dedicated_single" {
		clusterSpec := buildVMClusterSpecFromInstance(inst, inst.Namespace)
		if err := s.vmOperator.ApplyVMCluster(ctx, clusterSpec); err != nil {
			return fmt.Errorf("apply vmcluster: %w", err)
		}
		return nil
	}
	return nil
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
type instanceSpecFromJSON struct {
	Mode       string `json:"mode"`
	Retention  int    `json:"retention"`
	Replicas   int    `json:"replicas"`
	VMStorage  componentSpec `json:"vmstorage"`
	VMSelect   componentSpec `json:"vmselect"`
	VMInsert   componentSpec `json:"vminsert"`
}

type componentSpec struct {
	CPU     int `json:"cpu"`
	Memory  int `json:"memory"`
	Storage int `json:"storage"` // vmstorage only
}

// buildVMClusterSpecFromInstance 从 instance 记录和 spec JSON 构建 VMCluster CR 规格。
func buildVMClusterSpecFromInstance(inst *model.Instance, namespace string) vm.VMClusterSpec {
	var parsed instanceSpecFromJSON
	_ = json.Unmarshal([]byte(inst.Spec), &parsed)

	if parsed.Replicas < 1 {
		parsed.Replicas = 2
	}
	if parsed.VMStorage.CPU < 1 {
		parsed.VMStorage.CPU = 4
	}
	if parsed.VMStorage.Memory < 1 {
		parsed.VMStorage.Memory = 8
	}
	if parsed.VMStorage.Storage < 1 {
		parsed.VMStorage.Storage = 200
	}
	if parsed.VMSelect.CPU < 1 {
		parsed.VMSelect.CPU = 2
	}
	if parsed.VMSelect.Memory < 1 {
		parsed.VMSelect.Memory = 4
	}
	if parsed.VMInsert.CPU < 1 {
		parsed.VMInsert.CPU = 2
	}
	if parsed.VMInsert.Memory < 1 {
		parsed.VMInsert.Memory = 4
	}

	retention := fmt.Sprintf("%dd", parsed.Retention)
	if parsed.Retention < 1 {
		retention = "15d"
	}

	return vm.VMClusterSpec{
		Name:              inst.ReleaseName,
		Namespace:         namespace,
		TenantID:          inst.TenantID.String(),
		RetentionPeriod:   retention,
		VMInsertReplicas:  int32(parsed.Replicas),
		VMInsertCPU:       fmt.Sprintf("%d", parsed.VMInsert.CPU),
		VMInsertMemory:    fmt.Sprintf("%dGi", parsed.VMInsert.Memory),
		VMSelectReplicas:  int32(parsed.Replicas),
		VMSelectCPU:       fmt.Sprintf("%d", parsed.VMSelect.CPU),
		VMSelectMemory:    fmt.Sprintf("%dGi", parsed.VMSelect.Memory),
		VMStorageReplicas: int32(parsed.Replicas),
		VMStorageCPU:      fmt.Sprintf("%d", parsed.VMStorage.CPU),
		VMStorageMemory:   fmt.Sprintf("%dGi", parsed.VMStorage.Memory),
		VMStorageSize:     fmt.Sprintf("%dGi", parsed.VMStorage.Storage),
	}
}
