package grafana

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// ImportDashboardJSON 将 JSON 格式的 Dashboard 导入到指定 Grafana 组织。
func (c *Client) ImportDashboardJSON(ctx context.Context, orgID int64, dashboardJSON []byte) error {
	if !c.Enabled() || orgID <= 0 || len(dashboardJSON) == 0 {
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(dashboardJSON, &raw); err != nil {
		return fmt.Errorf("invalid dashboard json: %w", err)
	}

	body := map[string]interface{}{
		"dashboard": raw,
		"overwrite": true,
		"message":   "imported by ops-system",
	}

	if err := c.doJSON(ctx, "POST", "/api/dashboards/db", body, orgID, nil); err != nil {
		c.log.Warn("grafana_import_dashboard_failed",
			zap.Int64("org_id", orgID),
			zap.Error(err))
		return err
	}
	c.log.Info("grafana_import_dashboard_ok", zap.Int64("org_id", orgID))
	return nil
}

// ImportDashboardsFromDir 从内嵌或外部目录批量导入 Dashboard（预留扩展点）。
func (c *Client) ImportDashboardsFromDir(ctx context.Context, orgID int64, dashboards [][]byte) error {
	if !c.Enabled() || orgID <= 0 {
		return nil
	}
	for i, d := range dashboards {
		if err := c.ImportDashboardJSON(ctx, orgID, d); err != nil {
			c.log.Warn("grafana_import_dashboard_batch_item_failed",
				zap.Int64("org_id", orgID),
				zap.Int("index", i),
				zap.Error(err))
		}
	}
	return nil
}
