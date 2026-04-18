# Ops System

云原生可观测性平台控制面，对齐腾讯云可观测平台（TCOP）形态，**全栈基于 VictoriaMetrics**（VM Operator + VMCluster + VMAgent + VMRule + VictoriaLogs），告警引擎复用 N9E（独立部署）。

## 当前能力边界

### 组织与权限
- 部门、租户、用户、角色（`admin` / `operator` / `viewer`）
- 多租户接口自动按 `tenant_id` 做数据隔离；`admin` 全局可见
- JWT 鉴权，登出走 token 黑名单（Redis），即时失效；前端 `useAuthStore` 与 `localStorage` 通过 `UNAUTHORIZED_EVENT` 同步
- 首次部署通过 `POST /users/bootstrap` 一次性建出 admin（已存在则拒绝）

### 监控管理
- 监控实例（VMCluster）生命周期：CRUD、详情、扩缩容
- **接入中心**：模板市场 + 版本快照 + 实例级安装 / 升级 / 重装 / 卸载，每次变更落 `revision` 审计
- **指标库**：从模板的 `CollectorSpec` / `DashboardSpec` 自动解析指标，支持手工覆盖；按组件/名称去重
- 实例详情四 Tab：基本信息 / 数据采集 / Dashboard / 告警

### 日志管理
- 日志实例（VictoriaLogs）注册与 LogsQL 查询接口

### 可视化
- Grafana 主机注册表（`platform` 共享 + `tenant` 自带）
- Dashboard 通过接入中心模板下发到指定 Grafana Host

### 平台运维
- 平台级 VMCluster 扩容（仅 admin，按注册目标，dry-run + Idempotency-Key 双保护）
- 共享集群初始化（admin，固定 `vm/victoria-metrics-k8s-stack` Chart）
- K8s 集群注册表（多集群下发，按 `cluster_id` 解析 kubeconfig，含线程安全缓存）
- 平台扩容审计 + 实例伸缩审计（`scale_events`） + 接入变更审计（`integration_installation_revisions`）

## 扩缩容策略

| 部署模式 | 实例级水平扩容 | 实例级垂直/存储扩容 | 平台级扩容 |
|---|---|---|---|
| `shared` | 拒绝 | 拒绝 | 仅 admin |
| `dedicated_cluster` | 拒绝 | 拒绝 | 仅 admin |
| `dedicated_single` | 拒绝 | 允许 | 仅 admin |

伸缩并发受互斥保护：同一 `instance_id` 同一时刻只允许一次 scale，防止并发覆盖。

## 技术栈

| 层 | 选型 |
|---|---|
| Backend | Go, Gin, GORM, Zap, Viper |
| Frontend | React 19, TypeScript, MUI 7, Zustand, Vite, axios |
| Data | PostgreSQL（业务+审计）, Redis（幂等键 + JWT 黑名单） |
| Infra | Kubernetes（多集群可注册）, Helm, VictoriaMetrics Operator, VictoriaLogs |
| Integrations | Grafana（多主机）, N9E |

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

所有接口前缀 `/api/v1`。鉴权策略：公共 → JWT → admin 三层。

| 模块 | 主要路由 | 写权限 |
|---|---|---|
| 认证 | `POST /auth/login`、`GET /auth/me`、`POST /auth/logout` | — |
| Bootstrap | `POST /users/bootstrap` | 公共（一次性） |
| 部门 | `/departments` | admin |
| 租户 | `/tenants` | admin |
| 用户 | `/users`（含本人改本人 PUT） | admin / 本人 |
| 监控实例 | `/instances`、`/instances/:id/scale-events`、`POST /instances/:id/scale` | admin |
| 接入中心 | `/integrations/templates`、`/integrations/templates/:id/versions`、`POST /integrations/install/plan`、`POST /integrations/install`、`/integrations/installations[/:id/revisions]`、`DELETE /integrations/installations/:id` | admin（模板/版本）/ 本租户 admin/operator（安装/卸载） |
| 指标库 | `/metrics`、`/metrics/:id/related`、`POST /metrics/reparse/:templateId` | admin |
| 日志 | `/log-instances`、`POST /log-instances/:id/query` | admin |
| Grafana 业务 | `/grafana/orgs/*` | admin |
| Grafana 主机 | `/grafana/hosts` | admin |
| K8s 集群 | `/clusters` | admin |
| 告警集成 | `/alerts/*` | admin |
| 平台扩容 | `/platform/scaling/vmcluster*`、`/platform/scaling/audits`、`/platform/scaling/bootstrap/shared/init` | admin |
| 健康检查 | `/health`、`/api/v1/health`、`/api/v1/health/db` | — |

完整字段、错误码、租户隔离规则见 [`docs/02-后端详细设计.md`](docs/02-%E5%90%8E%E7%AB%AF%E8%AF%A6%E7%BB%86%E8%AE%BE%E8%AE%A1.md)。

## 接入中心生命周期（要点）

- 安装记录在 `(instance_id, template_id)` 上有 `where deleted_at IS NULL` 的 partial unique index：同一实例同一模板只允许一条活跃记录
- 卸载是软删 + revision 审计；卸载后再次 `Install` 同模板会复用旧 ID，`action='reinstall'`
- 删除模板会软删所有版本；若仍有活跃安装会被 409 拒绝
- 所有渲染调用支持 `dryRun`，写入 K8s 资源统一打 `managed-by=ops-system` / `template=<id>` / `installation=<id>` 标签

## 审计与可追溯

| 审计表 | 触发时机 | 字段要点 |
|---|---|---|
| `platform_scale_audits` | 平台扩容请求（含 dry-run、apply、replay） | 操作者、IP、目标、状态、`spec_patch` |
| `scale_events` | 实例 scale（成功/失败/拒绝） | scale_type、method、replicas/cpu/memory/storage、operator |
| `integration_installation_revisions` | 接入安装/升级/重装/卸载 | action、spec_diff、applied_resources、operator、status |

## 文档

| 文档 | 内容 |
|---|---|
| [`docs/云原生可观测性监控平台-总体设计文档v3.md`](docs/%E4%BA%91%E5%8E%9F%E7%94%9F%E5%8F%AF%E8%A7%82%E6%B5%8B%E6%80%A7%E7%9B%91%E6%8E%A7%E5%B9%B3%E5%8F%B0-%E6%80%BB%E4%BD%93%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3v3.md) | 总体设计 |
| `docs/01-前端详细设计.md` | 前端页面 / 路由 / 状态 / 错误归一 |
| `docs/02-后端详细设计.md` | 分层、模块、模型、错误码、租户隔离 |
| `docs/03-部署架构详细设计.md` | 部署形态、依赖矩阵、敏感凭据 |
| `docs/04-数据流详细设计.md` | 关键数据流（含接入安装、scale、指标解析） |
| `docs/05-告警引擎详细设计.md` | alertTargets → VMRule，N9E 占位 |
| `docs/06-用户同步详细设计.md` | bootstrap、外部组织映射 |
| `docs/07-运维监控设计.md` | 健康/审计/告警建议 |
| `docs/99-TCOP对齐改造实施计划.md` | 历史阶段计划归档（M1–M6 已完成） |

文档与代码冲突时，**以代码实现为准**，并修正文档。

## License

Private internal project.
