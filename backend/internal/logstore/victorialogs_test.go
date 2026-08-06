package logstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildLogsQL_TenantFilter(t *testing.T) {
	q := buildLogsQL(QueryRequest{
		TenantFilter: TenantFilter{TenantID: "550e8400-e29b-41d4-a716-446655440000"},
		RawQuery:     "error",
	})
	if !strings.Contains(q, `ops_tenant_id:"550e8400-e29b-41d4-a716-446655440000"`) {
		t.Fatalf("missing tenant filter: %s", q)
	}
	if !strings.Contains(q, " AND (error)") {
		t.Fatalf("missing raw query wrap: %s", q)
	}
}

func TestVictoriaLogsStore_QueryForcesTenantFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/stream+json")
		_, _ = w.Write([]byte(`{"_time":"2026-08-06T00:00:00Z","_msg":"hello","level":"info"}` + "\n"))
	}))
	defer srv.Close()

	store, err := NewVictoriaLogsStore(Config{SelectURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	res, err := store.Query(context.Background(), QueryRequest{
		TenantFilter: TenantFilter{TenantID: "tenant-a"},
		RawQuery:     "*",
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, `ops_tenant_id:"tenant-a"`) {
		t.Fatalf("query missing tenant filter: %s", gotQuery)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(res.Entries))
	}
	if res.Entries[0].Message != "hello" {
		t.Fatalf("unexpected message: %s", res.Entries[0].Message)
	}
}

func TestNormalizeLimit(t *testing.T) {
	if normalizeLimit(0) != defaultQueryLimit {
		t.Fatal("expected default limit")
	}
	if normalizeLimit(5000) != maxQueryLimit {
		t.Fatal("expected max limit cap")
	}
}
