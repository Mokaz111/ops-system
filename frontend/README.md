# Frontend

Ops System 前端控制台，基于 React + TypeScript + MUI，实现租户、实例、用户、告警集成、Grafana 管理与平台扩容页面。

## 环境要求

- Node.js 20.19+（或 22.12+）
- npm 10+

## 启动

```bash
npm install
npm run dev
```

开发默认地址：`http://localhost:5173`

## 构建与检查

```bash
npm run lint
npm run build
npm run preview
```

## 目录说明

- `src/api`：接口封装（Axios）
- `src/pages`：页面模块
- `src/components`：布局与通用组件
- `src/stores`：Zustand 状态管理
- `src/types`：共享类型定义
- `src/router.tsx`：路由定义

## 关键页面

- `Dashboard`
- `Department` / `Tenant` / `Instance`
- `Alert`（N9E 集成视图）
- `Grafana`
- `PlatformScaling`（平台级扩容 + 变更历史审计）

## 平台扩容页面能力

- 历史列表：时间、操作者、目标、状态、来源 IP、错误信息
- 筛选：目标、状态、操作者、开始时间、结束时间
- 查看变更详情：查看每次操作的 `spec_patch`
- 一键重置筛选

## 系统设置页面能力

- 共享集群初始化（admin）：支持 `vm/victoria-metrics-k8s-stack` 的 dry-run 与确认应用

## 环境变量

按项目实际 `.env` 配置为准，通常至少包含：

- `VITE_API_BASE_URL`

## 集群前端改版要点（2026-04）

- 实例详情升级为路由页：`/instances/:instanceId`，支持直达访问与分享链接。
- 实例管理页新增“接入数据概览”和类型快速筛选，列表操作统一为详情/监控/伸缩/删除。
- 告警页支持实例上下文参数（`instance_id`、`instance_name`），便于从实例详情直接联动到 N9E。
- 路由与侧栏导航统一由 `src/config/appRoutes.ts` 管理，减少菜单与路由双维护问题。
- 新增可复用组件：
  - `src/components/common/FilterToolbar.tsx`
  - `src/components/common/DataTableCard.tsx`
  - `src/components/common/DetailTabs.tsx`

