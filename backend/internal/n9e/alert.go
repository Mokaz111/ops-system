package n9e

import (
	"context"
)

// UpsertAlertRule 告警规则同步占位（§2.4.5，后续与 model.AlertRule 对齐）。
func (c *Client) UpsertAlertRule(ctx context.Context, teamID int64, payload map[string]any) error {
	if !c.Enabled() || len(payload) == 0 {
		return nil
	}
	if teamID > 0 {
		payload["team_id"] = teamID
	}
	return c.doJSON(ctx, "POST", c.prefix+"/alert-rules", payload, nil)
}
