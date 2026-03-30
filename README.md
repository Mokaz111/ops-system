# 云原生可观测性监控平台 (Ops System)

全界面化的云原生监控平台，将 VictoriaMetrics、VictoriaLogs、Grafana、夜莺 (N9E) 等可观测性组件封装为平台服务，用户通过 Web 控制台即可完成租户创建、监控实例开通、数据可视化等全部操作，无需关注底层 Kubernetes 与 Helm 的部署细节。

## 核心特性

- **全界面化运营** — 租户管理、实例开通、告警配置等均通过 Web 控制台完成
- **自动化编排** — 后端自动执行 Helm Release 部署、K8s Namespace 创建、资源配额配置
- **多租户隔离** — 租户与 VMuser 一一映射，网络、数据、资源全维度隔离
- **灵活实例模板** — 共享版 / 独享单节点版 / 独享集群版，覆盖从开发测试到大型生产的场景
- **统一用户体系** — 平台用户自动同步至 Grafana Org、N9E Team，实现单点登录与权限对齐
- **可观测性整合** — 指标 (Metrics) + 日志 (Logs) + 告警 + 可视化，一站式解决方案

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端 | Go (Gin + GORM) | 1.25.8 |
| 前端 | React + TypeScript + MUI (Material Design 3) | React 19, MUI 7 |
| 构建工具 | Vite | 8.x |
| 状态管理 | Zustand | 5.x |
| 时序数据库 | VictoriaMetrics | 1.102.x |
| 日志存储 | VictoriaLogs | 1.12.x |
| 告警引擎 | 夜莺 N9E (独立部署) | v8.beta14+ |
| 可视化 | Grafana | - |
| 容器编排 | Kubernetes + Helm | K8s 1.26+, Helm 3.14+ |
| 数据库 | PostgreSQL + Redis | 15.x / 7.x |

## 项目结构

```
ops-system/
├── backend/                  # Go 后端服务
│   ├── cmd/server/           # 程序入口
│   ├── configs/              # 配置文件
│   └── internal/
│       ├── auth/             # JWT 认证
│       ├── config/           # 配置加载
│       ├── grafana/          # Grafana API 客户端（组织、用户、数据源、Dashboard）
│       ├── handler/          # HTTP Handler（租户、用户、实例、告警、部门）
│       ├── helm/             # Helm 客户端与 values 模板
│       ├── k8s/              # Kubernetes 客户端
│       ├── middleware/       # 中间件（认证、CORS、限流、日志）
│       ├── model/            # 数据模型
│       ├── n9e/              # 夜莺 N9E API 客户端
│       ├── server/           # HTTP 服务器与路由
│       ├── service/          # 业务逻辑层
│       ├── vm/               # VictoriaMetrics vmauth 操作
│       └── worker/           # 后台任务
├── frontend/                 # React 前端
│   └── src/
│       ├── api/              # Axios API 封装
│       ├── components/       # 公共组件（布局、通用 UI）
│       ├── pages/            # 页面（Dashboard、租户、实例、告警、用户、部门等）
│       ├── stores/           # Zustand 状态管理
│       ├── theme/            # Material Design 3 主题
│       └── types/            # TypeScript 类型定义
└── docs/                     # 设计文档
```

## 快速开始

### 环境要求

- Go 1.25.8+
- Node.js 20+ / npm
- PostgreSQL 15+
- Redis 7+ (可选)
- Kubernetes 1.26+ 集群 (生产环境)

### 后端启动

```bash
cd backend

# 安装依赖
go mod tidy

# 修改配置（数据库连接、JWT 密钥等）
cp configs/config.yaml configs/config.yaml.local
vi configs/config.yaml

# 开发模式运行
make run

# 或编译后运行
make build
./bin/server
```

后端默认监听 `0.0.0.0:8080`。

### 前端启动

```bash
cd frontend

# 安装依赖
npm install

# 配置环境变量
cp .env.example .env

# 开发模式运行
npm run dev

# 生产构建
npm run build
```

### 配置说明

后端核心配置位于 `backend/configs/config.yaml`，主要配置项：

| 配置项 | 说明 |
|--------|------|
| `server` | HTTP 服务地址与端口 |
| `database` | PostgreSQL 连接信息 |
| `jwt` | JWT 密钥与过期时间 |
| `cors` | 跨域策略，生产环境需设置 `allowed_origins` |
| `orchestration` | K8s/Helm 编排开关，默认关闭 |
| `vm` | VictoriaMetrics vmauth 配置 |
| `grafana` | Grafana API 地址与密钥 |
| `n9e` | 夜莺 N9E 连接配置 |

前端环境变量位于 `frontend/.env`：

| 变量 | 说明 |
|------|------|
| `VITE_API_BASE_URL` | 后端 API 地址，默认 `/api/v1` |
| `VITE_GRAFANA_URL` | Grafana 访问地址 |
| `VITE_N9E_URL` | 夜莺 N9E 访问地址 |

## API 概览

所有接口均以 `/api/v1` 为前缀，需携带 JWT Token（登录接口除外）。

| 模块 | 路径前缀 | 说明 |
|------|---------|------|
| 认证 | `/auth` | 登录、注册、Token 刷新 |
| 租户 | `/tenants` | 租户 CRUD、资源同步 |
| 实例 | `/instances` | 监控实例管理、扩缩容 |
| 用户 | `/users` | 用户 CRUD、角色管理 |
| 部门 | `/departments` | 组织结构管理 |
| 告警 | `/alerts` | 告警规则、告警事件、通知渠道 |
| Grafana | `/grafana` | 组织、用户、数据源、Dashboard 管理 |

## 系统架构

```
┌──────────────────────────┐
│   Web 控制台 (React/MUI)  │
└────────────┬─────────────┘
             │
┌────────────▼─────────────┐
│   API Gateway (Go/Gin)   │
│   认证 · 限流 · 日志审计    │
└────────────┬─────────────┘
             │
   ┌─────────┼──────────┐
   ▼         ▼          ▼
 租户服务  实例服务  Grafana服务
   │         │          │
   └─────────┼──────────┘
             ▼
┌──────────────────────────┐
│      编排引擎             │
│  Helm · K8s · VM Operator │
└────────────┬─────────────┘
             │
   ┌─────────┼──────────┐
   ▼         ▼          ▼
VMCluster  VLogs    Grafana    ← 每租户独立或共享
             │
          N9E (独立告警引擎)
```

## 多租户模型

- **部门 ↔ 租户 1:1 映射**：每个部门对应一个租户，部门用户自动关联到租户资源
- **实例模板**：
  - **共享版** — 复用全局 VMCluster，按 tenant_id 隔离数据，成本最低
  - **独享单节点版** — 独立 VMSingle，适合中小型业务
  - **独享集群版** — 独立 VMCluster 高可用集群，适合大型生产环境
- **隔离策略**：K8s Namespace 隔离 + NetworkPolicy + ResourceQuota + 数据层 tenant_id 隔离

## 开发命令

```bash
# === 后端 ===
make run          # 开发模式运行
make build        # 编译二进制
make test         # 运行测试
make fmt          # 代码格式化
make vet          # 静态检查
make tidy         # 整理依赖

# === 前端 ===
npm run dev       # 开发服务器
npm run build     # 生产构建
npm run lint      # ESLint 检查
npm run preview   # 预览构建产物
```

## 设计文档

详细设计文档位于 `docs/` 目录：

| 文档 | 内容 |
|------|------|
| `云原生可观测性监控平台-总体设计文档v3.md` | 系统总体设计 |
| `01-前端详细设计.md` | 前端架构与页面设计 |
| `02-后端详细设计.md` | 后端服务与 API 设计 |
| `03-部署架构详细设计.md` | K8s 部署与 Helm 编排 |
| `04-数据流详细设计.md` | 数据写入与查询链路 |
| `05-告警引擎详细设计.md` | 夜莺 N9E 告警引擎集成 |
| `06-用户同步详细设计.md` | 跨系统用户同步方案 |
| `07-运维监控设计.md` | 平台自身运维监控 |

## License

Private — 内部项目，未公开授权。
