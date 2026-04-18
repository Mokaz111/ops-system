package grafana

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// ImportDashboardResult 返回 Grafana 创建 Dashboard 后的标识，供卸载时使用。
type ImportDashboardResult struct {
	UID     string `json:"uid"`
	URL     string `json:"url"`
	Slug    string `json:"slug"`
	ID      int64  `json:"id"`
	Version int64  `json:"version"`
}

// ImportDashboardJSON 将 JSON 格式的 Dashboard 导入到指定 Grafana 组织。
func (c *Client) ImportDashboardJSON(ctx context.Context, orgID int64, dashboardJSON []byte) error {
	_, err := c.ImportDashboardJSONWithResult(ctx, orgID, dashboardJSON)
	return err
}

// ImportDashboardJSONWithResult 与 ImportDashboardJSON 一致，但返回 Grafana 生成的 UID 等信息。
func (c *Client) ImportDashboardJSONWithResult(ctx context.Context, orgID int64, dashboardJSON []byte) (*ImportDashboardResult, error) {
	if !c.Enabled() || orgID <= 0 || len(dashboardJSON) == 0 {
		return nil, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(dashboardJSON, &raw); err != nil {
		return nil, fmt.Errorf("invalid dashboard json: %w", err)
	}

	body := map[string]interface{}{
		"dashboard": raw,
		"overwrite": true,
		"message":   "imported by ops-system",
	}

	out := &ImportDashboardResult{}
	if err := c.doJSON(ctx, "POST", "/api/dashboards/db", body, orgID, out); err != nil {
		c.log.Warn("grafana_import_dashboard_failed",
			zap.Int64("org_id", orgID),
			zap.Error(err))
		return nil, err
	}
	c.log.Info("grafana_import_dashboard_ok",
		zap.Int64("org_id", orgID),
		zap.String("uid", out.UID))
	return out, nil
}

// DeleteDashboardByUID 根据 UID 删除 Grafana Dashboard；404 视为成功。
func (c *Client) DeleteDashboardByUID(ctx context.Context, orgID int64, uid string) error {
	if !c.Enabled() || orgID <= 0 || uid == "" {
		return nil
	}
	err := c.doJSON(ctx, "DELETE", "/api/dashboards/uid/"+uid, nil, orgID, nil)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		c.log.Warn("grafana_delete_dashboard_failed",
			zap.Int64("org_id", orgID),
			zap.String("uid", uid),
			zap.Error(err))
		return err
	}
	c.log.Info("grafana_delete_dashboard_ok",
		zap.Int64("org_id", orgID),
		zap.String("uid", uid))
	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "http 404") || contains(msg, "not found") || contains(msg, "Not found")
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
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
