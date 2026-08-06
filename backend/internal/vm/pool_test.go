package vm

import "testing"

func TestBuildSharedPoolEndpoints(t *testing.T) {
	ep := BuildSharedPoolEndpoints("monitoring-cn-north", "vm-shared-stack", "http://vmauth.example")
	if ep.VMAuthURL != "http://vm-shared-stack-victoria-metrics-auth.monitoring-cn-north.svc:8427" {
		t.Fatalf("unexpected vmauth url: %s", ep.VMAuthURL)
	}
	if ep.Namespace != "monitoring-cn-north" {
		t.Fatalf("unexpected namespace: %s", ep.Namespace)
	}
}

func TestResolvePoolTarget(t *testing.T) {
	got := ResolvePoolTarget("http://custom-target", nil)
	if got != "http://custom-target" {
		t.Fatalf("expected custom target, got %s", got)
	}
}
