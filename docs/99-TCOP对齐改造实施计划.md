# TCOP 对齐改造实施计划（临时）

> 本文档为阶段性工作计划，M6 完成后应并入 `01-前端详细设计.md` / `02-后端详细设计.md` / `云原生可观测性监控平台-总体设计文档v3.md`，并删除此文件。

## 一、背景与目标

对齐腾讯云可观测平台（TCOP）的产品形态，在现有 `ops-system`（控制面，已完成部门/租户/用户/VM 实例/平台扩容/Grafana 集成/N9E 告警接口）基础上扩展为：

- 监控管理（VM 实例为中心）
- 日志管理（VictoriaLogs）
- Grafana 管理（多主机：平台共享 + 租户自带）
- Dashboard 管理（平台托管大盘）
- 接入中心（采集 + 告警 + 大盘三合一模版市场）
- 指标库（指标字典 + 模版关联）

**核心替换**：全部使用 VictoriaMetrics 栈，不引入 Prometheus。

## 二、关键决策（已与用户对齐）

| 决策项 | 结论 |
|---|---|
| 日志引擎 | VictoriaLogs（Helm/VLogs CR） |
| 告警下发 | 模版内 `alertTargets` 声明；**M2 仅实现 VMRule**；N9E 下发器占位 |
| 模版存储 | PostgreSQL，版本化 |
| 指标来源 | 从采集模版 YAML 与 Dashboard JSON 自动解析；允许手工覆盖 |
| 安装粒度 | 按 VM 监控实例（1 模版 × 1 实例 = 1 安装记录） |
| Grafana 归属 | 平台共享 + 租户自带（GrafanaHost 注册表） |
| 实例详情 Tab | 基本信息 / 数据采集 / Dashboard / 告警（四 Tab） |
| 告警菜单 | 移除独立菜单，入口收敛到实例详情 |

## 三、信息架构与菜单

| 分区 | 菜单 | 说明 |
|---|---|---|
| 概览 | Dashboard 概览 | 保留 |
| 资源 | 部门 / 租户 / 用户 | 保留 |
| 监控 | 监控实例 / 接入中心 / 指标库 | **新增 2 项** |
| 日志 | 日志实例 / 日志查询 | **全新分区** |
| 可视化 | Grafana 管理 / Dashboard 管理 | **拆分原 Grafana 菜单** |
| 系统 | 平台扩容 / 系统设置 | 保留 |

移除：独立的"告警引擎"菜单。

## 四、后端实现

### 4.1 新增数据模型（`backend/internal/model`）

- `log_instance.go`：LogInstance
- `integration_template.go`：IntegrationTemplate + IntegrationTemplateVersion
- `integration_installation.go`：IntegrationInstallation + IntegrationInstallationRevision
- `metric.go`：Metric + MetricTemplateMapping
- `grafana_host.go`：GrafanaHost（scope=platform|tenant）

关键字段见主对话，本文档不展开。

### 4.2 新增服务（`backend/internal/service`）

- `vlogs_service.go`
- `integration_template_service.go`
- `integration_installer.go`（核心：Plan + Apply，子下发器：vmAgent/vmPodScrape/vmServiceScrape/vmRule/grafanaDashboard；N9eAlertApplier 接口 + noop 实现）
- `integration_installation_service.go`（升级/回滚/卸载）
- `metric_parser_service.go`（解析 CollectorSpec + DashboardSpec）
- `metric_service.go`
- `grafana_host_service.go`

### 4.3 基础设施封装

- `internal/vm`：ApplyVMPodScrape/ApplyVMServiceScrape/ApplyVMRule（支持 server-side dryRun，统一打 `managed-by=ops-system` + `template=<id>` + `installation=<id>` 标签）
- `internal/grafana`：ImportDashboard / UpdateDashboard / DeleteDashboard / EnsureFolder
- `internal/helm`：支持 `vm/victoria-logs-single` 与 `vm/victoria-logs-cluster`

### 4.4 新增路由（`/api/v1`）

```
# 日志
GET/POST/PUT/DELETE /log-instances
POST  /log-instances/:id/query

# 接入中心
GET   /integrations/categories
GET   /integrations/templates
POST  /integrations/templates                 (admin)
GET   /integrations/templates/:id
POST  /integrations/templates/:id/versions    (admin)
POST  /integrations/install/plan
POST  /integrations/install                   (Idempotency-Key)
GET   /integrations/installations
POST  /integrations/installations/:id/upgrade
POST  /integrations/installations/:id/rollback
DELETE /integrations/installations/:id

# 指标库
GET   /metrics
GET   /metrics/:id
POST  /metrics                                (admin)
PUT   /metrics/:id
DELETE /metrics/:id
POST  /metrics/reparse/:templateId            (admin)
GET   /metrics/:id/related

# Dashboard
GET   /dashboards
GET   /dashboards/installed
POST  /dashboards/install
DELETE /dashboards/installed/:id

# Grafana 多主机
GET/POST/PUT/DELETE /grafana/hosts            (admin)
```

### 4.5 幂等与审计

- 写操作均走现网 `idempotency` 中间件
- 新增 `integration_audit` 表（或扩展 `platform_scale_audit` 增 `action_type`）

### 4.6 权限矩阵

| 接口类 | viewer | operator | admin |
|---|---|---|---|
| 列表/详情、日志查询、已装查询 | V | V | V |
| 安装/升级/回滚/卸载（本租户实例） | - | V | V |
| 模版/指标本体 CRUD、日志实例 CRUD、Grafana 主机注册 | - | - | V |

## 五、前端实现

### 5.1 路由 / 侧边栏（`src/config/appRoutes.ts` + `src/router.tsx`）

移除 `alerts`；新增：`integrations`、`metrics`、`log-instances`、`logs/query`、`dashboards`、（已有 `grafana` 保留）。

### 5.2 新增 / 改造页面

| 页面 | 备注 |
|---|---|
| `pages/Integration/` | 分类树 + 卡片 + TemplateDrawer（Tab：安装/指标/Dashboard/告警/已集成/Grafana）+ InstallWizard（Stepper：选目标 → 勾选 parts → dry-run diff → 确认） |
| `pages/Metric/` | DataGrid + MetricDrawer（描述/标签/示例/关联模版/大盘 panel） |
| `pages/LogInstance/` + `pages/LogQuery/` | 实例 CRUD + LogsQL 查询台 |
| `pages/Dashboard/`（改造） | 内置模版 + 已安装列表（按 GrafanaHost + Org 过滤） |
| `pages/InstanceDetail/`（改造） | 四 Tab：基本信息 / 数据采集（跳转接入中心带 `?instanceId=`）/ Dashboard（本实例已装大盘）/ 告警（已下发 VMRule 只读 + N9E 跳转占位） |
| `pages/Grafana/`（增强） | 增加 Grafana Host 子页 |

### 5.3 API / Store 层

- `src/api/integration.ts`、`metric.ts`、`logs.ts`、`grafanaHost.ts`
- `src/stores/integrationStore.ts`、`metricStore.ts`

### 5.4 新依赖（待定）

- 差异预览：`react-diff-viewer-continued`
- YAML/JSON 只读：`@uiw/react-textarea-code-editor`（或 Monaco，视包体积）

## 六、里程碑

| 阶段 | 交付物 | 估算 |
|---|---|---|
| **M1 骨架** | 侧边栏重排；InstanceDetail 四 Tab 壳；Metric/Integration/LogInstance 空 CRUD；后端 migration + model + 路由壳 | 3~4 天 |
| M2 接入中心核心 | 模版/版本模型、安装器、dry-run diff、审计、安装向导 | 6~8 天 |
| M3 指标库闭环 | 解析器 + 关联展示 | 4~5 天 |
| M4 日志管理 | VLogs 实例 CRUD + LogsQL 查询 + 日志采集模版入驻 | 4 天 |
| M5 Grafana 多主机 | GrafanaHost + Dashboard 管理页 | 2 天 |
| M6 加固 | N9E 下发器占位、权限 e2e、审计查询扩展 | 2 天 |

## 七、M1 拆解（立即执行）

### 后端
1. 新建 model 文件（LogInstance、IntegrationTemplate + Version、IntegrationInstallation + Revision、Metric + Mapping、GrafanaHost），提供 `AutoMigrate` 注册
2. 新增 repository 层（最小 CRUD）
3. 新增 service 层空实现（仅返回列表 / 404 / 占位）
4. 新增 handler 层 + 路由注册（list/get/create/update/delete 壳）
5. 保持全部写接口 `admin` 鉴权、查询接口 `authenticated`

### 前端
1. 更新 `appRoutes.ts`：移除 `alerts`，新增 `integrations` / `metrics` / `log-instances` / `logs-query` / `dashboards`，调整 sidebar section
2. `router.tsx`：lazy 引入新页面
3. 新建页面壳（空列表 + Breadcrumb + 占位文案）
4. `InstanceDetail` 改造为四 Tab（基本信息已有内容保留，其余 Tab 占位文案）
5. 侧边栏分区重排，新增 `monitor` / `logs` / `visualization` 分区

### 交付验收
- 后端：`go build` 通过、`make run` 起得来、新接口能返回空列表
- 前端：`npm run build` 通过；侧边栏新菜单可点开且显示占位页；InstanceDetail 展示四 Tab
- 无业务逻辑实现，仅骨架，方便后续 M2 在已经接通的通道上填肉
