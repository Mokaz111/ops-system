# 阶段二: 后端核心开发详细任务

**开发周期**: 4-6周

## 2.1 项目骨架搭建 (第1周)

### 2.1.1 项目初始化

```
backend/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── server/server.go
│   ├── server/router.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── logging.go
│   └── model/
├── pkg/utils/
├── go.mod
├── go.sum
└── Makefile
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.1.1 | 初始化 Go module，添加依赖 (gin, gorm, jwt, client-go, helm) | 2h |
| T2.1.2 | 配置管理 (config.yaml + viper) | 4h |
| T2.1.3 | 项目目录结构创建 | 2h |
| T2.1.4 | 日志初始化 (zap) | 2h |
| T2.1.5 | HTTP 服务框架搭建 (Gin) | 4h |
| T2.1.6 | 中间件集成 (CORS, Recovery, Logger) | 2h |

**关键代码片段**:

```go
// config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"

database:
  host: "postgresql"
  port: 5432
  user: "postgres"
  password: "${DB_PASSWORD}"
  name: "monitoring"

redis:
  host: "redis"
  port: 6379

kubernetes:
  incluster: false
  kubeconfig: "/etc/kubernetes/admin.conf"

helm:
  repos:
    - name: vm
      url: https://victoriametrics.github.io/helm-charts/
    - name: grafana
      url: https://grafana.github.io/helm-charts/
```

### 2.1.2 数据库连接

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.1.7 | GORM 初始化，数据库连接 | 2h |
| T2.1.8 | 自动迁移脚本 (department, tenant, user, instance) | 4h |
| T2.1.9 | 数据库连接池配置 | 2h |

## 2.2 用户/部门/租户模块 (第2周)

### 2.2.1 数据模型定义

```go
// internal/model/department.go
type Department struct {
    ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    DeptName    string     `json:"dept_name" gorm:"type:varchar(255);not null"`
    ParentID    *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
    TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;uniqueIndex"`
    LeaderUserID *uuid.UUID `json:"leader_user_id" gorm:"type:uuid"`
    Status      string     `json:"status" gorm:"type:varchar(20);default:'active'"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// internal/model/tenant.go
type Tenant struct {
    ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    TenantName    string     `json:"tenant_name" gorm:"type:varchar(255);not null"`
    DeptID        uuid.UUID  `json:"dept_id" gorm:"type:uuid;uniqueIndex"`
    VMUserID      string     `json:"vmuser_id" gorm:"type:varchar(100);uniqueIndex"`
    VMUserKey     string     `json:"vmuser_key" gorm:"type:varchar(255)"`
    TemplateType  string     `json:"template_type" gorm:"type:varchar(50)"` // shared/dedicated_single/dedicated_cluster
    QuotaConfig   string     `json:"quota_config" gorm:"type:jsonb"`
    Status        string     `json:"status" gorm:"type:varchar(20);default:'creating'"`
    N9ETeamID     int64      `json:"n9e_team_id"`
    GrafanaOrgID  int64      `json:"grafana_org_id"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

// internal/model/user.go
type User struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    Username     string     `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
    PasswordHash string     `json:"-" gorm:"type:varchar(255);not null"`
    Email        string     `json:"email" gorm:"type:varchar(255)"`
    Phone        string     `json:"phone" gorm:"type:varchar(50)"`
    DeptID       *uuid.UUID `json:"dept_id" gorm:"type:uuid"`
    TenantID     *uuid.UUID `json:"tenant_id" gorm:"type:uuid"`
    Role         string     `json:"role" gorm:"type:varchar(20);default:'user'"` // admin/user
    Status       string     `json:"status" gorm:"type:varchar(20);default:'active'"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.2.1 | 定义 Department/Tenant/User model | 4h |
| T2.2.2 | 部门 repository 层 | 4h |
| T2.2.3 | 租户 repository 层 | 4h |
| T2.2.4 | 用户 repository 层 | 4h |

### 2.2.2 部门 API

**API 接口**:

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/departments | 获取部门列表 |
| GET | /api/v1/departments/tree | 获取部门树 |
| POST | /api/v1/departments | 创建部门 |
| GET | /api/v1/departments/:id | 获取部门详情 |
| PUT | /api/v1/departments/:id | 更新部门 |
| DELETE | /api/v1/departments/:id | 删除部门 |

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.2.5 | 部门 handler | 4h |
| T2.2.6 | 部门 service | 4h |
| T2.2.7 | 部门 router 注册 | 2h |

### 2.2.3 租户 API

**API 接口**:

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/tenants | 获取租户列表 |
| POST | /api/v1/tenants | 创建租户 |
| GET | /api/v1/tenants/:id | 获取租户详情 |
| PUT | /api/v1/tenants/:id | 更新租户 |
| DELETE | /api/v1/tenants/:id | 删除租户 |
| GET | /api/v1/tenants/:id/metrics | 获取租户资源使用 |

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.2.8 | 租户 handler | 4h |
| T2.2.9 | 租户 service (创建租户核心逻辑) | 8h |
| T2.2.10 | 租户 router 注册 | 2h |

### 2.2.4 用户 API

**API 接口**:

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/users | 获取用户列表 |
| POST | /api/v1/users | 创建用户 |
| POST | /api/v1/auth/login | 用户登录 |
| GET | /api/v1/users/:id | 获取用户详情 |
| PUT | /api/v1/users/:id | 更新用户 |
| DELETE | /api/v1/users/:id | 删除用户 |

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.2.11 | 用户 handler | 4h |
| T2.2.12 | 用户 service | 4h |
| T2.2.13 | 登录认证 (JWT) | 4h |
| T2.2.14 | 用户 router 注册 | 2h |

## 2.3 VMuser 同步模块 (第2-3周)

### 2.3.1 VM Operator 集成

```go
// internal/vm/operator.go
type VMOperatorClient struct {
    client *client.Client
}

func NewVMOperatorClient(config *rest.Config) (*VMOperatorClient, error) {
    c, err := client.New(config, client.Options{})
    if err != nil {
        return nil, err
    }
    return &VMOperatorClient{client: c}, nil
}

// 创建 VMAuth 用户
func (c *VMOperatorClient) UpdateVMAuthUsers(ctx context.Context, users []VMAuthUser) error {
    // 通过 VMAuth CRD 或直接调用 vmauth API
}
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.3.1 | VM Operator client 初始化 | 4h |
| T2.3.2 | VMAuth 用户管理 (添加/删除) | 6h |
| T2.3.3 | VMuser API Key 生成 | 2h |
| T2.3.4 | 配额管理 (QPS/Series) | 4h |

### 2.3.2 vmauth 配置

```go
// internal/vm/vmauth.go
type VMAuthService struct {
    endpoint string // http://vmauth.platform:8421
}

func (s *VMAuthService) AddUser(ctx context.Context, vmuserID, apiKey string) error {
    // 调用 vmauth API 添加用户
    // POST /api/v1/auth/users
}

// 获取租户访问端点
func (s *VMAuthService) GetAccessEndpoint(vmuserID string) string {
    return fmt.Sprintf("http://vmauth.platform:8421/insert/%s", vmuserID)
}
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.3.5 | vmauth HTTP client | 4h |
| T2.3.6 | 用户创建/删除同步 | 4h |
| T2.3.7 | 访问端点生成 | 2h |

## 2.4 N9E 同步模块 (第3周)

### 2.4.1 N9E Client

```go
// internal/n9e/client.go
type N9EClient struct {
    baseURL  string
    username string
    password string
    token    string
}

// 创建团队
func (c *N9EClient) CreateTeam(ctx context.Context, name, note string) (int64, error) {
    // POST /api/n9e/teams
}

// 创建用户
func (c *N9EClient) CreateUser(ctx context.Context, user *N9EUser) error {
    // POST /api/n9e/users
}

// 创建数据源
func (c *N9EClient) CreateDatasource(ctx context.Context, tenantID string, ds *N9EDatasource) error {
    // POST /api/n9e/datasources
}
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.4.1 | N9E client 初始化 | 4h |
| T2.4.2 | 团队管理 (创建/删除) | 4h |
| T2.4.3 | 用户同步 | 4h |
| T2.4.4 | 数据源注册 | 4h |
| T2.4.5 | 告警规则同步 | 4h |

## 2.5 Grafana 同步模块 (第3周)

### 2.5.1 Grafana Client

```go
// internal/grafana/client.go
type GrafanaClient struct {
    baseURL string
    apiKey  string
}

// 创建组织
func (c *GrafanaClient) CreateOrg(ctx context.Context, org *GrafanaOrg) (int64, error) {
    // POST /api/orgs
}

// 添加组织用户
func (c *GrafanaClient) AddOrgUser(ctx context.Context, orgID int64, user *GrafanaOrgUser) error {
    // POST /api/orgs/:id/users
}

// 设置组织管理员
func (c *GrafanaClient) SetOrgAdmin(ctx context.Context, orgID int64, userID string) error {
    // PUT /api/orgs/:id/users/:userId
}

// 创建数据源
func (c *GrafanaClient) CreateDatasource(ctx context.Context, orgID int64, ds *GrafanaDatasource) error {
    // POST /api/datasources
}
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.5.1 | Grafana client 初始化 | 4h |
| T2.5.2 | 组织管理 (创建/删除) | 4h |
| T2.5.3 | 用户同步 | 4h |
| T2.5.4 | 数据源配置 | 6h |
| T2.5.5 | Dashboard 模板同步 | 4h |

## 2.6 Helm 编排模块 (第4周)

### 2.6.1 Helm Client

```go
// internal/helm/client.go
type HelmClient struct {
    kubeconfig string
    repos      map[string]string
}

func NewHelmClient(kubeconfig string) (*HelmClient, error) {
    // 初始化 Helm client
}

// 安装 Release
func (c *HelmClient) InstallRelease(ctx context.Context, name, chart, namespace string, values map[string]interface{}) error {
    // helm install
}

// 升级 Release
func (c *HelmClient) UpgradeRelease(ctx context.Context, name, chart, namespace string, values map[string]interface{}) error {
    // helm upgrade
}

// 卸载 Release
func (c *HelmClient) UninstallRelease(ctx context.Context, name, namespace string) error {
    // helm uninstall
}

// 获取 Release 状态
func (c *HelmClient) GetReleaseStatus(ctx context.Context, name, namespace string) (*ReleaseStatus, error) {
    // helm list
}
```

### 2.6.2 Values 模板

```
internal/helm/values/
├── vm-shared.yaml      # 共享版 (仅 VL + n9e-edge)
├── vm-single.yaml      # 独享单节点版
├── vm-cluster.yaml     # 独享集群版
├── vl.yaml             # VictoriaLogs
├── grafana.yaml        # Grafana
└── n9e-edge.yaml       # n9e-edge
```

**任务清单**:

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T2.6.1 | Helm client 初始化 | 4h |
| T2.6.2 | Helm values 模板编写 | 8h |
| T2.6.3 | Release 安装/升级/卸载 | 6h |
| T2.6.4 | K8s Namespace 管理 | 4h |
| T2.6.5 | ResourceQuota 配置 | 4h |
| T2.6.6 | NetworkPolicy 配置 | 4h |

### 2.6.3 编排服务

```go
// internal/service/orchestrator.go
type OrchestratorService struct {
    helmClient    *HelmClient
    k8sClient     *K8sClient
    vmClient      *VMOperatorClient
}

func (s *OrchestratorService) DeployTenant(ctx context.Context, tenant *model.Tenant) error {
    // 1. 创建 Namespace
    // 2. 配置 ResourceQuota
    // 3. 根据模板类型部署
    // 4. 部署 n9e-edge
}

func (s *OrchestratorService) DeleteTenant(ctx context.Context, tenant *model.Tenant) error {
    // 1. 删除 Helm Release
    // 2. 删除 Namespace
}
```

## 2.7 阶段二验收标准

- [ ] 部门 CRUD 功能正常
- [ ] 租户 CRUD 功能正常
- [ ] 用户 CRUD + 登录认证正常
- [ ] 创建租户自动同步 VMuser
- [ ] 创建租户自动同步 N9E 团队
- [ ] 创建租户自动同步 Grafana 组织
- [ ] 创建租户自动部署监控实例
- [ ] API 单元测试覆盖率 > 60%