package n9e

import (
	"context"
	"fmt"

	"ops-system/backend/internal/model"

	"go.uber.org/zap"
)

// SyncTenantOnCreate 创建 N9E 团队、同步管理员用户、注册数据源；成功时写入 t.N9ETeamID。
func (c *Client) SyncTenantOnCreate(ctx context.Context, t *model.Tenant) error {
	if c == nil || !c.Enabled() || t == nil {
		return nil
	}
	teamID, err := c.CreateTeam(ctx, t.TenantName, "platform tenant "+t.ID.String())
	if err != nil {
		c.log.Warn("n9e_create_team_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
		return err
	}
	t.N9ETeamID = teamID

	adminUser := &N9EUser{
		Username: fmt.Sprintf("tenant_%s", t.VMUserID),
		Password: t.VMUserKey[:16],
		Email:    fmt.Sprintf("%s@ops.internal", t.VMUserID),
	}
	if err := c.CreateUser(ctx, adminUser); err != nil {
		c.log.Warn("n9e_create_user_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
	}

	if err := c.CreatePrometheusDatasource(ctx, t); err != nil {
		c.log.Warn("n9e_create_datasource_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
	}
	return nil
}

// SyncTenantOnDelete 删除 N9E 团队（幂等）。
func (c *Client) SyncTenantOnDelete(ctx context.Context, t *model.Tenant) {
	if c == nil || !c.Enabled() || t == nil || t.N9ETeamID <= 0 {
		return
	}
	if err := c.DeleteTeam(ctx, t.N9ETeamID); err != nil {
		c.log.Warn("n9e_delete_team_failed", zap.String("tenant_id", t.ID.String()), zap.Int64("n9e_team_id", t.N9ETeamID), zap.Error(err))
	}
}
