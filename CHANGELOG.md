# Changelog

## [Unreleased] - 2026-06-27

### BREAKING — 纯 SaaS 平台化

- **Tenant → Workspace**：`/api/v1/tenants` → `/api/v1/workspaces`，模型/表名/API 全部重命名
- **删除 Department 模块**：`/api/v1/departments` 路由移除，`ops_departments` 表备份为 `_deprecated`
- **删除 TenantMember / AuthzService**：细粒度 RBAC 移除，`ops_tenant_members` 表备份
- **删除 ServiceAccount / APIToken**：独立权限体系移除
- **User.role 简化**：仅 `admin` / `user` 两种角色；`platform_admin` → `admin`；`operator`/`viewer` → `user`
- **User.TenantID → WorkspaceID**：`tenant_id` 列替换为 `workspace_id`
- **删除 instance_type='visual'**：Grafana 不再是 Instance 类型；GrafanaInstance 新增 `source` 字段（platform/external）
- **GrafanaInstance 删除 scope/tenant_id**：统一为平台级，通过 Org 映射隔离工作空间
- **SSO 登录统一入口**：`POST /api/v1/instances/:id/login` 删除，仅保留 `POST /api/v1/grafana/instances/:id/login`
- **前端删除**：Department、DashboardMgmt、GrafanaHost(旧)、Grafana(旧)、VMStats 页面全部删除

### Added — Grafana 管理端到端可用

- Zone InitGrafana 自动注册 GrafanaInstance（source='platform'）
- Grafana Helm values 增加 `[auth.proxy]` 配置，SSO 免登录链路打通
- 代理 cookie Secure 动态化（X-Forwarded-Proto）
- Dashboard 管理合并到 Grafana 详情页 Dashboard Tab

### BREAKING — 早期 Grafana/VMs 生命周期修复

- **Grafana 代理路径行为变更**：反向代理现在会在转发前剥离 `/api/v1/grafana/proxy` 路径前缀，Grafana values 配置了 `serve_from_sub_path: true` + 匹配的 `root_url`。依赖旧路径直转行为的客户端需适配。前端 SSO 跳转已同步更新。

### Fixed — 后端 Grafana 生命周期

- 修复反向代理不剥离路径前缀导致 SSO 代理后 Grafana 404（`grafana_proxy_handler.go`）
- Grafana values 配置 `serve_from_sub_path` 与 `root_url`，子路径代理可用（`grafana.yaml`）
- `InitGrafana` 内联 values 同步配置子路径 + admin 凭据（`zone_service.go`）
- dedicated 模板 Grafana 密码占位符 `${GRAFANA_ADMIN_PASSWORD}` 做 envsubst，未设置时拒绝部署（`values.go`、`orchestrator.go`）
- `DeleteOrg`/`DeleteDatasource` 处理 404 视为成功，租户删除可重试（`org.go`、`user.go`）
- `ZoneID` 端到端贯通：Create/Update Request 增字段、Create 写入、List 支持过滤（`grafana_instance_service.go`、`grafana_instance_repo.go`、`grafana_instance_handler.go`）
- `CreateOrgForTenant` 回写 `GrafanaOrgID` 错误上抛/记日志，不再 `_ =` 静默吞（`grafana_service.go`）
- 代理 cookie `Secure` 按 `X-Forwarded-Proto` 动态设置，确认 `HttpOnly`/`SameSite=Lax`（`context.go`、`grafana_instance_handler.go`、`instance_handler.go`）
- Grafana 客户端写操作补有限重试（2 次指数退避，仅 5xx/网络错误）与超时控制（`client.go`）

### Fixed — 后端 VM 生命周期

- VM 实例 `Delete` 实现资源回收：shared 回收 VMUser/VMRoute，dedicated 回收 VMCluster CR（`instance_service.go`）
- VM Operator 新增 `DeleteVMCluster`/`DeleteVMUser` 删除方法（幂等，已不存在视为成功）（`operator.go`）
- `Create` 部署成功后显式置 `running`，不依赖默认关闭的 `InstanceStatusAutoAdvance`（`instance_service.go`）
- 实例变更引入 repository 事务，状态修正错误不再 `_ =` 静默丢弃（`instance_service.go`、`instance_repo.go`）
- `Rebuild`/`Upgrade` 改为实例级操作：按实例自身 CR 重建/升级，不再误删整个租户命名空间（`instance_service.go`）
- `GetMetrics` PromQL 选择器对实例名做转义防注入（`instance_service.go`）
- 删除回收部分失败 best-effort：记告警日志后仍软删 DB，避免操作卡死

### Fixed — 前端

- 修复 GrafanaInstance 统一列表分页逻辑（`count` 不再叠加不分页的 hosts、翻页不重复渲染 hosts）
- 删除/重建 ConfirmDialog 补 `loading` 锁定，防止重复点击（GrafanaInstance、Instance、DashboardMgmt）
- VM 详情页重建增加二次确认弹窗、补删除入口（`InstanceDetail/index.tsx`）
- VM 创建页补回 `dedicated_single` 模板选项（`Instance/Create.tsx`）
- 修复 VM 详情页编辑保存丢失 `replicas`/独享集群 spec 结构（`InstanceDetail/index.tsx`）
- SSO `window.open` 在用户手势上下文内同步打开，弹窗被拦截时提示含 URL 的错误信息（`grafanaSso.ts`）
- GrafanaInstanceDetail 写操作按钮按 `isAdmin` 角色控制（`GrafanaInstanceDetail/index.tsx`）
- 编辑保存改局部 refetch 替换 `window.location.reload()`（InstanceDetail、GrafanaInstanceDetail）
- 伸缩弹窗补校验与下限保护、移除 `horizontal` 死分支（`Instance/index.tsx`）

### Removed — 死代码清理

- 删除 `frontend/src/pages/VMStats/` 目录（未挂路由）
- 删除 `frontend/src/pages/GrafanaHost/` 目录与路由（功能已被 GrafanaInstance 覆盖）
- 删除 `frontend/src/pages/Grafana/`（旧）目录与路由（功能已被 GrafanaInstanceDetail 覆盖）

### Added

- 一次性迁移脚本 `backend/scripts/mark_stale_creating_instances_failed.sql`：将无活跃集成的 `creating` 僵尸实例标记为 `failed`
- 单元测试：代理路径剥离（`grafana_proxy_handler_test.go`）、PromQL 转义（`instance_service_test.go`）、DeleteOrg 幂等（`client_test.go`）、envsubst（`values_test.go`）
