package vm

import (
	"testing"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
)

func TestRouteBuilderBuildTenantRoutes(t *testing.T) {
	builder := NewRouteBuilder(&config.VMConfig{VMAuthBaseURL: "http://vm-auth:8427/"})
	tenant := &model.Tenant{VMUserID: "vmuser-test"}

	routes := builder.BuildTenantRoutes(tenant)
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
