package helm

import "testing"

// LoadValuesYAML 应对 ${VAR} 占位符做环境变量替换。
func TestLoadValuesYAMLEnvSubst(t *testing.T) {
	t.Setenv("GRAFANA_ADMIN_PASSWORD", "supersecret")

	m, err := LoadValuesYAML("grafana.yaml")
	if err != nil {
		t.Fatalf("LoadValuesYAML: %v", err)
	}
	grafana, ok := m["grafana"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing grafana key")
	}
	ini, ok := grafana["grafana.ini"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing grafana.ini")
	}
	sec, ok := ini["security"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing security")
	}
	pw, _ := sec["admin_password"].(string)
	if pw != "supersecret" {
		t.Fatalf("admin_password = %q, want supersecret (envsubst not applied)", pw)
	}

	// serve_from_sub_path 应为 true（子路径代理配置）。
	srv, ok := ini["server"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing server")
	}
	if sfp, _ := srv["serve_from_sub_path"].(bool); !sfp {
		t.Fatalf("serve_from_sub_path not enabled")
	}
}
