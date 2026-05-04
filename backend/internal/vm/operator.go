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

func (c *VMOperatorClient) ApplyTenantUser(ctx context.Context, t *model.Tenant, routes RouteSet) error {
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

func quoteYAML(s string) string {
	return fmt.Sprintf("%q", s)
}
