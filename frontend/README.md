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

## 平台扩容审计页面能力

- 历史列表：时间、操作者、目标、状态、来源 IP、错误信息
- 筛选：目标、状态、操作者、开始时间、结束时间
- 查看变更详情：查看每次操作的 `spec_patch`
- 一键重置筛选

## 环境变量

按项目实际 `.env` 配置为准，通常至少包含：

- `VITE_API_BASE_URL`

