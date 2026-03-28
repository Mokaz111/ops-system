package grafana

import (
	"bytes"
	"context"
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
	cfg  *config.GrafanaConfig
	log  *zap.Logger
	http *http.Client
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
	return &Client{
		cfg: cfg,
		log: log,
		http: &http.Client{
			Timeout: time.Duration(sec) * time.Second,
		},
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
