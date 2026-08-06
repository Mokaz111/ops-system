# Ops System

云原生可观测性平台控制面，对齐腾讯云可观测平台（TCOP）形态。**全栈基于 VictoriaVictoriaMetrics**（VM Operator + VMCluster + VMAgent + VMRule + VictoriaLogs），告警由平台自管（规则 → VMRule，事件经 Alertmanager Webhook 入库，通知渠道投递）。

## 当前能力边界

### 组织与权限
- **工作空间（Workspace）** 为唯一租户边界；已移除 Department / 旧 Tenant 模块
- 平台角色：`admin` / `user`；工作空间成员角色：`admin` / `member` / `viewer`（`ops_workspace_members`）
- 数据访问按工作空间隔离；平台 `admin` 可跨空间操作，前端提供 **WorkspaceSwitcher**
- JWT 鉴权（支持 API Token）；首次部署通过 `POST /users/bootstrap` 一次性建出 admin

> 残余：部分模型/标签仍使用历史字段名 `tenant_id` / `ops_tenant_id`，语义上等于 Workspace UUID，API 查询参数优先使用 `workspace_id`。

### 监控
- **监控实例（Instance）**：共享池上的逻辑观测单元（CRUD、详情、指标）
- **接入中心**：模板市场 + 版本快照 + 实例级安装 / 升级 / 回滚 / 卸载，变更落 `revision` 审计
- **指标库**：从模板 `CollectorSpec` / `DashboardSpec` 解析指标，支持手工覆盖
- **业务集群**：下发 VMAgent；支持 **采集配置**（抓取间隔、命名空间包含/排除）
- 实例详情 Tab：基本信息 / 数据采集 / Dashboard / 告警

### 日志
- **日志实例（LogInstance）** 注册与 LogsQL 查询
- 业务集群可启停日志采集（Vector），配置路径/命名空间过滤
- UModel **LogSet** 元数据

### 告警（一等公民）
- 规则 CRUD + Prometheus 风格 YAML/Zip **批量导入**
- 事件列表 / 确认；Alertmanager Webhook 入库
- 通知渠道 CRUD；静默为前端路线图占位（Alertmanager Silence）
- 统计：summary / trend / by-level / by-rule

### 可视化与资源
- Grafana 实例（平台/外部）+ Org / Datasource / Dashboard 代理与 SSO
- Zone（可用区）初始化：共享指标池 / 日志管道 / Grafana
- UModel：Entity / MetricSet / LogSet / DataLink
- K8s 集群注册表（多集群下发）

### 平台运维
- 平台级 VMCluster 扩容（仅 admin，注册目标 + dry-run + Idempotency-Key）
- 共享集群初始化、审计日志

## 控制台信息架构

侧栏按业务域嵌套（见 `frontend/src/config/appRoutes.ts` → `sidebarNav`）：

| 域 | 入口 |
|---|---|
| 概览 | Dashboard |
| 接入中心 | 模板市场 |
| 监控 | 监控实例 · 指标库 · Grafana |
| 日志 | 日志查询 · 日志实例 |
| 链路追踪 | 占位页 |
| 告警 | 事件 · 规则 · 静默（渠道为规则页内链） |
| 资源 | 业务集群 · UModel |
| 管理 | 工作空间 · 用户* · 可用区 · 审计* · 系统设置 |

`*` 仅平台 `admin` 在侧栏显示。

## 扩缩容策略

当前以 **shared** 形态为主：实例级水平/垂直扩容拒绝，容量由平台管理员通过平台级扩容调整。伸缩/扩容写操作受幂等与审计保护。

## 技术栈

| 层 | 选型 |
|---|---|
| Backend | Go, Gin, GORM, Zap, Viper |
| Frontend | React 19, TypeScript, MUI 7, Zustand, Vite, axios, recharts |
| Data | PostgreSQL（业务+审计）, Redis（幂等键等） |
| Infra | Kubernetes（多集群可注册）, Helm, VictoriaMetrics Operator, VictoriaLogs, Vector, Kafka |
| Integrations | Grafana（多实例）, Alertmanager Webhook |

## 快速开始

### 后端
```bash
cd backend
go mod tidy
make run
```

默认监听 `0.0.0.0:8080`，配置见 `backend/configs/config.yaml`。
关键环境变量：

- `OPS_JWT_SECRET`：JWT 密钥（必填，长度 ≥ 32）
- `OPS_BCRYPT_COST`：密码哈希成本（默认 10，生产建议 12+）
- `OPS_DB_*` / `OPS_REDIS_*`：数据库/缓存连接

### 前端
```bash
cd frontend
npm install
npm run dev
```

## 后端 API 概览

所有接口前缀 `/api/v1`。鉴权：公共 → JWT（含 API Token）→ admin 写操作。

| 模块 | 主要路由 | 写权限 |
|---|---|---|
| 认证 | `POST /auth/login`、`GET /auth/me` | — |
| Bootstrap | `POST /users/bootstrap` | 公共（一次性） |
| 工作空间 | `/workspaces`、`/workspaces/:id/members` | admin / 成员管理 |
| 用户 | `/users` | admin / 本人 |
| API Token | `/api-tokens` | 登录用户 |
| 监控实例 | `/instances` | admin |
| 接入中心 | `/integrations/*` | 安装类登录；模板 admin |
| 指标库 | `/metrics`、`POST /metrics/reparse/:templateId` | admin |
| 日志实例 | `/log-instances`、`POST .../query` | admin |
| 告警 | `/alerts/rules[/import]`、`/events`、`/channels`、`/stats/*` | admin（写） |
| Webhook | `POST /webhooks/alertmanager` | 公共（token 校验） |
| 业务集群 | `/business-clusters`、`.../collect-config`、`enable-logs` / `disable-logs` | admin |
| UModel | `/umodel/entities|metric-sets|log-sets|data-links` | 登录 / 写视接口 |
| Zone | `/zones`、`preflight`、`components`、`init-*` | admin（写） |
| Grafana | `/grafana/*`、`/grafana/instances` | admin（写） |
| K8s 集群 | `/clusters` | admin |
| 平台扩容 | `/platform/scaling/*` | admin |
| 审计 | `/audits` | admin |
| 健康检查 | `/health`、`/api/v1/health`、`/api/v1/health/db` | — |

完整字段与隔离规则见 [`docs/02-后端详细设计.md`](docs/02-%E5%90%8E%E7%AB%AF%E8%AF%A6%E7%BB%86%E8%AE%BE%E8%AE%A1.md)。

## 接入中心生命周期（要点）

- 安装记录在 `(instance_id, template_id)` 上有 `where deleted_at IS NULL` 的 partial unique index
- 卸载是软删 + revision 审计；再次 `Install` 同模板复用旧 ID，`action='reinstall'`
- 删除模板会软删所有版本；若仍有活跃安装会被 409 拒绝
- 渲染支持 `dryRun`，K8s 资源统一打 `managed-by=ops-system` / `template=<id>` / `installation=<id>`

## 审计与可追溯

| 审计表 | 触发时机 |
|---|---|
| `platform_scale_audits` | 平台扩容（含 dry-run、apply、replay） |
| `ops_integration_installation_revisions` | 接入安装/升级/重装/卸载 |
| `ops_audit_logs` 等 | 管理操作审计（见审计页） |

## 文档

| 文档 | 内容 |
|---|---|
| [`docs/云原生可观测性监控平台-总体设计文档v4.md`](docs/%E4%BA%91%E5%8E%9F%E7%94%9F%E5%8F%AF%E8%A7%82%E6%B5%8B%E6%80%A7%E7%9B%91%E6%8E%A7%E5%B9%B3%E5%8F%B0-%E6%80%BB%E4%BD%93%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3v4.md) | SaaS 目标架构（权威） |
| `docs/01-前端详细设计.md` | 嵌套 IA、页面、WorkspaceSwitcher |
| `docs/02-后端详细设计.md` | 分层、路由、Workspace RBAC、模型 |
| `docs/03-部署架构详细设计.md` | 部署形态、依赖、凭据 |
| `docs/04-数据流详细设计.md` | Provisioning / 采集配置 / 告警导入 |
| `docs/05-告警引擎详细设计.md` | 平台告警 + VMRule + Webhook |
| `docs/06-用户同步详细设计.md` | Bootstrap、Workspace 成员 |
| `docs/07-运维监控设计.md` | 健康/审计/告警建议 |

文档与代码冲突时，**以代码实现为准**，并修正文档。

## License

Private internal project.
