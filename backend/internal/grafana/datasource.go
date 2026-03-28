package grafana

import (
	"context"
	"strings"

	"ops-system/backend/internal/model"
)

// CreatePrometheusDatasource 在指定组织下创建 Prometheus 数据源。
func (c *Client) CreatePrometheusDatasource(ctx context.Context, orgID int64, t *model.Tenant) error {
	if !c.Enabled() || c.cfg == nil || orgID <= 0 || t == nil {
		return nil
	}
	url := strings.TrimSpace(c.cfg.PrometheusDatasourceURL)
	if url == "" {
		return nil
	}
	body := map[string]any{
		"name":      "vm-" + t.VMUserID,
		"type":      "prometheus",
		"url":       url,
		"access":    "proxy",
		"isDefault": true,
		"jsonData": map[string]any{
			"timeInterval": "30s",
		},
	}
	return c.doJSON(ctx, "POST", "/api/datasources", body, orgID, nil)
}
