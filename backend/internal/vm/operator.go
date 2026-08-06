package vm

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
)

type VMOperatorClient struct {
	k8s *k8s.Client
	cfg *config.VMConfig
}

func NewVMOperatorClient(k8sClient *k8s.Client, cfg *config.VMConfig) *VMOperatorClient {
	return &VMOperatorClient{k8s: k8sClient, cfg: cfg}
}

func (c *VMOperatorClient) Enabled() bool {
	return c != nil && c.k8s != nil
}

func (c *VMOperatorClient) ApplyWorkspaceUser(ctx context.Context, t *model.Workspace, routes RouteSet, poolTargetURL string) error {
	if !c.Enabled() || t == nil {
		return nil
	}
	ns := routes.Namespace
	if ns == "" {
		ns = PoolTargetFromConfig(c.cfg).Namespace
	}
	targetURL := ResolvePoolTarget(poolTargetURL, c.cfg)
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
        url: %s
      paths:
        - /select/%s/prometheus
        - /insert/%s
`, safeName(t.VMUserID)+"-auth", ns, t.ID.String(), t.VMUserID, t.VMUserKey,
		safeName(t.VMUserID), ns, t.ID.String(), t.VMUserID, safeName(t.VMUserID)+"-auth",
		targetURL, t.VMUserID, t.VMUserID)
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
