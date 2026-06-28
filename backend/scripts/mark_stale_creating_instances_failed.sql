-- 一次性迁移脚本：修复存量 VM 实例状态机缺口。
--
-- 背景：在状态机修复前，InstanceService.Create 成功后不会将状态从 creating 转为 running
-- （依赖默认关闭的 InstanceStatusAutoAdvance），导致部分实例永久停留在 creating。
-- 本脚本将"无活跃集成"的 creating 实例标记为 failed，供人工排查后重建或删除；
-- 仍存在活跃集成的 creating 实例保留，避免误判。
--
-- 执行前请先备份数据库。幂等：可重复执行，仅作用于 status='creating' 的记录。
--
-- 用法（psql）：
--   psql "$DATABASE_URL" -f backend/scripts/mark_stale_creating_instances_failed.sql
--
-- 回滚：如需恢复，按执行前备份还原 ops_instances.status 字段。

BEGIN;

UPDATE ops_instances i
SET status = 'failed',
    updated_at = NOW()
WHERE i.deleted_at IS NULL
  AND i.status = 'creating'
  AND NOT EXISTS (
    SELECT 1
    FROM ops_integration_installations inst
    WHERE inst.instance_id = i.id
      AND inst.deleted_at IS NULL
      AND inst.status NOT IN ('uninstalled', 'uninstall_failed')
  );

COMMIT;

-- 验证：查看仍处于 creating 的实例（应有活跃集成）。
-- SELECT id, instance_name, status FROM ops_instances WHERE status = 'creating' AND deleted_at IS NULL;
