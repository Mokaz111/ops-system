# Backend

Ops System 后端控制面服务，基于 Go + Gin + GORM，提供多租户管理、实例管理、平台级扩容与审计能力。

## 运行要求

- Go 1.25.8+
- PostgreSQL 15+
- Redis 7+（建议；用于平台扩容幂等）

## 启动

```bash
go mod tidy
cp configs/config.example.yaml configs/config.yaml   # 首次需要
# 填好 configs/config.yaml 中的 ${OPS_*} 或设置环境变量
make run
```

默认地址：`http://0.0.0.0:8080`

### 配置与秘钥

- `configs/config.example.yaml` 是 git 中跟踪的模板；**实际配置请写在 `configs/config.yaml` 并不要提交**。
- 所有敏感项（`database.password / jwt.secret / grafana.api_key / n9e.token` 等）都支持 `${OPS_*}` 占位符，由 `os.ExpandEnv` 展开。
- `server.mode: release` 时会额外强校验：
  - `jwt.secret` 长度必须 ≥ 32 且非仓库示例值；
  - `database.password` 非空且非仓库示例值；
  - `database.sslmode` 不能为 `disable`；
  - `cors.allowed_origins` 必须非空。

## 常用命令

```bash
make run
make build
make test
make fmt
make vet
make tidy
```

> 在部分 Windows 环境中 `go vet` 可能出现工具链异常；若仅做编译验证，可临时使用 `go test -vet=off ./...`。

## 关键模块

- `internal/server`：路由与服务组装
- `internal/handler`：HTTP handler
- `internal/service`：业务逻辑（含扩缩容策略与平台扩容）
- `internal/repository`：数据库访问
- `internal/k8s`：K8s 客户端封装
- `internal/idempotency`：Redis 幂等存储
- `internal/model`：数据模型（含平台扩容审计表）

## 权限模型（摘要）

- 普通用户：仅可访问自身租户/实例数据
- `admin`：可访问管理接口（如平台级扩容与审计查询）

## 平台扩容相关接口（admin）

- `POST /api/v1/platform/scaling/bootstrap/shared/init`
- `GET /api/v1/platform/scaling/vmcluster/targets`
- `POST /api/v1/platform/scaling/vmcluster`
- `GET /api/v1/platform/scaling/audits`

