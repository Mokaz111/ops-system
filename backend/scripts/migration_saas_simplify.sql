-- ============================================================================
-- Migration: 纯 SaaS 平台化
-- Date: 2026-06-27
--
-- Summary:
--   1. ops_tenants → ops_workspaces (rename, drop dept_id)
--   2. User.tenant_id → workspace_id
--   3. Backup departments / tenant_members / service_accounts / api_tokens
--   4. ops_grafana_instances: drop scope/tenant_id, add source
--   5. Clean instance_type='visual' records
--   6. User.role migration: platform_admin→admin, operator→user, viewer→user
--
-- !! BACKUP YOUR DATABASE BEFORE RUNNING !!
-- ============================================================================

BEGIN;

-- ── 1. Tenant → Workspace ──────────────────────────────────────────────

ALTER TABLE ops_tenants RENAME TO ops_workspaces;
ALTER TABLE ops_workspaces DROP COLUMN IF EXISTS dept_id;
ALTER TABLE ops_workspaces ADD COLUMN IF NOT EXISTS grafana_org_id BIGINT DEFAULT 0;
COMMENT ON TABLE ops_workspaces IS 'workspace (formerly tenant) — pure SaaS organisation unit';

-- Drop the old partial unique indexes that referenced dept_id
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'uk_tenant_dept_active') THEN
    DROP INDEX uk_tenant_dept_active;
  END IF;
END $$;

-- ── 2. User: tenant_id → workspace_id ──────────────────────────────────

ALTER TABLE ops_users ADD COLUMN IF NOT EXISTS workspace_id UUID;
-- Migrate existing data: copy tenant_id → workspace_id
UPDATE ops_users SET workspace_id = tenant_id WHERE workspace_id IS NULL AND tenant_id IS NOT NULL;
ALTER TABLE ops_users DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE ops_users DROP COLUMN IF EXISTS dept_id;
CREATE INDEX IF NOT EXISTS idx_ops_users_workspace_id ON ops_users(workspace_id);

-- ── 3. Backup deprecated tables ─────────────────────────────────────────

DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ops_departments') THEN
    ALTER TABLE ops_departments RENAME TO ops_departments_deprecated;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ops_tenant_members') THEN
    ALTER TABLE ops_tenant_members RENAME TO ops_tenant_members_deprecated;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ops_service_accounts') THEN
    ALTER TABLE ops_service_accounts RENAME TO ops_service_accounts_deprecated;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ops_api_tokens') THEN
    ALTER TABLE ops_api_tokens RENAME TO ops_api_tokens_deprecated;
  END IF;
END $$;

-- ── 4. Grafana instances: add source, drop scope/tenant_id ─────────────

ALTER TABLE ops_grafana_instances ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'external';
UPDATE ops_grafana_instances SET source = 'external' WHERE source IS NULL;
ALTER TABLE ops_grafana_instances DROP COLUMN IF EXISTS scope;
ALTER TABLE ops_grafana_instances DROP COLUMN IF EXISTS tenant_id;

-- ── 5. Clean visual instances ───────────────────────────────────────────

-- Soft-delete instance_type='visual' records that have no active integrations.
-- Those with active integrations are kept but marked as deprecated.
UPDATE ops_instances
SET status = 'deprecated',
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND instance_type = 'visual'
  AND NOT EXISTS (
    SELECT 1 FROM ops_integration_installations inst
    WHERE inst.instance_id = ops_instances.id
      AND inst.deleted_at IS NULL
      AND inst.status NOT IN ('uninstalled', 'uninstall_failed')
  );

-- Soft-delete visual instances with no integrations at all
UPDATE ops_instances
SET deleted_at = NOW(),
    status = 'deleted',
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND instance_type = 'visual'
  AND status = 'deprecated'
  AND NOT EXISTS (
    SELECT 1 FROM ops_integration_installations inst
    WHERE inst.instance_id = ops_instances.id
      AND inst.deleted_at IS NULL
  );

-- ── 6. User role migration ──────────────────────────────────────────────

UPDATE ops_users SET role = 'admin' WHERE role = 'platform_admin' AND deleted_at IS NULL;
UPDATE ops_users SET role = 'user' WHERE role = 'operator' AND deleted_at IS NULL;
UPDATE ops_users SET role = 'user' WHERE role = 'viewer' AND deleted_at IS NULL;

COMMIT;

-- ── Verification queries (run after COMMIT) ─────────────────────────────
-- SELECT count(*) FROM ops_workspaces;
-- SELECT count(*) FROM ops_users WHERE workspace_id IS NOT NULL;
-- SELECT count(*) FROM ops_grafana_instances WHERE source IS NOT NULL;
-- SELECT count(*) FROM ops_instances WHERE instance_type = 'visual' AND deleted_at IS NULL;
-- SELECT role, count(*) FROM ops_users WHERE deleted_at IS NULL GROUP BY role;
