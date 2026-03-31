package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidPlatformScope   = errors.New("invalid platform scale scope")
	ErrPlatformTargetRequired = errors.New("target_id is required")
	ErrPlatformTargetNotFound = errors.New("platform scale target not found")
	ErrPlatformScaleNoop      = errors.New("at least one scaling field is required")
	ErrInvalidReplicas        = errors.New("replicas must be greater than 0")
	ErrInvalidStorageSize     = errors.New("storage_size must match ^[1-9][0-9]*(Gi|Ti)$")
)

var storageSizePattern = regexp.MustCompile(`^[1-9][0-9]*(Gi|Ti)$`)

type PlatformScaleScope string

const (
	PlatformScopeSharedMetrics   PlatformScaleScope = "shared_metrics"
	PlatformScopeDedicatedMetric PlatformScaleScope = "dedicated_metrics"
)

// ScaleVMClusterRequest 平台级 VMCluster 扩容请求。
// 统一由平台管理员触发，不暴露给租户级用户。
type ScaleVMClusterRequest struct {
	TargetID string
	DryRun   bool

	VMSelectReplicas  *int32
	VMInsertReplicas  *int32
	VMStorageReplicas *int32
	StorageSize       string
}

// ScaleVMClusterPlan 返回本次变更的目标与 spec patch 内容。
type ScaleVMClusterPlan struct {
	TargetID  string                 `json:"target_id"`
	Scope     PlatformScaleScope     `json:"scope"`
	Namespace string                 `json:"namespace"`
	Name      string                 `json:"name"`
	DryRun    bool                   `json:"dry_run"`
	Resource  string                 `json:"resource"`
	SpecPatch map[string]interface{} `json:"spec_patch"`
}

type VMClusterScaleTarget struct {
	ID          string             `json:"id"`
	Scope       PlatformScaleScope `json:"scope"`
	Namespace   string             `json:"namespace"`
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name"`
}

// PlatformScaleService 平台级扩容入口（共享集群/独享集群）。
type PlatformScaleService struct {
	k8sOps  *K8sResourceOperator
	targets map[string]VMClusterScaleTarget
}

func NewPlatformScaleService(k8sOps *K8sResourceOperator) *PlatformScaleService {
	targets := map[string]VMClusterScaleTarget{
		"shared-metrics-main": {
			ID:          "shared-metrics-main",
			Scope:       PlatformScopeSharedMetrics,
			Namespace:   "monitoring",
			Name:        "vmcluster-shared",
			DisplayName: "共享监控主集群",
		},
		"dedicated-metrics-pool": {
			ID:          "dedicated-metrics-pool",
			Scope:       PlatformScopeDedicatedMetric,
			Namespace:   "monitoring",
			Name:        "vmcluster-dedicated",
			DisplayName: "独享监控资源池集群",
		},
	}
	return &PlatformScaleService{k8sOps: k8sOps, targets: targets}
}

func (s *PlatformScaleService) ListVMClusterTargets() []VMClusterScaleTarget {
	orderedIDs := []string{"shared-metrics-main", "dedicated-metrics-pool"}
	out := make([]VMClusterScaleTarget, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if t, ok := s.targets[id]; ok {
			out = append(out, t)
		}
	}
	return out
}

// ScaleVMCluster 修改 VMCluster CR 的 spec。
// 统一通过 k8s 资源操作封装，便于后续 logs/tracing 复用。
func (s *PlatformScaleService) ScaleVMCluster(ctx context.Context, req *ScaleVMClusterRequest) (*ScaleVMClusterPlan, error) {
	if req.TargetID == "" {
		return nil, ErrPlatformTargetRequired
	}
	target, ok := s.targets[req.TargetID]
	if !ok {
		return nil, ErrPlatformTargetNotFound
	}
	if target.Scope != PlatformScopeSharedMetrics && target.Scope != PlatformScopeDedicatedMetric {
		return nil, ErrInvalidPlatformScope
	}
	if err := validatePlatformScaleRequest(req); err != nil {
		return nil, err
	}

	spec := map[string]interface{}{}
	if req.VMSelectReplicas != nil {
		spec["vmselect"] = map[string]interface{}{
			"replicaCount": *req.VMSelectReplicas,
		}
	}
	if req.VMInsertReplicas != nil {
		spec["vminsert"] = map[string]interface{}{
			"replicaCount": *req.VMInsertReplicas,
		}
	}
	if req.VMStorageReplicas != nil || req.StorageSize != "" {
		vmstorage := map[string]interface{}{}
		if req.VMStorageReplicas != nil {
			vmstorage["replicaCount"] = *req.VMStorageReplicas
		}
		if req.StorageSize != "" {
			vmstorage["storage"] = map[string]interface{}{
				"volumeClaimTemplate": map[string]interface{}{
					"spec": map[string]interface{}{
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": req.StorageSize,
							},
						},
					},
				},
			}
		}
		spec["vmstorage"] = vmstorage
	}
	plan := &ScaleVMClusterPlan{
		TargetID:  target.ID,
		Scope:     target.Scope,
		Namespace: target.Namespace,
		Name:      target.Name,
		DryRun:    req.DryRun,
		Resource:  "vmclusters.operator.victoriametrics.com",
		SpecPatch: spec,
	}
	if req.DryRun {
		return plan, nil
	}

	if err := s.k8sOps.PatchCustomResourceSpec(
		ctx,
		"operator.victoriametrics.com",
		"v1beta1",
		"vmclusters",
		target.Namespace,
		target.Name,
		spec,
	); err != nil {
		return nil, err
	}
	return plan, nil
}

func validatePlatformScaleRequest(req *ScaleVMClusterRequest) error {
	if req.VMSelectReplicas == nil &&
		req.VMInsertReplicas == nil &&
		req.VMStorageReplicas == nil &&
		strings.TrimSpace(req.StorageSize) == "" {
		return ErrPlatformScaleNoop
	}
	if req.VMSelectReplicas != nil && *req.VMSelectReplicas < 1 {
		return ErrInvalidReplicas
	}
	if req.VMInsertReplicas != nil && *req.VMInsertReplicas < 1 {
		return ErrInvalidReplicas
	}
	if req.VMStorageReplicas != nil && *req.VMStorageReplicas < 1 {
		return ErrInvalidReplicas
	}
	if req.StorageSize != "" {
		if !storageSizePattern.MatchString(strings.TrimSpace(req.StorageSize)) {
			return ErrInvalidStorageSize
		}
	}
	return nil
}
