package grafana

import (
	"context"
	"fmt"
)

// OrgUser 添加至组织的用户（§2.5.3 占位）。
type OrgUser struct {
	LoginOrEmail string `json:"loginOrEmail"`
	Role         string `json:"role"` // Admin, Editor, Viewer
}

// AddOrgUser POST /api/orgs/:orgId/users
func (c *Client) AddOrgUser(ctx context.Context, orgID int64, u *OrgUser) error {
	if !c.Enabled() || orgID <= 0 || u == nil {
		return nil
	}
	path := fmt.Sprintf("/api/orgs/%d/users", orgID)
	return c.doJSON(ctx, "POST", path, u, 0, nil)
}
