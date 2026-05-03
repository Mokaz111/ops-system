package grafana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ops-system/backend/internal/config"

	"go.uber.org/zap"
)

// Client Grafana HTTP API（§2.5）。
type Client struct {
	cfg          *config.GrafanaConfig
	log          *zap.Logger
	http         *http.Client
	adminBasicAuth string // cached "user:password" base64
}

// NewClient 创建客户端。
func NewClient(cfg *config.GrafanaConfig, log *zap.Logger) *Client {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg == nil {
		cfg = &config.GrafanaConfig{}
	}
	sec := cfg.HTTPTimeoutSeconds
	if sec <= 0 {
		sec = 30
	}
	var basicAuth string
	if cfg.AdminUser != "" && cfg.AdminPassword != "" {
		basicAuth = base64.StdEncoding.EncodeToString([]byte(cfg.AdminUser + ":" + cfg.AdminPassword))
	}
	return &Client{
		cfg:            cfg,
		log:            log,
		http: &http.Client{
			Timeout: time.Duration(sec) * time.Second,
		},
		adminBasicAuth: basicAuth,
	}
}

// Enabled 需 base_url 与 api_key。
func (c *Client) Enabled() bool {
	if c == nil || c.cfg == nil || !c.cfg.Enabled {
		return false
	}
	return strings.TrimSpace(c.cfg.BaseURL) != "" && strings.TrimSpace(c.cfg.APIKey) != ""
}

func (c *Client) base() string {
	return strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/")
}

// doJSON 调用 Grafana API；orgID>0 时设置 X-Grafana-Org-Id（组织内操作）。
func (c *Client) doJSON(ctx context.Context, method, path string, body any, orgID int64, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("grafana disabled")
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	u := c.base() + path
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.cfg.APIKey))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if orgID > 0 {
		req.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", orgID))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("grafana %s %s http %d: %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil && len(b) > 0 {
		return json.Unmarshal(b, out)
	}
	return nil
}

// DoJSON 公开方法，供 GrafanaService 调用。
func (c *Client) DoJSON(ctx context.Context, method, path string, body any, orgID int64, out any) error {
	return c.doJSON(ctx, method, path, body, orgID, out)
}

// hasAdminAuth 是否已配置 Basic Auth（Admin API 必需）。
func (c *Client) hasAdminAuth() bool {
	return c.adminBasicAuth != ""
}

// doAdminJSON 使用 Basic Auth 调用 Grafana Admin API。
func (c *Client) doAdminJSON(ctx context.Context, method, path string, body any, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("grafana disabled")
	}
	if !c.hasAdminAuth() {
		return fmt.Errorf("grafana admin auth not configured (admin_user + admin_password required)")
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	u := c.base() + path
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Basic "+c.adminBasicAuth)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("grafana admin %s %s http %d: %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil && len(b) > 0 {
		return json.Unmarshal(b, out)
	}
	return nil
}

// GrafanaStats Grafana /api/admin/stats 返回值。
type GrafanaStats struct {
	Users       int64 `json:"users"`
	Orgs        int64 `json:"orgs"`
	Dashboards  int64 `json:"dashboards"`
	Snapshots   int64 `json:"snapshots"`
	Tags        int64 `json:"tags"`
	Datasources int64 `json:"datasources"`
	Playlists   int64 `json:"playlists"`
	Stars       int64 `json:"stars"`
	Alerts      int64 `json:"alerts"`
	ActiveUsers int64 `json:"activeUsers"`
}

// AdminStats 获取 Grafana 全局统计信息（需要 Basic Auth）。
func (c *Client) AdminStats(ctx context.Context) (*GrafanaStats, error) {
	var s GrafanaStats
	if err := c.doAdminJSON(ctx, "GET", "/api/admin/stats", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// AdminSettings 获取 Grafana 服务器配置（需要 Basic Auth）。
func (c *Client) AdminSettings(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doAdminJSON(ctx, "GET", "/api/admin/settings", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GrafanaHealth grafana 健康检查结果。
type GrafanaHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthCheck 检查 Grafana 是否健康。
func (c *Client) HealthCheck(ctx context.Context) (*GrafanaHealth, error) {
	if !c.Enabled() {
		return &GrafanaHealth{Status: "disabled"}, nil
	}
	var h GrafanaHealth
	if err := c.doJSON(ctx, "GET", "/api/health", nil, 0, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
