package grafana

import (
	"context"
	"fmt"
)

// GrafanaPlugin Grafana 插件信息。
type GrafanaPlugin struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	Pinned  bool   `json:"pinned"`
}

// ListPlugins 列出所有已安装插件。
func (c *Client) ListPlugins(ctx context.Context) ([]GrafanaPlugin, error) {
	if !c.Enabled() {
		return nil, nil
	}
	var items []GrafanaPlugin
	if err := c.doJSON(ctx, "GET", "/api/plugins", nil, 0, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// InstallPlugin 安装插件，可选指定版本。
func (c *Client) InstallPlugin(ctx context.Context, pluginID, version string) error {
	if !c.Enabled() || pluginID == "" {
		return nil
	}
	var body map[string]any
	if version != "" {
		body = map[string]any{"version": version}
	}
	return c.doJSON(ctx, "POST", fmt.Sprintf("/api/plugins/%s/install", pluginID), body, 0, nil)
}

// UninstallPlugin 卸载插件。
func (c *Client) UninstallPlugin(ctx context.Context, pluginID string) error {
	if !c.Enabled() || pluginID == "" {
		return nil
	}
	return c.doJSON(ctx, "POST", fmt.Sprintf("/api/plugins/%s/uninstall", pluginID), nil, 0, nil)
}
