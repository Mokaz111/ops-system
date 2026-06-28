package vm

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
)

type VMOperatorClient struct {
	k8s *k8s.Client
}

func NewVMOperatorClient(k8sClient *k8s.Client) *VMOperatorClient {
	return &VMOperatorClient{k8s: k8sClient}
}

func (c *VMOperatorClient) Enabled() bool {
	return c != nil && c.k8s != nil
}

func (c *VMOperatorClient) ApplyWorkspaceUser(ctx context.Context, t *model.Workspace, routes RouteSet) error {
	if !c.Enabled() || t == nil {
		return nil
	}
	ns := routes.Namespace
	if ns == "" {
		ns = "default"
	}
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  labels:
    ops-system/tenant-id: %s
type: Opaque
stringData:
  username: %s
  password: %s
---
apiVersion: operator.victoriametrics.com/v1beta1
kind: VMUser
metadata:
  name: %s
  namespace: %s
  labels:
    ops-system/tenant-id: %s
spec:
  username: %s
  passwordRef:
    name: %s
    key: password
  targetRefs:
    - static:
        url: http://vmsingle-vmselect.%s.svc:8428
      paths:
        - /select/%s/prometheus
        - /insert/%s
`, safeName(t.VMUserID)+"-auth", ns, t.ID.String(), t.VMUserID, t.VMUserKey,
		safeName(t.VMUserID), ns, t.ID.String(), t.VMUserID, safeName(t.VMUserID)+"-auth", ns, t.VMUserID, t.VMUserID)
	_, err := c.k8s.ApplyYAML(ctx, yaml, ns)
	return err
}

type AlertRuleSpec struct {
	Name        string
	Namespace   string
	TenantID    string
	Expr        string
	For         string
	Severity    string
	Annotations string
	Enabled     bool
}

func (c *VMOperatorClient) ApplyAlertRule(ctx context.Context, spec AlertRuleSpec) error {
	if !c.Enabled() || !spec.Enabled {
		return nil
	}
	if spec.For == "" {
		spec.For = "1m"
	}
	annotations := "summary: " + quoteYAML(spec.Name)
	if strings.TrimSpace(spec.Annotations) != "" {
		annotations = "description: " + quoteYAML(spec.Annotations)
	}
	yaml := fmt.Sprintf(`apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
metadata:
  name: %s
  namespace: %s
  labels:
    ops-system/tenant-id: %s
spec:
  groups:
    - name: %s
      rules:
        - alert: %s
          expr: %s
          for: %s
          labels:
            severity: %s
            tenant_id: %s
          annotations:
            %s
`, safeName(spec.Name), spec.Namespace, spec.TenantID, safeName(spec.Name), safeName(spec.Name),
		quoteYAML(spec.Expr), spec.For, quoteYAML(spec.Severity), quoteYAML(spec.TenantID), annotations)
	_, err := c.k8s.ApplyYAML(ctx, yaml, spec.Namespace)
	return err
}

func safeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	repl := strings.NewReplacer("_", "-", ".", "-", ":", "-", "/", "-")
	s = repl.Replace(s)
	if s == "" {
		return "vm-rule"
	}
	return s
}

// VMClusterSpec 定义要创建的 VMCluster CR 规格。
type VMClusterSpec struct {
	Name              string // CR name
	Namespace         string // 目标命名空间
	TenantID          string
	RetentionPeriod   string // 例: "15d"
	VMInsertReplicas  int32
	VMInsertCPU       string // 例: "2"
	VMInsertMemory    string // 例: "4Gi"
	VMSelectReplicas  int32
	VMSelectCPU       string
	VMSelectMemory    string
	VMStorageReplicas int32
	VMStorageCPU      string
	VMStorageMemory   string
	VMStorageSize     string // 例: "200Gi"
}

// ApplyVMCluster 在目标集群中创建或更新 VMCluster CR。
// Operator 会监听 CR 并自动创建 vminsert/vmselect/vmstorage 组件。
func (c *VMOperatorClient) ApplyVMCluster(ctx context.Context, spec VMClusterSpec) error {
	if !c.Enabled() {
		return nil
	}
	ns := spec.Namespace
	if ns == "" {
		ns = "default"
	}
	retention := spec.RetentionPeriod
	if retention == "" {
		retention = "15d"
	}

	yaml := fmt.Sprintf(`apiVersion: operator.victoriametrics.com/v1beta1
kind: VMCluster
metadata:
  name: %s
  namespace: %s
  labels:
    ops-system/tenant-id: %s
    managed-by: ops-system
spec:
  retentionPeriod: "%s"
  vminsert:
    replicaCount: %d
    resources:
      requests:
        cpu: "%s"
        memory: "%s"
      limits:
        cpu: "%s"
        memory: "%s"
  vmselect:
    replicaCount: %d
    resources:
      requests:
        cpu: "%s"
        memory: "%s"
      limits:
        cpu: "%s"
        memory: "%s"
  vmstorage:
    replicaCount: %d
    storageDataPath: /vm-data
    resources:
      requests:
        cpu: "%s"
        memory: "%s"
      limits:
        cpu: "%s"
        memory: "%s"
    storage:
      volumeClaimTemplate:
        spec:
          resources:
            requests:
              storage: "%s"
`,
		safeName(spec.Name), ns, spec.TenantID,
		retention,
		spec.VMInsertReplicas, spec.VMInsertCPU, spec.VMInsertMemory, spec.VMInsertCPU, spec.VMInsertMemory,
		spec.VMSelectReplicas, spec.VMSelectCPU, spec.VMSelectMemory, spec.VMSelectCPU, spec.VMSelectMemory,
		spec.VMStorageReplicas, spec.VMStorageCPU, spec.VMStorageMemory, spec.VMStorageCPU, spec.VMStorageMemory,
		spec.VMStorageSize,
	)
	_, err := c.k8s.ApplyYAML(ctx, yaml, ns)
	return err
}

// DeleteVMCluster 删除 VMCluster CR；CR 已不存在视为成功（幂等）。
func (c *VMOperatorClient) DeleteVMCluster(ctx context.Context, name, namespace string) error {
	if !c.Enabled() {
		return nil
	}
	ns := namespace
	if ns == "" {
		ns = "default"
	}
	return c.k8s.DeleteByGVK(ctx, "operator.victoriametrics.com/v1beta1", "VMCluster", ns, safeName(name))
}

// DeleteVMUser 删除租户 VMUser CR 及其认证 Secret；资源已不存在视为成功（幂等）。
func (c *VMOperatorClient) DeleteVMUser(ctx context.Context, vmUserID, namespace string) error {
	if !c.Enabled() || vmUserID == "" {
		return nil
	}
	ns := namespace
	if ns == "" {
		ns = "default"
	}
	if err := c.k8s.DeleteByGVK(ctx, "operator.victoriametrics.com/v1beta1", "VMUser", ns, safeName(vmUserID)); err != nil {
		return err
	}
	// best-effort 清理认证 Secret；失败不阻塞（Secret 残留不影响重建）。
	_ = c.k8s.DeleteByGVK(ctx, "v1", "Secret", ns, safeName(vmUserID)+"-auth")
	return nil
}

func quoteYAML(s string) string {
	return fmt.Sprintf("%q", s)
}
