package integration

import (
	"strings"
	"testing"
)

func TestRenderNodeExporter(t *testing.T) {
	seed := SeedTemplates()[0]
	r := NewRenderer()
	out, err := r.Render(RenderInput{
		Spec: seed.Spec,
		Values: map[string]string{
			"namespace":       "monitoring",
			"scrape_interval": "15s",
			"cpu_threshold":   "90",
		},
		Ctx: RenderContext{InstanceName: "vm-inst-1", TenantID: "t1"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(out) < 3 {
		t.Fatalf("expected at least 3 resources (collector + vmrule + dashboard), got %d", len(out))
	}
	var hasCollector, hasVMRule, hasDashboard bool
	for _, r := range out {
		switch r.Part {
		case "collector":
			hasCollector = true
			if !strings.Contains(r.YAML, "namespace: monitoring") {
				t.Errorf("collector not rendered with namespace: %q", r.YAML)
			}
			if !strings.Contains(r.YAML, "interval: 15s") {
				t.Errorf("collector not rendered with scrape_interval: %q", r.YAML)
			}
		case "vmrule":
			hasVMRule = true
			if !strings.Contains(r.YAML, "> 90") {
				t.Errorf("vmrule not rendered with cpu_threshold: %q", r.YAML)
			}
			if !strings.Contains(r.YAML, "{{ $labels.instance }}") {
				t.Errorf("vmrule lost label placeholder: %q", r.YAML)
			}
		case "dashboard":
			hasDashboard = true
			if !strings.Contains(r.Dashboard, "node_cpu_seconds_total") {
				t.Errorf("dashboard missing expected metric: %q", r.Dashboard)
			}
		}
	}
	if !hasCollector || !hasVMRule || !hasDashboard {
		t.Fatalf("rendered parts incomplete: collector=%v vmrule=%v dashboard=%v", hasCollector, hasVMRule, hasDashboard)
	}
}

func TestExtractFromSpec(t *testing.T) {
	seed := SeedTemplates()[0]
	extracted := ExtractFromSpec(seed.Spec)
	if _, ok := extracted["node_cpu_seconds_total"]; !ok {
		t.Fatalf("expected node_cpu_seconds_total to be extracted, got %+v", keysOf(extracted))
	}
	m := extracted["node_cpu_seconds_total"]
	if !m.AppearsInCollector {
		t.Errorf("should appear in collector: %+v", m)
	}
	if !m.AppearsInAlert {
		t.Errorf("should appear in alert: %+v", m)
	}
	if !m.AppearsInDashboard {
		t.Errorf("should appear in dashboard: %+v", m)
	}
}

func keysOf(m map[string]*ExtractedMetric) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
