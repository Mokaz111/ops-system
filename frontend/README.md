# Frontend

Ops System SaaS 可观测控制台：工作空间、监控/日志实例、接入中心、平台告警、业务集群采集配置、Grafana 与 Zone 管理。

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
- `src/pages`：页面模块（含 `Alert/` 子域）
- `src/components`：布局（Sidebar / WorkspaceSwitcher）与通用组件
- `src/config/appRoutes.ts`：路由元数据 + **嵌套** `sidebarNav`
- `src/stores`：`useAuthStore` / `useWorkspaceStore`
- `src/utils/membership.ts`：工作空间成员角色
- `src/types`：共享类型
- `src/router.tsx`：路由定义

## 信息架构（侧栏域）

概览 · 接入中心 · 监控（监控实例 / 指标库 / Grafana）· 日志（查询 / 日志实例）· 链路追踪 · 告警（事件 / 规则 / 静默）· 资源（业务集群 / UModel）· 管理（工作空间 / 用户 / 可用区 / 审计 / 设置）

文案统一：**监控实例 / 日志实例 / 工作空间**。

## 关键页面

- `Dashboard`：概览 + 告警统计
- `Workspace`：工作空间与成员
- `Instance` / `InstanceDetail`：监控实例
- `LogInstance` / `LogQuery`：日志
- `Alert/{Events,Rules,Channels,Silences}`：平台告警（规则支持 YAML/Zip 导入）
- `BusinessCluster`：业务集群 + **采集配置**
- `UModel`：Entity / MetricSet / LogSet
- `Integrations` / `Metrics` / `Grafana*` / `Zone` / `Audit` / `Settings`
- `Trace`：占位

## 环境变量

- `VITE_API_BASE_URL`

## 改版要点（SaaS IA）

- 嵌套可折叠侧栏由 `sidebarNav` 单一来源驱动
- TopBar `WorkspaceSwitcher` + `useWorkspaceStore`
- 告警一等公民：事件 / 规则 / 静默；渠道为规则内链；不再依赖 N9E
- 业务集群采集配置对话框（指标 + 日志）
- 实例详情路由：`/instances/:instanceId`
