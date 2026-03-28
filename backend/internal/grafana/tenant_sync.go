package grafana

import (
	"context"
	"strings"

	"ops-system/backend/internal/model"

	"go.uber.org/zap"
)

// SyncTenantOnCreate 创建 Grafana 组织并可选数据源；成功时写入 t.GrafanaOrgID。
func (c *Client) SyncTenantOnCreate(ctx context.Context, t *model.Tenant) error {
	if c == nil || !c.Enabled() || t == nil {
		return nil
	}
	name := t.TenantName
	if c.cfg != nil && strings.TrimSpace(c.cfg.OrgNamePrefix) != "" {
		name = strings.TrimSpace(c.cfg.OrgNamePrefix) + t.VMUserID
	}
	orgID, err := c.CreateOrg(ctx, name)
	if err != nil {
		c.log.Warn("grafana_create_org_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
		return err
	}
	t.GrafanaOrgID = orgID
	if err := c.CreatePrometheusDatasource(ctx, orgID, t); err != nil {
		c.log.Warn("grafana_create_datasource_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
	}
	return nil
}

// SyncTenantOnDelete 删除 Grafana 组织。
func (c *Client) SyncTenantOnDelete(ctx context.Context, t *model.Tenant) {
	if c == nil || !c.Enabled() || t == nil || t.GrafanaOrgID <= 0 {
		return
	}
	if err := c.DeleteOrg(ctx, t.GrafanaOrgID); err != nil {
		c.log.Warn("grafana_delete_org_failed", zap.String("tenant_id", t.ID.String()), zap.Int64("grafana_org_id", t.GrafanaOrgID), zap.Error(err))
	}
}
