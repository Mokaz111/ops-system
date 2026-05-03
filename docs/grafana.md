## 可观测性平台 Grafana 服务功能设计

基于您的需求，我来设计一个完整的 Grafana 服务管理模块。

---

## 一、核心功能架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Grafana 服务管理平台                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 实例管理     │  │ 监控集成    │  │ 组织管理    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 配置管理     │  │ 权限控制    │  │ 健康巡检    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、详细功能模块

### 1. Grafana 实例生命周期管理

```go
type GrafanaInstance struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`        // 实例名称
    URL         string    `json:"url"`         // 访问地址
    Type        string    `json:"type"`        // managed(平台纳管) / external(外部接入)
    
    // 认证信息
    AdminUser   string    `json:"admin_user"`   // 管理员账号
    AdminPass   string    `json:"admin_pass"`   // 管理员密码（加密存储）
    APIToken    string    `json:"api_token"`    // 服务账号令牌
    
    // 状态信息
    Status      string    `json:"status"`       // active / inactive / error
    Version     string    `json:"version"`      // Grafana版本
    LastHealthCheck time.Time `json:"last_health_check"`
    
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**功能列表：**
- ✅ 新建平台托管实例（自动部署）
- ✅ 登记外部自建实例
- ✅ 一键登录跳转
- ✅ 实例健康检查
- ✅ 实例删除/注销

### 2. 监控集成（关联监控数据源）

```go
// 支持的数据源类型
const (
    DatasourcePrometheus   = "prometheus"
    DatasourceLoki         = "loki"
    DatasourceTempo        = "tempo"
    DatasourceMimir        = "mimir"
    DatasourceCloudWatch   = "cloudwatch"
    DatasourceElasticsearch = "elasticsearch"
)

type DatasourceConfig struct {
    Name         string            `json:"name"`
    Type         string            `json:"type"`
    URL          string            `json:"url"`
    Access       string            `json:"access"`      // proxy / direct
    IsDefault    bool              `json:"is_default"`
    BasicAuth    bool              `json:"basic_auth"`
    BasicAuthUser string           `json:"basic_auth_user"`
    BasicAuthPass string           `json:"basic_auth_pass"`
    JSONData     map[string]interface{} `json:"json_data"`
    SecureJSONData map[string]string    `json:"secure_json_data"`
}
```

**自动集成流程：**
```
创建实例 → 自动配置数据源 → 导入默认仪表盘 → 验证数据连通性
```

**预置数据源模板：**
| 数据源 | 用途 | 默认URL |
|--------|------|---------|
| Prometheus | 指标监控 | http://prometheus:9090 |
| Loki | 日志查询 | http://loki:3100 |
| Tempo | 链路追踪 | http://tempo:3200 |
| AlertManager | 告警管理 | http://alertmanager:9093 |

### 3. 组织管理

```go
type Organization struct {
    ID      int64  `json:"id"`
    Name    string `json:"name"`
    Role    string `json:"role"`    // Admin, Editor, Viewer
}

type Team struct {
    ID      int64    `json:"id"`
    Name    string   `json:"name"`
    Members []string `json:"members"`
}
```

**组织管理功能：**
- 📁 创建/切换组织
- 👥 团队管理（创建、成员管理）
- 👤 用户管理（邀请、角色分配）
- 🔐 权限策略配置

**常用API操作：**
```bash
# 创建组织
POST /api/orgs
{"name": "新组织"}

# 添加用户到组织
POST /api/orgs/:orgId/users
{"loginOrEmail": "user@example.com", "role": "Viewer"}

# 创建团队
POST /api/teams
{"name": "后端团队", "email": "backend@example.com"}
```

### 4. 配置管理

```go
type GrafanaConfig struct {
    // 基础配置
    Server   ServerConfig   `json:"server"`
    Auth     AuthConfig     `json:"auth"`
    Security SecurityConfig `json:"security"`
    
    // 告警配置
    Alerting AlertingConfig `json:"alerting"`
    
    // 插件配置
    Plugins  PluginsConfig  `json:"plugins"`
}

type ServerConfig struct {
    Domain      string `json:"domain"`
    RootURL     string `json:"root_url"`
    CertFile    string `json:"cert_file"`
    CertKey     string `json:"cert_key"`
    EnableGzip  bool   `json:"enable_gzip"`
}

type AuthConfig struct {
    DisableLoginForm   bool `json:"disable_login_form"`
    DisableSignoutMenu bool `json:"disable_signout_menu"`
    OAuthAutoLogin     bool `json:"oauth_auto_login"`
}
```

**配置管理功能：**
| 配置项 | 说明 | 可修改方式 |
|--------|------|------------|
| 域名/URL | 访问地址 | GUI / API |
| 认证方式 | LDAP/OAuth/SAML | API / 配置文件 |
| 告警配置 | 通知渠道、静默规则 | GUI / API |
| 日志级别 | debug/info/warn/error | API |
| 会话超时 | 登录有效期 | API |

### 5. 权限控制体系

```
平台层权限（由您的前端控制）
    ├── 创建Grafana实例权限
    ├── 删除Grafana实例权限
    └── 查看所有实例权限

Grafana层权限（由Grafana自身RBAC控制）
    ├── 组织管理员：管理用户、团队、数据源
    ├── 组织编辑者：创建/编辑仪表盘
    ├── 组织查看者：仅查看仪表盘
    └── 服务账号：API调用权限
```

### 6. 健康巡检与告警

```go
type HealthCheck struct {
    InstanceID  uuid.UUID `json:"instance_id"`
    Status      string    `json:"status"`    // healthy / degraded / unhealthy
    ResponseTime int64    `json:"response_time_ms"`
    
    // 详细检查项
    APIAvailable   bool `json:"api_available"`
    LoginAvailable bool `json:"login_available"`
    Datasources    []DatasourceHealth `json:"datasources"`
}

type DatasourceHealth struct {
    Name   string `json:"name"`
    Status string `json:"status"`  // ok / error
    Error  string `json:"error"`
}
```

**巡检策略：**
- ⏱ 每5分钟检查实例存活状态
- 📊 验证关键数据源连通性
- 🔔 异常时发送告警到平台

### 7. 仪表盘生命周期管理

```go
// 预置仪表盘模板
var PredefinedDashboards = []string{
    "kubernetes-cluster.json",     // K8S集群监控
    "node-exporter-full.json",     // 主机监控
    "prometheus-stats.json",       // Prometheus自身监控
    "alertmanager-overview.json",  // 告警概览
}

// 批量导入仪表盘
func ImportDashboards(instance *GrafanaInstance, dashboardDir string) error
```


---

## 三、前后端交互API设计

### 平台需要提供的API

```go
// 1. Grafana实例管理
POST   /api/grafana/instances          // 创建实例（新建或登记）
GET    /api/grafana/instances          // 列出所有实例
GET    /api/grafana/instances/:id      // 获取实例详情
PUT    /api/grafana/instances/:id      // 更新实例（密码/令牌）
DELETE /api/grafana/instances/:id      // 删除实例
POST   /api/grafana/instances/:id/login // 一键登录（返回跳转URL或session）

// 2. 数据源管理
GET    /api/grafana/instances/:id/datasources     // 列出数据源
POST   /api/grafana/instances/:id/datasources     // 添加数据源
DELETE /api/grafana/instances/:id/datasources/:uid // 删除数据源
POST   /api/grafana/instances/:id/datasources/test // 测试连接

// 3. 组织管理
GET    /api/grafana/instances/:id/orgs              // 列出组织
POST   /api/grafana/instances/:id/orgs              // 创建组织
PUT    /api/grafana/instances/:id/orgs/:orgId/users // 添加用户到组织

// 4. 配置管理
GET    /api/grafana/instances/:id/config    // 获取配置
PUT    /api/grafana/instances/:id/config    // 更新配置

// 5. 健康检查
GET    /api/grafana/instances/:id/health    // 获取健康状态
POST   /api/grafana/instances/:id/health/check // 手动触发检查
```

---

## 四、一键登录实现

```go
// 一键登录 - 使用保存的凭证直接登录
func (s *GrafanaService) OneClickLogin(instanceID uuid.UUID) (*LoginResponse, error) {
    instance := s.GetInstance(instanceID)
    
    // 方法1：直接返回登录URL（带上用户名密码的POST请求）
    loginURL := fmt.Sprintf("%s/login?user=%s&password=%s", 
        instance.URL, instance.AdminUser, instance.AdminPass)
    
    // 方法2：后端模拟登录，返回session cookie
    client := &http.Client{}
    data := url.Values{}
    data.Set("user", instance.AdminUser)
    data.Set("password", instance.AdminPass)
    
    resp, err := client.PostForm(instance.URL+"/login", data)
    if err != nil {
        return nil, err
    }
    
    // 提取session cookie
    var cookies []*http.Cookie
    for _, c := range resp.Cookies() {
        if c.Name == "grafana_session" {
            cookies = append(cookies, c)
        }
    }
    
    // 返回URL和cookie，前端可以使用document.cookie设置后跳转
    return &LoginResponse{
        RedirectURL: instance.URL,
        Cookies:     cookies,
    }, nil
}
```

---

## 五、前端界面设计建议

### 实例管理页面（整合版）

```
┌──────────────────────────────────────────────────────────┐
│  Grafana 服务管理                          [+ 新建实例]   │
│                                           [纳管实例]  │
├──────────────────────────────────────────────────────────┤
│  ┌─────────┬──────────┬────────┬──────────┬──────────┐ │
│  │ 实例名称 │ 类型      │ 状态    │ 规格     │ 创建时间│ 操作     │ │
│  ├─────────┼──────────┼────────┼──────────┼──────────┤ │
│  │ prod-g8 │ 平台托管   │ ● 健康 │ 10.4.0   │ │ [登录][更多]│ │
│  │ ext-old │ 外部接入   │ ● 健康 │ 9.5.2    │ │ [登录][更多]│ │
│  │ test-g8 │ 平台托管   │ ○ 异常 │ 10.4.0   │ │ [登录][更多]│ │
│  └─────────┴──────────┴────────┴──────────┴──────────┘ │
└──────────────────────────────────────────────────────────┘
```

### 实例详情页（Tab页）

```
┌──────────────────────────────────────────────────────────┐
│  prod-grafana                                     [编辑]  │
├──────────────────────────────────────────────────────────┤
│  [概览] [数据源] [组织] [配置] [健康检查]                 │
├──────────────────────────────────────────────────────────┤
│  概览信息                                               │
│  ┌────────────────────────────────────────────────────┐ │
│  │ URL: https://grafana.example.com                   │ │
│  │ 版本: v10.4.0 │ 状态: 健康 │ 最后检查: 2分钟前     │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  数据源列表                            [+ 添加数据源]    │
│  ┌────────────┬──────────┬──────────┬────────────┐    │
│  │ 名称       │ 类型      │ 默认     │ 状态       │    │
│  ├────────────┼──────────┼──────────┼────────────┤    │
│  │ Prometheus │ Prometheus│ ● 是    │ ● 正常     │    │
│  │ Loki       │ Loki      │ ○ 否    │ ● 正常     │    │
│  └────────────┴──────────┴──────────┴────────────┘    │
└──────────────────────────────────────────────────────────┘
```

---

## 六、技术实现要点

1. **密码安全**：使用 AES-256 加密存储 Grafana 密码和令牌
2. **网络连通**：平台需要能访问到所有 Grafana 实例（支持代理）
3. **会话管理**：一键登录时使用服务端代理或返回 session cookie
4. **错误处理**：Grafana API 调用失败时有清晰的错误提示
5. **审计日志**：记录所有对 Grafana 实例的操作