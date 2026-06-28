package handler

import (
	"net/url"
	"strings"
	"testing"
)

func TestStripProxyPrefix(t *testing.T) {
	const prefix = "/api/v1/grafana/proxy"
	cases := []struct {
		in, want string
	}{
		{"/api/v1/grafana/proxy", "/"},
		{"/api/v1/grafana/proxy/", "/"},
		{"/api/v1/grafana/proxy/api/health", "/api/health"},
		{"/api/v1/grafana/proxy/d/dash/x", "/d/dash/x"},
		{"/api/v1/grafana/proxy/api/datasources", "/api/datasources"},
		{"/unrelated/path", "/unrelated/path"},
	}
	for _, c := range cases {
		got := stripProxyPrefix(c.in, prefix)
		if got != c.want {
			t.Errorf("stripProxyPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRewriteGrafanaLocation(t *testing.T) {
	const prefix = "/api/v1/grafana/proxy"
	target, _ := url.Parse("http://grafana-internal:3000")

	t.Run("subpath mode", func(t *testing.T) {
		cases := []struct {
			name string
			loc  string
			want string
		}{
			{"absolute same host", "http://grafana-internal:3000/api/v1/grafana/proxy/login?redirectTo=%2F", "/api/v1/grafana/proxy/login?redirectTo=%2F"},
			{"relative with prefix", "/api/v1/grafana/proxy/d/abc/dashboard", "/api/v1/grafana/proxy/d/abc/dashboard"},
			{"empty", "", ""},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				got := rewriteGrafanaLocation(c.loc, target, prefix, false)
				if got != c.want {
					t.Errorf("rewriteGrafanaLocation(%q) = %q, want %q", c.loc, got, c.want)
				}
			})
		}
	})

	t.Run("external mode adds prefix", func(t *testing.T) {
		cases := []struct {
			loc  string
			want string
		}{
			{"/login", "/api/v1/grafana/proxy/login"},
			{"/d/abc/dashboard", "/api/v1/grafana/proxy/d/abc/dashboard"},
			{"http://grafana-internal:3000/login", "/api/v1/grafana/proxy/login"},
			{"http://localhost:3000/?orgId=2", "/api/v1/grafana/proxy/?orgId=2"},
		}
		for _, c := range cases {
			got := rewriteGrafanaLocation(c.loc, target, prefix, true)
			if got != c.want {
				t.Errorf("rewriteGrafanaLocation(%q) = %q, want %q", c.loc, got, c.want)
			}
		}
	})
}

func TestBuildGrafanaProxyURL(t *testing.T) {
	cases := []struct {
		redirect string
		want     string
	}{
		{"", "/api/v1/grafana/proxy/"},
		{"/d/uid/my-dashboard", "/api/v1/grafana/proxy/d/uid/my-dashboard"},
		{"/explore", "/api/v1/grafana/proxy/explore"},
		{"https://evil.com", "/api/v1/grafana/proxy/"},
		{"no-leading-slash", "/api/v1/grafana/proxy/"},
	}
	for _, c := range cases {
		t.Run(c.redirect, func(t *testing.T) {
			got := buildGrafanaProxyURL(c.redirect)
			if got != c.want {
				t.Errorf("buildGrafanaProxyURL(%q) = %q, want %q", c.redirect, got, c.want)
			}
		})
	}
}

func TestRewriteGrafanaAssetPaths(t *testing.T) {
	in := []byte(`<html><head><base href="/" /><link href="public/build/grafana.css"></head><body><script>window.grafanaBootData={"user":{"analytics":{"identifier":"admin@localhost@http://localhost:3000/"}},"settings":{"appSubUrl":"","appUrl":"http://localhost:3000/"}}</script><script src='/public/build/app.js'></script><img src="/api/v1/grafana/proxy/avatar/a"></body></html>`)
	got := string(rewriteGrafanaAssetPaths(in, "/api/v1/grafana/proxy", "http://192.168.19.135:5175/api/v1/grafana/proxy/"))
	if !strings.Contains(got, `<base href="/api/v1/grafana/proxy/" />`) {
		t.Fatalf("expected base href rewritten once, got: %s", got)
	}
	if strings.Contains(got, `/api/v1/grafana/proxy/api/v1/grafana/proxy/`) {
		t.Fatalf("base href duplicated prefix, got: %s", got)
	}
	if !strings.Contains(got, `/api/v1/grafana/proxy/public/build/grafana.css`) {
		t.Fatalf("expected css path rewritten, got: %s", got)
	}
	if !strings.Contains(got, `/api/v1/grafana/proxy/public/build/app.js`) {
		t.Fatalf("expected js path rewritten, got: %s", got)
	}
	if !strings.Contains(got, `"appSubUrl":"/api/v1/grafana/proxy"`) {
		t.Fatalf("expected appSubUrl rewritten, got: %s", got)
	}
	if !strings.Contains(got, `"appUrl":"http://192.168.19.135:5175/api/v1/grafana/proxy/"`) {
		t.Fatalf("expected appUrl rewritten, got: %s", got)
	}
	if !strings.Contains(got, `admin@localhost@http://192.168.19.135:5175/api/v1/grafana/proxy/`) {
		t.Fatalf("expected analytics identifier rewritten, got: %s", got)
	}
}

func TestRewriteGrafanaFrontendSettingsJSON(t *testing.T) {
	in := []byte(`{"appUrl":"http://localhost:3000/","appSubUrl":""}`)
	got := string(rewriteGrafanaFrontendSettingsJSON(in, "/api/v1/grafana/proxy", "http://192.168.19.135:5175/api/v1/grafana/proxy/"))
	if !strings.Contains(got, `"appUrl":"http://192.168.19.135:5175/api/v1/grafana/proxy/"`) {
		t.Fatalf("expected appUrl rewritten, got: %s", got)
	}
	if !strings.Contains(got, `"appSubUrl":"/api/v1/grafana/proxy"`) {
		t.Fatalf("expected appSubUrl rewritten, got: %s", got)
	}
}
