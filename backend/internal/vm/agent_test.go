package vm

import (
	"strings"
	"testing"

	"ops-system/backend/internal/model"
)

func TestBuildVMAgentYAMLCollectConfig(t *testing.T) {
	sel := false
	yaml := BuildVMAgentYAML(AgentSpec{
		Name:           "ops-abc",
		Namespace:      "vmagent",
		RemoteWriteURL: "http://vminsert/insert/1/prometheus",
		BasicAuthUser:  "u",
		BasicAuthPass:  "p",
		TenantID:       "tid",
		ZoneSlug:       "z1",
		WorkspaceID:    "wid",
		ClusterName:    "biz",
		Collect: model.MetricsCollectConfig{
			SelectAllByDefault: &sel,
			ScrapeInterval:     "15s",
			ScrapeTimeout:      "5s",
			NamespaceInclude:   []string{"app", "prod"},
			NamespaceExclude:   []string{"kube-system"},
		},
	})
	for _, want := range []string{
		"selectAllByDefault: false",
		`scrapeInterval: "15s"`,
		`scrapeTimeout: "5s"`,
		"matchNames:",
		`- "app"`,
		`- "prod"`,
		"action: drop",
		`regex: "kube-system"`,
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("yaml missing %q\n%s", want, yaml)
		}
	}
}
