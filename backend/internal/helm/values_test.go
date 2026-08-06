package helm

import "testing"

// parseValuesYAML 应对 ${VAR} 占位符做环境变量替换。
func TestParseValuesYAMLEnvSubst(t *testing.T) {
	t.Setenv("TEST_VALUES_SECRET", "supersecret")

	m, err := parseValuesYAML([]byte("auth:\n  password: ${TEST_VALUES_SECRET}\n"))
	if err != nil {
		t.Fatalf("parseValuesYAML: %v", err)
	}
	auth, ok := m["auth"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing auth key")
	}
	if pw, _ := auth["password"].(string); pw != "supersecret" {
		t.Fatalf("password = %q, want supersecret (envsubst not applied)", pw)
	}
}
