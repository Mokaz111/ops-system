# Ops System

云原生可观测性平台控制面，当前聚焦 **VictoriaMetrics 管理能力**，并与 Grafana、夜莺 N9E 做系统集成。  
告警引擎由 N9E 独立运行，平台不自研告警引擎。

## 当前能力边界

- 组织与权限：部门、租户、用户、角色（含 admin/operator/viewer）
- 监控实例管理：创建、查询、更新、删除、实例级扩缩容（按策略限制）
- 平台级扩容：仅 `admin` 可对注册的 VMCluster 目标做 dry-run/应用
- 平台级审计：记录扩容变更人、目标、状态、变更内容；支持筛选查询
- 集成能力：Grafana 组织/用户/数据源，N9E 规则/事件/通知渠道接口

## 扩缩容策略

- `shared`：不允许实例级扩容；仅平台 `admin` 做平台级扩容
- `dedicated_cluster`：不允许实例级扩容；仅平台 `admin` 做平台级扩容
- `dedicated_single`：允许实例级垂直扩容与存储扩容，不允许实例级水平扩容

## 技术栈

| 层 | 选型 |
|---|---|
| Backend | Go, Gin, GORM, Zap, Viper |
| Frontend | React 19, TypeScript, MUI 7, Zustand, Vite 8 |
| Data | PostgreSQL, Redis |
| Infra | Kubernetes, Helm, VictoriaMetrics Operator |
| Integrations | Grafana, N9E |

## 快速开始

### 后端

```bash
cd backend
go mod tidy
make run
```

默认监听 `0.0.0.0:8080`。

### 前端

```bash
cd frontend
npm install
npm run dev
```

## 后端 API 概览

所有接口前缀为 `/api/v1`。

- 认证：`/auth/login`、`/auth/me`
- 公共初始化：`/users/bootstrap`
- 部门：`/departments`
- 租户：`/tenants`
- 用户：`/users`
- 实例：`/instances`
- 告警集成：`/alerts`
- Grafana 管理：`/grafana/orgs`
- 平台扩容（admin）：`/platform/scaling/vmcluster/targets`、`/platform/scaling/vmcluster`
- 平台扩容审计（admin）：`/platform/scaling/audits`

## 平台扩容审计查询

`GET /api/v1/platform/scaling/audits`（admin）

查询参数：

- `page`, `page_size`
- `target_id`
- `status`：`success` / `failed` / `replayed`
- `operator`：操作者用户名模糊查询
- `start_time`, `end_time`：RFC3339

## 目录结构

```
ops-system/
├── backend/
├── frontend/
└── docs/
```

## 文档说明

- `docs/` 存放设计与架构文档
- 历史阶段性开发计划文档已清理，不再作为当前实现依据
- 具体运行与开发说明见：
  - `backend/README.md`
  - `frontend/README.md`

## License

Private internal project.
