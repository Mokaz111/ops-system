package grafana

import "context"

// ImportDashboardJSON 占位：§2.5.5 Dashboard 模板同步（后续 POST /api/dashboards/db）。
func (c *Client) ImportDashboardJSON(ctx context.Context, orgID int64, dashboardJSON []byte) error {
	if !c.Enabled() || orgID <= 0 || len(dashboardJSON) == 0 {
		return nil
	}
	_ = ctx
	return nil
}
