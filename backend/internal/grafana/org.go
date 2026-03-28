package grafana

import (
	"context"
	"fmt"
)

// CreateOrg 创建组织，返回 orgId。
func (c *Client) CreateOrg(ctx context.Context, name string) (int64, error) {
	if !c.Enabled() {
		return 0, fmt.Errorf("grafana disabled")
	}
	body := map[string]string{"name": name}
	var out struct {
		OrgID   int64  `json:"orgId"`
		Message string `json:"message"`
	}
	if err := c.doJSON(ctx, "POST", "/api/orgs", body, 0, &out); err != nil {
		return 0, err
	}
	if out.OrgID <= 0 {
		return 0, fmt.Errorf("grafana create org: missing orgId")
	}
	return out.OrgID, nil
}

// DeleteOrg 删除组织（需 Grafana 服务账号具备权限）。
func (c *Client) DeleteOrg(ctx context.Context, orgID int64) error {
	if !c.Enabled() || orgID <= 0 {
		return nil
	}
	path := fmt.Sprintf("/api/orgs/%d", orgID)
	return c.doJSON(ctx, "DELETE", path, nil, 0, nil)
}
