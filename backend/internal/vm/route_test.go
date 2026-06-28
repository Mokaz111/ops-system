package vm

import (
	"testing"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
)

func TestRouteBuilderBuildWorkspaceRoutes(t *testing.T) {
	builder := NewRouteBuilder(&config.VMConfig{VMAuthBaseURL: "http://vm-auth:8427/"})
	tenant := &model.Workspace{VMUserID: "vmuser-test"}

	routes := builder.BuildWorkspaceRoutes(tenant)
	if routes.InsertURL != "http://vm-auth:8427/insert/vmuser-test" {
		t.Fatalf("unexpected insert url: %s", routes.InsertURL)
	}
	if routes.SelectURL != "http://vm-auth:8427/select/vmuser-test/prometheus" {
		t.Fatalf("unexpected select url: %s", routes.SelectURL)
	}
	if routes.Namespace == "" {
		t.Fatal("namespace should be generated")
	}
}
