package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrScaleInstanceNotFound  = errors.New("instance not found for scaling")
	ErrInvalidScaleType       = errors.New("invalid scale_type")
	ErrScaleNotSupported      = errors.New("scale operation not supported for this instance")
	ErrScaleManagedByPlatform = errors.New("shared/dedicated_cluster instances must be scaled at platform level")
	ErrScaleTypeNotAllowed    = errors.New("scale_type not allowed for this template_type")
)

// vmOperatorGroup / vmOperatorVersion 是 VictoriaMetrics Operator 维护的 CRD group/version。
// 单节点 metrics/logs 实例由 Operator 直接 reconcile，patch CR spec 会触发 sts 滚动。
const (
	vmOperatorGroup   = "operator.victoriametrics.com"
	vmOperatorVersion = "v1beta1"
)

// ListScaleEvents 按 instance 分页查询伸缩事件。
func (s *ScaleService) ListScaleEvents(ctx context.Context, f repository.ScaleEventListFilter, page, pageSize int) ([]model.ScaleEvent, int64, error) {
	if s.scaleEventRepo == nil {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	f.Offset = (page - 1) * pageSize
	f.Limit = pageSize
	return s.scaleEventRepo.List(ctx, f)
}

// vmCRResourceFor 根据 instance_type 返回对应的 VM Operator CR resource 名称（复数）。
// 仅 metrics/logs 存在对应 CR；其它类型（如 visual=Grafana）返回空表示无法 CR 直 patch。
func vmCRResourceFor(instanceType string) string {
	switch instanceType {
	case "metrics":
		return "vmsingles"
	case "logs":
		return "vlsingles"
	default:
		return ""
	}
}

// ScaleRequest 伸缩请求。
type ScaleRequest struct {
	ScaleType string
	Replicas  *int32
	CPU       string
	Memory    string
	Storage   string
	Operator  string // 审计字段，非业务必填
}

// ScaleService 实例伸缩（水平 / 垂直 / 存储）。
type ScaleService struct {
	helmClient     *helm.Client
	k8sClient      *k8s.Client
	instanceRepo   *repository.InstanceRepository
	scaleEventRepo *repository.ScaleEventRepository
	log            *zap.Logger
}

func NewScaleService(
	helmClient *helm.Client,
	k8sClient *k8s.Client,
	instanceRepo *repository.InstanceRepository,
	scaleEventRepo *repository.ScaleEventRepository,
	log *zap.Logger,
) *ScaleService {
	return &ScaleService{
		helmClient:     helmClient,
		k8sClient:      k8sClient,
		instanceRepo:   instanceRepo,
		scaleEventRepo: scaleEventRepo,
		log:            log,
	}
}

// recordEvent 把一次伸缩结果写入审计。任何 DB 错误仅记录日志，不向上冒泡。
func (s *ScaleService) recordEvent(ctx context.Context, inst *model.Instance, req *ScaleRequest, method string, err error) {
	if s.scaleEventRepo == nil || inst == nil {
		return
	}
	status := "success"
	errMsg := ""
	if err != nil {
		status = "failed"
		errMsg = err.Error()
	}
	e := &model.ScaleEvent{
		InstanceID:   inst.ID,
		InstanceName: inst.InstanceName,
		TenantID:     inst.TenantID,
		ScaleType:    req.ScaleType,
		Method:       method,
		Replicas:     req.Replicas,
		CPU:          req.CPU,
		Memory:       req.Memory,
		Storage:      req.Storage,
		Status:       status,
		ErrorMessage: errMsg,
		Operator:     req.Operator,
	}
	if werr := s.scaleEventRepo.Create(ctx, e); werr != nil && s.log != nil {
		s.log.Warn("scale_event_persist_failed", zap.Error(werr))
	}
}

// Scale 根据类型执行对应伸缩操作，并写入审计事件。
func (s *ScaleService) Scale(ctx context.Context, instanceID uuid.UUID, req *ScaleRequest) error {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrScaleInstanceNotFound
	}
	if err := validateScalePolicy(inst, req); err != nil {
		s.recordEvent(ctx, inst, req, "rejected", err)
		return err
	}

	switch req.ScaleType {
	case "horizontal":
		return s.scaleHorizontal(ctx, inst, req)
	case "vertical":
		return s.scaleVertical(ctx, inst, req)
	case "storage":
		return s.scaleStorage(ctx, inst, req)
	default:
		s.recordEvent(ctx, inst, req, "rejected", ErrInvalidScaleType)
		return ErrInvalidScaleType
	}
}

func validateScalePolicy(inst *model.Instance, req *ScaleRequest) error {
	switch inst.TemplateType {
	case "shared", "dedicated_cluster":
		// 策略A：共享版与独享集群版的容量由平台级管理员在集群层统一处理。
		return ErrScaleManagedByPlatform
	case "dedicated_single":
		// 单节点模板只允许垂直与存储扩容，不允许水平扩容。
		if req.ScaleType == "horizontal" {
			return ErrScaleTypeNotAllowed
		}
	}
	return nil
}

func (s *ScaleService) scaleHorizontal(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if s.k8sClient == nil {
		s.recordEvent(ctx, inst, req, "k8s_native", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}
	if inst.Namespace == "" {
		s.recordEvent(ctx, inst, req, "k8s_native", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}
	if req.Replicas == nil {
		err := fmt.Errorf("replicas required for horizontal scaling")
		s.recordEvent(ctx, inst, req, "k8s_native", err)
		return err
	}
	deployName := inst.ReleaseName
	if deployName == "" {
		deployName = inst.InstanceName
	}
	if err := s.k8sClient.ScaleDeployment(ctx, inst.Namespace, deployName, *req.Replicas); err != nil {
		wrapped := fmt.Errorf("scale deployment: %w", err)
		s.recordEvent(ctx, inst, req, "k8s_native", wrapped)
		return wrapped
	}
	if err := s.updateInstanceSpec(ctx, inst, req); err != nil {
		s.recordEvent(ctx, inst, req, "k8s_native", err)
		return err
	}
	s.recordEvent(ctx, inst, req, "k8s_native", nil)
	return nil
}

// scaleVertical 先尝试直接 patch VMSingle/VLSingle CR 的 spec.resources，失败或 CR 不存在时回退到 helm upgrade。
//
// 直接 patch CR 的好处：
//   - 无需加载 chart、无需重新渲染整套 values；
//   - 幂等、秒级生效（VM Operator 会 reconcile 对应 StatefulSet）；
//   - 不影响其它 values（告警、数据源等），避免 helm 意外覆盖。
func (s *ScaleService) scaleVertical(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if req.CPU == "" && req.Memory == "" {
		err := fmt.Errorf("cpu or memory required for vertical scaling")
		s.recordEvent(ctx, inst, req, "rejected", err)
		return err
	}
	if inst.Namespace == "" {
		s.recordEvent(ctx, inst, req, "rejected", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}

	patched, err := s.tryPatchVMCRResources(ctx, inst, req)
	if err != nil {
		if s.log != nil {
			s.log.Warn("scale_vertical_cr_patch_failed",
				zap.String("instance", inst.InstanceName),
				zap.String("namespace", inst.Namespace),
				zap.Error(err))
		}
	} else if patched {
		if s.log != nil {
			s.log.Info("scale_vertical_cr_patched",
				zap.String("instance", inst.InstanceName),
				zap.String("namespace", inst.Namespace),
				zap.String("type", inst.InstanceType))
		}
		if uerr := s.updateInstanceSpec(ctx, inst, req); uerr != nil {
			s.recordEvent(ctx, inst, req, "cr_patch", uerr)
			return uerr
		}
		s.recordEvent(ctx, inst, req, "cr_patch", nil)
		return nil
	}

	if s.helmClient == nil {
		s.recordEvent(ctx, inst, req, "helm_upgrade", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}
	if inst.ReleaseName == "" {
		s.recordEvent(ctx, inst, req, "helm_upgrade", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}

	resources := map[string]interface{}{}
	if req.CPU != "" {
		resources["cpu"] = req.CPU
	}
	if req.Memory != "" {
		resources["memory"] = req.Memory
	}
	vals := map[string]interface{}{
		"resources": map[string]interface{}{
			"requests": resources,
			"limits":   resources,
		},
	}

	rs, rerr := s.helmClient.GetReleaseStatus(ctx, inst.ReleaseName, inst.Namespace)
	if rerr != nil {
		wrapped := fmt.Errorf("get release status: %w", rerr)
		s.recordEvent(ctx, inst, req, "helm_upgrade", wrapped)
		return wrapped
	}
	if uerr := s.helmClient.UpgradeRelease(ctx, inst.ReleaseName, rs.Chart, inst.Namespace, vals); uerr != nil {
		wrapped := fmt.Errorf("upgrade release: %w", uerr)
		s.recordEvent(ctx, inst, req, "helm_upgrade", wrapped)
		return wrapped
	}
	if uerr := s.updateInstanceSpec(ctx, inst, req); uerr != nil {
		s.recordEvent(ctx, inst, req, "helm_upgrade", uerr)
		return uerr
	}
	s.recordEvent(ctx, inst, req, "helm_upgrade", nil)
	return nil
}

// tryPatchVMCRResources 直接 patch VMSingle/VLSingle CR 的 spec.resources。
// 返回 (patched bool, err error)。patched=false && err=nil 表示该实例不适用（调用方需回退到 helm）。
func (s *ScaleService) tryPatchVMCRResources(ctx context.Context, inst *model.Instance, req *ScaleRequest) (bool, error) {
	if s.k8sClient == nil {
		return false, nil
	}
	resource := vmCRResourceFor(inst.InstanceType)
	if resource == "" {
		return false, nil
	}
	name := inst.ReleaseName
	if name == "" {
		name = inst.InstanceName
	}
	existing, err := s.k8sClient.GetCustomResource(ctx, vmOperatorGroup, vmOperatorVersion, resource, inst.Namespace, name)
	if err != nil {
		return false, fmt.Errorf("get %s/%s: %w", resource, name, err)
	}
	if existing == nil {
		return false, nil
	}
	limits := map[string]interface{}{}
	if req.CPU != "" {
		limits["cpu"] = req.CPU
	}
	if req.Memory != "" {
		limits["memory"] = req.Memory
	}
	spec := map[string]interface{}{
		"resources": map[string]interface{}{
			"limits":   limits,
			"requests": limits,
		},
	}
	if err := s.k8sClient.PatchCustomResourceSpec(ctx, vmOperatorGroup, vmOperatorVersion, resource, inst.Namespace, name, spec); err != nil {
		return false, fmt.Errorf("patch %s/%s spec: %w", resource, name, err)
	}
	return true, nil
}

// scaleStorage 优先 patch VMSingle/VLSingle CR 的 spec.storage，让 Operator 处理 PVC expand；
// 若 CR 不存在（例如老实例直接由 helm 管理 PVC），回退到直接 PVC resize。
func (s *ScaleService) scaleStorage(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if s.k8sClient == nil {
		s.recordEvent(ctx, inst, req, "k8s_native", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}
	if inst.Namespace == "" {
		s.recordEvent(ctx, inst, req, "k8s_native", ErrScaleNotSupported)
		return ErrScaleNotSupported
	}
	if req.Storage == "" {
		err := fmt.Errorf("storage size required for storage scaling")
		s.recordEvent(ctx, inst, req, "rejected", err)
		return err
	}

	patched, err := s.tryPatchVMCRStorage(ctx, inst, req)
	if err != nil {
		if s.log != nil {
			s.log.Warn("scale_storage_cr_patch_failed",
				zap.String("instance", inst.InstanceName),
				zap.String("namespace", inst.Namespace),
				zap.Error(err))
		}
	} else if patched {
		if s.log != nil {
			s.log.Info("scale_storage_cr_patched",
				zap.String("instance", inst.InstanceName),
				zap.String("namespace", inst.Namespace),
				zap.String("type", inst.InstanceType))
		}
		if uerr := s.updateInstanceSpec(ctx, inst, req); uerr != nil {
			s.recordEvent(ctx, inst, req, "cr_patch", uerr)
			return uerr
		}
		s.recordEvent(ctx, inst, req, "cr_patch", nil)
		return nil
	}

	pvcName := inst.ReleaseName
	if pvcName == "" {
		pvcName = inst.InstanceName
	}
	if rerr := s.k8sClient.ResizePVC(ctx, inst.Namespace, pvcName, req.Storage); rerr != nil {
		wrapped := fmt.Errorf("resize pvc: %w", rerr)
		s.recordEvent(ctx, inst, req, "k8s_native", wrapped)
		return wrapped
	}
	if uerr := s.updateInstanceSpec(ctx, inst, req); uerr != nil {
		s.recordEvent(ctx, inst, req, "k8s_native", uerr)
		return uerr
	}
	s.recordEvent(ctx, inst, req, "k8s_native", nil)
	return nil
}

// tryPatchVMCRStorage patch VMSingle/VLSingle CR 的 spec.storage.resources.requests.storage。
func (s *ScaleService) tryPatchVMCRStorage(ctx context.Context, inst *model.Instance, req *ScaleRequest) (bool, error) {
	if s.k8sClient == nil {
		return false, nil
	}
	resource := vmCRResourceFor(inst.InstanceType)
	if resource == "" {
		return false, nil
	}
	name := inst.ReleaseName
	if name == "" {
		name = inst.InstanceName
	}
	existing, err := s.k8sClient.GetCustomResource(ctx, vmOperatorGroup, vmOperatorVersion, resource, inst.Namespace, name)
	if err != nil {
		return false, fmt.Errorf("get %s/%s: %w", resource, name, err)
	}
	if existing == nil {
		return false, nil
	}
	spec := map[string]interface{}{
		"storage": map[string]interface{}{
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"storage": req.Storage,
				},
			},
		},
	}
	if err := s.k8sClient.PatchCustomResourceSpec(ctx, vmOperatorGroup, vmOperatorVersion, resource, inst.Namespace, name, spec); err != nil {
		return false, fmt.Errorf("patch %s/%s spec.storage: %w", resource, name, err)
	}
	return true, nil
}

func (s *ScaleService) updateInstanceSpec(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	var specMap map[string]interface{}
	if inst.Spec != "" {
		if err := json.Unmarshal([]byte(inst.Spec), &specMap); err != nil {
			specMap = map[string]interface{}{}
		}
	} else {
		specMap = map[string]interface{}{}
	}

	if req.Replicas != nil {
		specMap["replicas"] = *req.Replicas
	}
	if req.CPU != "" {
		specMap["cpu"] = req.CPU
	}
	if req.Memory != "" {
		specMap["memory"] = req.Memory
	}
	if req.Storage != "" {
		specMap["storage"] = req.Storage
	}

	raw, err := json.Marshal(specMap)
	if err != nil {
		return err
	}
	inst.Spec = string(raw)
	return s.instanceRepo.Update(ctx, inst)
}
