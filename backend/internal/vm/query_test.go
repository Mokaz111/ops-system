package vm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
)

func TestQueryClientScalarUsesTenantSelectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		user, pass, ok := r.BasicAuth()
		if !ok || user != "vmuser-test" || pass != "secret" {
			t.Fatalf("missing tenant basic auth")
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"42"]}]}}`))
	}))
	defer srv.Close()

	client := NewQueryClient(&config.VMConfig{VMAuthBaseURL: srv.URL})
	tenant := &model.Tenant{VMUserID: "vmuser-test", VMUserKey: "secret"}
	value, err := client.Scalar(context.Background(), tenant, "up")
	if err != nil {
		t.Fatalf("scalar query failed: %v", err)
	}
	if value != 42 {
		t.Fatalf("unexpected scalar value: %v", value)
	}
	if gotPath != "/select/vmuser-test/prometheus/api/v1/query" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}
