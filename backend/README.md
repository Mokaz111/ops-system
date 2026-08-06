# Backend

Ops System 后端控制面：多工作空间隔离、监控/日志实例、接入中心、平台告警、业务集群采集配置、Zone/Grafana 与平台扩容审计。

## 运行要求

- Go 1.25.8+
- PostgreSQL 15+
- Redis 7+（建议；幂等键等）

## 启动

```bash
go mod tidy
cp configs/config.example.yaml configs/config.yaml   # 首次需要
# 填好 configs/config.yaml 中的 ${OPS_*} 或设置环境变量
make run
```

默认地址：`http://0.0.0.0:8080`

### 配置与秘钥

- `configs/config.example.yaml` 是模板；**实际配置写在 `configs/config.yaml` 且勿提交**。
- 敏感项（`database.password` / `jwt.secret` / `grafana.api_key` 等）支持 `${OPS_*}` 占位符。
- `server.mode: release` 时强校验 JWT 长度、数据库密码/sslmode、CORS 白名单。
- 示例中若仍残留 `n9e.*`，视为未接线遗留，运行不依赖 N9E。

## 常用命令

```bash
make run
make build
make test
make fmt
make vet
make tidy
```

> 部分 Windows 环境 `go vet` 可能异常；可临时 `go test -vet=off ./...`。

## 关键模块

- `internal/server`：路由组装（`router.go`）
- `internal/handler` / `service` / `repository` / `model`
- `internal/vm`：VMOperator、Query、Route、VMAgent 相关
- `internal/logagent`：Vector 配置构建（读取 logs_collect_config）
- `internal/integration`：接入中心渲染与下发
- `internal/idempotency`：Redis 幂等

## 权限模型（摘要）

- 平台角色：`admin` / `user`
- 工作空间成员：`admin` / `member` / `viewer`（`ops_workspace_members`）
- 普通用户仅访问所属工作空间数据；平台 `admin` 可跨空间与平台运维接口
- 残余：部分模型列名 `tenant_id` = Workspace UUID

## 代表性接口

| 域 | 路径 |
|---|---|
| 工作空间 | `/api/v1/workspaces`、`.../members` |
| 告警 | `/api/v1/alerts/rules[/import]`、`/events`、`/channels`、`/stats/*` |
| Webhook | `POST /api/v1/webhooks/alertmanager` |
| 业务集群 | `/api/v1/business-clusters`、`.../collect-config`、`enable-logs` |
| UModel | `/api/v1/umodel/*` |
| Zone | `/api/v1/zones`、`init-shared|logs|grafana` |
| 平台扩容 | `/api/v1/platform/scaling/*` |

完整清单见 `docs/02-后端详细设计.md` 与 `internal/server/router.go`。
