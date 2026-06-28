package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ops-system/backend/internal/config"

	"go.uber.org/zap"
)

func newTestClient(t *testing.T, baseURL string, status int) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if status == http.StatusNotFound {
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		} else {
			_, _ = w.Write([]byte(`{"message":"internal server error"}`))
		}
	}))
	t.Cleanup(srv.Close)
	cfg := &config.GrafanaConfig{
		Enabled:        true,
		BaseURL:        srv.URL,
		AdminUser:      "admin",
		AdminPassword:  "admin",
		MaxRetries:     0, // 关闭重试，便于断言单次 404 行为
	}
	return NewClient(cfg, zap.NewNop())
}

// DeleteOrg 在 404 时应视为成功（幂等），使租户清理可重试。
func TestDeleteOrgIdempotentOn404(t *testing.T) {
	c := newTestClient(t, "", http.StatusNotFound)
	if err := c.DeleteOrg(context.Background(), 42); err != nil {
		t.Fatalf("DeleteOrg on 404 should be nil, got %v", err)
	}
}

// DeleteDatasource 在 404 时应视为成功（幂等）。
func TestDeleteDatasourceIdempotentOn404(t *testing.T) {
	c := newTestClient(t, "", http.StatusNotFound)
	if err := c.DeleteDatasource(context.Background(), 1, 99); err != nil {
		t.Fatalf("DeleteDatasource on 404 should be nil, got %v", err)
	}
}

// DeleteOrg 在非 404 错误（如 500）时必须返回错误。
func TestDeleteOrgReturnsErrorOn500(t *testing.T) {
	c := newTestClient(t, "", http.StatusInternalServerError)
	if err := c.DeleteOrg(context.Background(), 42); err == nil {
		t.Fatalf("DeleteOrg on 500 should return error")
	}
}
