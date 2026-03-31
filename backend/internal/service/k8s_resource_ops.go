package service

import (
	"context"
	"errors"
	"fmt"

	"ops-system/backend/internal/k8s"

	"go.uber.org/zap"
)

var ErrK8sOperatorNotConfigured = errors.New("k8s resource operator is not configured")

// K8sResourceOperator 统一封装平台对 k8s 资源的读写能力。
// 后续 VictoriaLogs / Tracing / 其他模块复用同一入口。
type K8sResourceOperator struct {
	client *k8s.Client
	log    *zap.Logger
}

func NewK8sResourceOperator(client *k8s.Client, log *zap.Logger) *K8sResourceOperator {
	return &K8sResourceOperator{client: client, log: log}
}

func (o *K8sResourceOperator) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	if o.client == nil {
		return ErrK8sOperatorNotConfigured
	}
	if err := o.client.ScaleDeployment(ctx, namespace, name, replicas); err != nil {
		return fmt.Errorf("scale deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

func (o *K8sResourceOperator) ResizePVC(ctx context.Context, namespace, name, size string) error {
	if o.client == nil {
		return ErrK8sOperatorNotConfigured
	}
	if err := o.client.ResizePVC(ctx, namespace, name, size); err != nil {
		return fmt.Errorf("resize pvc %s/%s: %w", namespace, name, err)
	}
	return nil
}

func (o *K8sResourceOperator) PatchCustomResourceSpec(
	ctx context.Context,
	group, version, resource, namespace, name string,
	spec map[string]interface{},
) error {
	if o.client == nil {
		return ErrK8sOperatorNotConfigured
	}
	if err := o.client.PatchCustomResourceSpec(ctx, group, version, resource, namespace, name, spec); err != nil {
		return fmt.Errorf("patch custom resource %s/%s: %w", namespace, name, err)
	}
	return nil
}
