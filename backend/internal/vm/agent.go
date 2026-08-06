package vm

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
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
	Collect        model.MetricsCollectConfig
}

// BuildVMAgentYAML 构建带 U-001 标签与可配置采集参数的 VMAgent CR。
func BuildVMAgentYAML(spec AgentSpec) string {
	ns := strings.TrimSpace(spec.Namespace)
	if ns == "" {
		ns = "vmagent"
	}
	name := safeName(spec.Name)
	if name == "" {
		name = "ops-vmagent"
	}

	collect := spec.Collect
	if collect.SelectAllByDefault == nil && collect.ScrapeInterval == "" {
		collect = model.DefaultMetricsCollectConfig()
	} else {
		// 补默认值
		def := model.DefaultMetricsCollectConfig()
		if collect.SelectAllByDefault == nil {
			collect.SelectAllByDefault = def.SelectAllByDefault
		}
		if strings.TrimSpace(collect.ScrapeInterval) == "" {
			collect.ScrapeInterval = def.ScrapeInterval
		}
		if strings.TrimSpace(collect.ScrapeTimeout) == "" {
			collect.ScrapeTimeout = def.ScrapeTimeout
		}
	}
	selectAll := true
	if collect.SelectAllByDefault != nil {
		selectAll = *collect.SelectAllByDefault
	}

	var b strings.Builder
	fmt.Fprintf(&b, `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMAgent
metadata:
  name: %s
  namespace: %s
  labels:
    managed-by: ops-system
    ops-system/tenant-id: %s
spec:
  selectAllByDefault: %t
  scrapeInterval: %q
  scrapeTimeout: %q
`, name, ns, spec.TenantID, selectAll, collect.ScrapeInterval, collect.ScrapeTimeout)

	if len(collect.NamespaceInclude) > 0 {
		b.WriteString("  namespaceSelector:\n    matchNames:\n")
		for _, n := range collect.NamespaceInclude {
			fmt.Fprintf(&b, "      - %q\n", n)
		}
	}

	b.WriteString("  remoteWrite:\n")
	fmt.Fprintf(&b, `    - url: %s
      basicAuth:
        username:
          name: %s-auth
          key: username
        password:
          name: %s-auth
          key: password
`, spec.RemoteWriteURL, name, name)

	b.WriteString("  inlineRelabelConfig:\n")
	fmt.Fprintf(&b, `    - target_label: ops_tenant_id
      replacement: %q
    - target_label: ops_zone
      replacement: %q
    - target_label: ops_workspace
      replacement: %q
    - target_label: ops_cluster
      replacement: %q
`, spec.TenantID, spec.ZoneSlug, spec.WorkspaceID, spec.ClusterName)

	if len(collect.NamespaceExclude) > 0 {
		regex := strings.Join(escapeRegexAlternation(collect.NamespaceExclude), "|")
		fmt.Fprintf(&b, `    - action: drop
      source_labels: [namespace]
      regex: %q
`, regex)
	}

	fmt.Fprintf(&b, `---
apiVersion: v1
kind: Secret
metadata:
  name: %s-auth
  namespace: %s
type: Opaque
stringData:
  username: %s
  password: %s
`, name, ns, spec.BasicAuthUser, spec.BasicAuthPass)

	return b.String()
}

func escapeRegexAlternation(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// 简单转义正则元字符，命名空间通常是 DNS label。
		r := strings.NewReplacer(
			`.`, `\.`, `+`, `\+`, `*`, `\*`, `?`, `\?`,
			`(`, `\(`, `)`, `\)`, `[`, `\[`, `]`, `\]`,
			`{`, `\{`, `}`, `\}`, `|`, `\|`, `^`, `\^`, `$`, `\$`,
		)
		out = append(out, r.Replace(item))
	}
	return out
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
