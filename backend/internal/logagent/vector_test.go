package logagent

import (
	"strings"
	"testing"

	"ops-system/backend/internal/model"
)

func TestBuildVectorAgentYAMLCollectConfig(t *testing.T) {
	yaml := BuildVectorAgentYAML(AgentSpec{
		Name:         "log-abc",
		Namespace:    "vector",
		KafkaBrokers: "kafka:9092",
		KafkaTopic:   "logs",
		TenantID:     "tid",
		ZoneSlug:     "z1",
		WorkspaceID:  "lid",
		ClusterName:  "biz",
		Collect: model.LogsCollectConfig{
			NamespaceInclude: []string{"app"},
			NamespaceExclude: []string{"kube-system"},
			ExcludePaths:     []string{"**/tmp/**"},
		},
	})
	for _, want := range []string{
		"exclude_paths_glob_patterns:",
		`"**/tmp/**"`,
		"filter_ns:",
		`includes(["app"], .kubernetes.pod_namespace)`,
		`!includes(["kube-system"], .kubernetes.pod_namespace)`,
		"inject_ops_labels:",
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("yaml missing %q\n%s", want, yaml)
		}
	}
}
