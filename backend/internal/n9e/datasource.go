package n9e

import (
	"context"
	"strings"

	"ops-system/backend/internal/model"
)

// CreatePrometheusDatasource 为租户注册 Prometheus 兼容数据源。
func (c *Client) CreatePrometheusDatasource(ctx context.Context, t *model.Tenant) error {
	if !c.Enabled() || c.cfg == nil {
		return nil
	}
	url := strings.TrimSpace(c.cfg.PrometheusDatasourceURL)
	if url == "" {
		return nil
	}
	body := map[string]any{
		"name":         "vm-" + t.VMUserID,
		"plugin_type":  "prometheus",
		"cluster_name": "default",
		"settings":     map[string]any{},
		"http": map[string]any{
			"url":     url,
			"timeout": 10000,
		},
		"auth": map[string]any{
			"basic_auth": false,
		},
	}
	return c.doJSON(ctx, "POST", c.prefix+"/datasource", body, nil)
}
