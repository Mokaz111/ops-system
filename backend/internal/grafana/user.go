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

// RemoveOrgUser DELETE /api/orgs/:orgId/users/:userId
func (c *Client) RemoveOrgUser(ctx context.Context, orgID, userID int64) error {
	if !c.Enabled() || orgID <= 0 || userID <= 0 {
		return nil
	}
	path := fmt.Sprintf("/api/orgs/%d/users/%d", orgID, userID)
	return c.doJSON(ctx, "DELETE", path, nil, 0, nil)
}

// DeleteDatasource DELETE /api/datasources/:id（需在对应组织上下文中）。
func (c *Client) DeleteDatasource(ctx context.Context, orgID, dsID int64) error {
	if !c.Enabled() || dsID <= 0 {
		return nil
	}
	path := fmt.Sprintf("/api/datasources/%d", dsID)
	return c.doJSON(ctx, "DELETE", path, nil, orgID, nil)
}
