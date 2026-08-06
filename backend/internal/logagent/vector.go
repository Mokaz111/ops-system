package logagent

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
)

// AgentSpec Vector Agent（业务集群）构建参数。
type AgentSpec struct {
	Name         string
	Namespace    string
	KafkaBrokers string
	KafkaTopic   string
	TenantID     string
	ZoneSlug     string
	WorkspaceID  string
	ClusterName  string
	Collect      model.LogsCollectConfig
}

func safeName(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "_", "-")
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "ops-vector-agent"
	}
	if len(out) > 52 {
		return out[:52]
	}
	return out
}

// BuildVectorAgentYAML 生成 Vector Agent DaemonSet（采集 → Kafka，注入 U-001 标签）。
func BuildVectorAgentYAML(spec AgentSpec) string {
	ns := strings.TrimSpace(spec.Namespace)
	if ns == "" {
		ns = "vector"
	}
	name := safeName(spec.Name)
	brokers := strings.TrimSpace(spec.KafkaBrokers)
	topic := strings.TrimSpace(spec.KafkaTopic)
	collect := spec.Collect
	if collect.NamespaceInclude == nil && collect.NamespaceExclude == nil && collect.ExcludePaths == nil {
		collect = model.DefaultLogsCollectConfig()
	}

	vectorCfg := buildVectorConfig(spec, collect, brokers, topic)

	return fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    managed-by: ops-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s
rules:
  - apiGroups: [""]
    resources: ["namespaces", "nodes", "pods"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
  - kind: ServiceAccount
    name: %s
    namespace: %s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
data:
  vector.yaml: |
%s
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: ops-system
spec:
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      serviceAccountName: %s
      containers:
        - name: vector
          image: timberio/vector:0.41.1-alpine
          args: ["--config", "/etc/vector/vector.yaml"]
          volumeMounts:
            - name: config
              mountPath: /etc/vector
            - name: data
              mountPath: /vector-data-dir
            - name: varlog
              mountPath: /var/log
              readOnly: true
            - name: varlib
              mountPath: /var/lib
              readOnly: true
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 256Mi
      volumes:
        - name: config
          configMap:
            name: %s-config
        - name: data
          emptyDir: {}
        - name: varlog
          hostPath:
            path: /var/log
        - name: varlib
          hostPath:
            path: /var/lib
`, ns, name, ns, name, name, name, name, ns, name, ns, indentYAML(vectorCfg, 4), name, ns, name, name, name, name, name)
}

func buildVectorConfig(spec AgentSpec, collect model.LogsCollectConfig, brokers, topic string) string {
	var b strings.Builder
	b.WriteString("data_dir: /vector-data-dir\n")
	b.WriteString("sources:\n")
	b.WriteString("  k8s_logs:\n")
	b.WriteString("    type: kubernetes_logs\n")
	if len(collect.ExcludePaths) > 0 {
		b.WriteString("    exclude_paths_glob_patterns:\n")
		for _, p := range collect.ExcludePaths {
			fmt.Fprintf(&b, "      - %q\n", p)
		}
	}

	remapInput := "k8s_logs"
	b.WriteString("transforms:\n")
	if len(collect.NamespaceInclude) > 0 || len(collect.NamespaceExclude) > 0 {
		b.WriteString("  filter_ns:\n")
		b.WriteString("    type: filter\n")
		b.WriteString("    inputs: [k8s_logs]\n")
		b.WriteString("    condition: |\n")
		fmt.Fprintf(&b, "      %s\n", buildNSFilterCondition(collect))
		remapInput = "filter_ns"
	}
	b.WriteString("  inject_ops_labels:\n")
	b.WriteString("    type: remap\n")
	fmt.Fprintf(&b, "    inputs: [%s]\n", remapInput)
	fmt.Fprintf(&b, `    source: |
      .ops_tenant_id = %q
      .ops_zone = %q
      .ops_workspace = %q
      .ops_cluster = %q
sinks:
  kafka_out:
    type: kafka
    inputs: [inject_ops_labels]
    bootstrap_servers: %q
    topic: %q
    key_field: ops_tenant_id
    encoding:
      codec: json
`, spec.TenantID, spec.ZoneSlug, spec.WorkspaceID, spec.ClusterName, brokers, topic)

	return b.String()
}

func buildNSFilterCondition(collect model.LogsCollectConfig) string {
	// Vector VRL condition；namespace 取自 kubernetes.pod_namespace。
	parts := make([]string, 0, 2)
	if len(collect.NamespaceInclude) > 0 {
		parts = append(parts, fmt.Sprintf(`includes([%s], .kubernetes.pod_namespace)`, quoteList(collect.NamespaceInclude)))
	}
	if len(collect.NamespaceExclude) > 0 {
		parts = append(parts, fmt.Sprintf(`!includes([%s], .kubernetes.pod_namespace)`, quoteList(collect.NamespaceExclude)))
	}
	if len(parts) == 0 {
		return "true"
	}
	return strings.Join(parts, " && ")
}

func quoteList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, ", ")
}

func indentYAML(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// VectorAgentClient 下发 Vector Agent 到业务集群。
type VectorAgentClient struct{}

func NewVectorAgentClient() *VectorAgentClient { return &VectorAgentClient{} }

func (c *VectorAgentClient) Apply(ctx context.Context, client *k8s.Client, spec AgentSpec) error {
	if client == nil {
		return fmt.Errorf("k8s client is nil")
	}
	ns := spec.Namespace
	if ns == "" {
		ns = "vector"
	}
	_, err := client.ApplyYAML(ctx, BuildVectorAgentYAML(spec), ns)
	return err
}

func (c *VectorAgentClient) Delete(ctx context.Context, client *k8s.Client, name, namespace string) error {
	if client == nil {
		return fmt.Errorf("k8s client is nil")
	}
	ns := namespace
	if ns == "" {
		ns = "vector"
	}
	n := safeName(name)
	if err := client.DeleteByGVK(ctx, "apps/v1", "DaemonSet", ns, n); err != nil {
		return err
	}
	_ = client.DeleteByGVK(ctx, "v1", "ConfigMap", ns, n+"-config")
	_ = client.DeleteByGVK(ctx, "v1", "ServiceAccount", ns, n)
	_ = client.DeleteByGVK(ctx, "rbac.authorization.k8s.io/v1", "ClusterRole", "", n)
	_ = client.DeleteByGVK(ctx, "rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "", n)
	return nil
}
