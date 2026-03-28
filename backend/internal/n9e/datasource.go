package n9e

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/model"
)

// CreatePrometheusDatasource 为租户注册 Prometheus 兼容数据源（指向 VM select 等）。
func (c *Client) CreatePrometheusDatasource(ctx context.Context, t *model.Tenant) error {
	if !c.Enabled() || c.cfg == nil {
		return nil
	}
	url := strings.TrimSpace(c.cfg.PrometheusDatasourceURL)
	if url == "" {
		return nil
	}
	// 夜莺数据源插件一般为 prometheus，config 为 JSON 字符串
	cfgJSON := fmt.Sprintf(`{"url":%q,"timeout":30000,"basic_auth":false}`, url)
	body := map[string]any{
		"name":        "vm-" + t.VMUserID,
		"ident":       t.VMUserID,
		"plugin_type": "prometheus",
		"config":      cfgJSON,
	}
	if t.N9ETeamID > 0 {
		body["team_id"] = t.N9ETeamID
	}
	return c.doJSON(ctx, "POST", c.prefix+"/datasources", body, nil)
}
