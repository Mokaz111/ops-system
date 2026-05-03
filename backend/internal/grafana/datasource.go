package grafana

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/model"
)

// UpdateDatasource 更新指定组织下的数据源配置。
func (c *Client) UpdateDatasource(ctx context.Context, orgID int64, dsID int64, body map[string]any) error {
	if !c.Enabled() || orgID <= 0 || dsID <= 0 {
		return nil
	}
	path := fmt.Sprintf("/api/datasources/%d", dsID)
	return c.doJSON(ctx, "PUT", path, body, orgID, nil)
}

// TestDatasource 测试数据源连通性（需 Basic Auth）。
func (c *Client) TestDatasource(ctx context.Context, orgID int64, body map[string]any) (map[string]any, error) {
	if !c.Enabled() || orgID <= 0 {
		return nil, nil
	}
	var out map[string]any
	if err := c.doJSON(ctx, "POST", "/api/datasources/test", body, orgID, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GrafanaAdminSettings Grafana /api/admin/settings 返回值。
type GrafanaAdminSettings map[string]any

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
