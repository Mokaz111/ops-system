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
	ErrScaleInstanceNotFound = errors.New("instance not found for scaling")
	ErrInvalidScaleType      = errors.New("invalid scale_type")
	ErrScaleNotSupported     = errors.New("scale operation not supported for this instance")
)

// ScaleRequest 伸缩请求。
type ScaleRequest struct {
	ScaleType string
	Replicas  *int32
	CPU       string
	Memory    string
	Storage   string
}

// ScaleService 实例伸缩（水平 / 垂直 / 存储）。
type ScaleService struct {
	helmClient   *helm.Client
	k8sClient    *k8s.Client
	instanceRepo *repository.InstanceRepository
	log          *zap.Logger
}

func NewScaleService(
	helmClient *helm.Client,
	k8sClient *k8s.Client,
	instanceRepo *repository.InstanceRepository,
	log *zap.Logger,
) *ScaleService {
	return &ScaleService{
		helmClient:   helmClient,
		k8sClient:    k8sClient,
		instanceRepo: instanceRepo,
		log:          log,
	}
}

// Scale 根据类型执行对应伸缩操作。
func (s *ScaleService) Scale(ctx context.Context, instanceID uuid.UUID, req *ScaleRequest) error {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrScaleInstanceNotFound
	}

	switch req.ScaleType {
	case "horizontal":
		return s.scaleHorizontal(ctx, inst, req)
	case "vertical":
		return s.scaleVertical(ctx, inst, req)
	case "storage":
		return s.scaleStorage(ctx, inst, req)
	default:
		return ErrInvalidScaleType
	}
}

func (s *ScaleService) scaleHorizontal(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if inst.Namespace == "" {
		return ErrScaleNotSupported
	}
	if req.Replicas == nil {
		return fmt.Errorf("replicas required for horizontal scaling")
	}
	deployName := inst.ReleaseName
	if deployName == "" {
		deployName = inst.InstanceName
	}
	if err := s.k8sClient.ScaleDeployment(ctx, inst.Namespace, deployName, *req.Replicas); err != nil {
		return fmt.Errorf("scale deployment: %w", err)
	}
	return s.updateInstanceSpec(ctx, inst, req)
}

func (s *ScaleService) scaleVertical(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if inst.ReleaseName == "" || inst.Namespace == "" {
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

	rs, err := s.helmClient.GetReleaseStatus(ctx, inst.ReleaseName, inst.Namespace)
	if err != nil {
		return fmt.Errorf("get release status: %w", err)
	}
	if err := s.helmClient.UpgradeRelease(ctx, inst.ReleaseName, rs.Chart, inst.Namespace, vals); err != nil {
		return fmt.Errorf("upgrade release: %w", err)
	}
	return s.updateInstanceSpec(ctx, inst, req)
}

func (s *ScaleService) scaleStorage(ctx context.Context, inst *model.Instance, req *ScaleRequest) error {
	if inst.Namespace == "" {
		return ErrScaleNotSupported
	}
	if req.Storage == "" {
		return fmt.Errorf("storage size required for storage scaling")
	}
	pvcName := inst.ReleaseName
	if pvcName == "" {
		pvcName = inst.InstanceName
	}
	if err := s.k8sClient.ResizePVC(ctx, inst.Namespace, pvcName, req.Storage); err != nil {
		return fmt.Errorf("resize pvc: %w", err)
	}
	return s.updateInstanceSpec(ctx, inst, req)
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
