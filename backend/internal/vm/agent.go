package vm

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/k8s"
)

// AgentSpec VMAgent CR 构建参数。
type AgentSpec struct {
	Name           string
	Namespace      string
	RemoteWriteURL string
	BasicAuthUser  string
	BasicAuthPass  string
	TenantID       string
	ZoneSlug       string
	WorkspaceID    string
	ClusterName    string
}

// BuildVMAgentYAML 构建带 U-001 标签的 VMAgent CR。
func BuildVMAgentYAML(spec AgentSpec) string {
	ns := strings.TrimSpace(spec.Namespace)
	if ns == "" {
		ns = "vmagent"
	}
	name := safeName(spec.Name)
	if name == "" {
		name = "ops-vmagent"
	}
	return fmt.Sprintf(`apiVersion: operator.victoriametrics.com/v1beta1
kind: VMAgent
metadata:
  name: %s
  namespace: %s
  labels:
    managed-by: ops-system
    ops-system/tenant-id: %s
spec:
  selectAllByDefault: true
  remoteWrite:
    - url: %s
      basicAuth:
        username:
          name: %s-auth
          key: username
        password:
          name: %s-auth
          key: password
  inlineRelabelConfig:
    - target_label: ops_tenant_id
      replacement: %s
    - target_label: ops_zone
      replacement: %s
    - target_label: ops_workspace
      replacement: %s
    - target_label: ops_cluster
      replacement: %s
---
apiVersion: v1
kind: Secret
metadata:
  name: %s-auth
  namespace: %s
type: Opaque
stringData:
  username: %s
  password: %s
`, name, ns, spec.TenantID,
		spec.RemoteWriteURL, name, name,
		spec.TenantID, spec.ZoneSlug, spec.WorkspaceID, spec.ClusterName,
		name, ns, spec.BasicAuthUser, spec.BasicAuthPass)
}

func (c *VMOperatorClient) ApplyVMAgent(ctx context.Context, client *k8s.Client, spec AgentSpec) error {
	if client == nil {
		return fmt.Errorf("k8s client is nil")
	}
	yaml := BuildVMAgentYAML(spec)
	ns := spec.Namespace
	if ns == "" {
		ns = "vmagent"
	}
	_, err := client.ApplyYAML(ctx, yaml, ns)
	return err
}

func (c *VMOperatorClient) DeleteVMAgent(ctx context.Context, client *k8s.Client, name, namespace string) error {
	if client == nil {
		return fmt.Errorf("k8s client is nil")
	}
	ns := namespace
	if ns == "" {
		ns = "vmagent"
	}
	if err := client.DeleteByGVK(ctx, "operator.victoriametrics.com/v1beta1", "VMAgent", ns, safeName(name)); err != nil {
		return err
	}
	_ = client.DeleteByGVK(ctx, "v1", "Secret", ns, safeName(name)+"-auth")
	return nil
}

// HasVMAgentCRD 检查业务集群是否安装 VM Operator VMAgent CRD。
func HasVMAgentCRD(ctx context.Context, client *k8s.Client) bool {
	if client == nil {
		return false
	}
	return client.HasCRD(ctx, "vmagents.operator.victoriametrics.com")
}
