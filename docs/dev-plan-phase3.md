# 阶段三: 告警与实例管理开发详细任务

**开发周期**: 3-4周

## 3.1 实例管理模块 (第1周)

### 3.1.1 数据模型

```go
// internal/model/instance.go
type Instance struct {
    ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    TenantID       uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null"`
    InstanceName   string     `json:"instance_name" gorm:"type:varchar(255);not null"`
    InstanceType   string     `json:"instance_type" gorm:"type:varchar(50)"` // metrics, logs, visual, alert
    TemplateType   string     `json:"template_type" gorm:"type:varchar(50)"` // shared, dedicated_single, dedicated_cluster
    ReleaseName    string     `json:"release_name" gorm:"type:varchar(100)"`
    Namespace      string     `json:"namespace" gorm:"type:varchar(100)"`
    Spec           string     `json:"spec" gorm:"type:jsonb"` // 资源配置
    Status         string     `json:"status" gorm:"type:varchar(20);default:'creating'"`
    URL            string     `json:"url"` // 访问 URL
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}
```

### 3.1.2 实例 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/instances | 获取实例列表 |
| POST | /api/v1/instances | 创建实例 |
| GET | /api/v1/instances/:id | 获取实例详情 |
| PUT | /api/v1/instances/:id | 更新实例 |
| DELETE | /api/v1/instances/:id | 删除实例 |
| POST | /api/v1/instances/:id/scale | 扩容实例 |
| GET | /api/v1/instances/:id/metrics | 获取实例指标 |

### 3.1.3 任务清单

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T3.1.1 | Instance model 定义 | 2h |
| T3.1.2 | Instance repository | 4h |
| T3.1.3 | Instance handler | 4h |
| T3.1.4 | Instance service (创建/删除) | 6h |
| T3.1.5 | 扩缩容 service | 8h |
| T3.1.6 | 实例状态同步 (定期轮询 K8s) | 4h |

### 3.1.4 扩缩容实现

```go
// internal/service/scale.go
type ScaleService struct {
    helmClient *HelmClient
    k8sClient  *K8sClient
}

// 垂直扩容 (更新 CPU/内存/存储)
func (s *ScaleService) VerticalScale(ctx context.Context, instanceID string, spec *ScaleSpec) error {
    // 1. 获取当前 values
    currentValues, _ := s.helmClient.GetValues(ctx, instance.ReleaseName, instance.Namespace)
    
    // 2. 更新资源配置
    updateValues(currentValues, spec)
    
    // 3. helm upgrade
    return s.helmClient.UpgradeRelease(ctx, instance.ReleaseName, chart, namespace, updateValues)
}

// 水平扩容 (更新副本数)
func (s *ScaleService) HorizontalScale(ctx context.Context, instanceID string, replicas int) error {
    // 使用 kubectl scale 或 helm upgrade --set replicas
    return s.k8sClient.ScaleDeployment(ctx, instance.Namespace, instance.ReleaseName, replicas)
}

// 存储扩容
func (s *ScaleService) StorageScale(ctx context.Context, instanceID string, newSize string) error {
    // 更新 PVC 大小
    return s.k8sClient.ResizePVC(ctx, instance.Namespace, instance.ReleaseName+"-data", newSize)
}
```

## 3.2 告警规则管理 (第2周)

### 3.2.1 数据模型

```go
// internal/model/alert.go
type AlertRule struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    TenantID     uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null"`
    RuleName     string     `json:"rule_name" gorm:"type:varchar(255);not null"`
    RuleType     string     `json:"rule_type" gorm:"type:varchar(50)"` // metrics, logs
    Query        string     `json:"query" gorm:"type:text"` // PromQL / LogQL
    Condition    string     `json:"condition" gorm:"type:jsonb"` // {operator, threshold, for}
    Level        string     `json:"level" gorm:"type:varchar(20)"` // critical, warning, info
    Channels     string     `json:"channels" gorm:"type:jsonb"` // [dingtalk, email]
    Annotations  string     `json:"annotations" gorm:"type:text"`
    Enabled      bool       `json:"enabled" gorm:"default:true"`
    N9ERuleID    int64      `json:"n9e_rule_id"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}

type AlertEvent struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    TenantID     string     `json:"tenant_id" gorm:"type:varchar(100)"`
    RuleID       uuid.UUID  `json:"rule_id" gorm:"type:uuid"`
    RuleName     string     `json:"rule_name"`
    Level        string     `json:"level"`
    Status       string     `json:"status"` // firing, resolved
    StartTime    time.Time  `json:"start_time"`
    EndTime      *time.Time `json:"end_time"`
    Details      string     `json:"details" gorm:"type:jsonb"`
    CreatedAt    time.Time  `json:"created_at"`
}

type NotificationChannel struct {
    ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
    TenantID     uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null"`
    ChannelName  string     `json:"channel_name" gorm:"type:varchar(255)"`
    ChannelType  string     `json:"channel_type" gorm:"type:varchar(50)"` // dingtalk, email, slack, sms, webhook
    Config       string     `json:"config" gorm:"type:jsonb"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}
```

### 3.2.2 告警 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/alerts/rules | 获取告警规则列表 |
| POST | /api/v1/alerts/rules | 创建告警规则 |
| PUT | /api/v1/alerts/rules/:id | 更新告警规则 |
| DELETE | /api/v1/alerts/rules/:id | 删除告警规则 |
| GET | /api/v1/alerts/events | 获取告警事件 |
| GET | /api/v1/alerts/events/:id | 获取告警事件详情 |
| PUT | /api/v1/alerts/events/:id/ack | 确认告警 |
| GET | /api/v1/alerts/channels | 获取通知渠道 |
| POST | /api/v1/alerts/channels | 创建通知渠道 |
| PUT | /api/v1/alerts/channels/:id | 更新通知渠道 |
| DELETE | /api/v1/alerts/channels/:id | 删除通知渠道 |

### 3.2.3 任务清单

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T3.2.1 | AlertRule/AlertEvent/NotificationChannel model | 2h |
| T3.2.2 | 告警规则 CRUD | 6h |
| T3.2.3 | 告警事件查询 | 4h |
| T3.2.4 | 通知渠道 CRUD | 6h |
| T3.2.5 | N9E 规则同步 (创建/更新/删除) | 8h |

### 3.2.4 N9E 规则同步

```go
// internal/n9e/alert.go
type AlertSyncService struct {
    n9eClient *N9EClient
}

// 同步告警规则到 N9E
func (s *AlertSyncService) SyncRule(ctx context.Context, rule *model.AlertRule) error {
    n9eRule := map[string]interface{}{
        "name":        rule.RuleName,
        "note":        rule.Annotations,
        "severity":    toN9ESeverity(rule.Level),
        "prom_for":    rule.Condition.For,
        "prom_ql":     rule.Query,
        "enable":      rule.Enabled,
        "team_id":     rule.TenantID, // 关联团队
    }
    
    if rule.N9ERuleID == 0 {
        // 创建
        id, err := s.n9eClient.CreateRule(ctx, n9eRule)
        rule.N9ERuleID = id
        return err
    } else {
        // 更新
        return s.n9eClient.UpdateRule(ctx, rule.N9ERuleID, n9eRule)
    }
}

func toN9ESeverity(level string) int {
    switch level {
    case "critical": return 1
    case "warning": return 2
    case "info": return 3
    default: return 3
    }
}
```

## 3.3 告警通知模块 (第3周)

### 3.3.1 通知服务

```go
// internal/service/notify.go
type NotifyService struct {
    emailSender   *EmailSender
    dingtalkSender *DingtalkSender
    slackSender   *SlackSender
    smsSender     *SMSSender
    webhookSender *WebhookSender
}

// 发送告警通知
func (s *NotifyService) SendAlert(ctx context.Context, event *model.AlertEvent, channels []string) error {
    for _, channelType := range channels {
        switch channelType {
        case "email":
            s.sendEmail(ctx, event)
        case "dingtalk":
            s.sendDingtalk(ctx, event)
        case "slack":
            s.sendSlack(ctx, event)
        case "sms":
            s.sendSMS(ctx, event)
        case "webhook":
            s.sendWebhook(ctx, event)
        }
    }
    return nil
}
```

### 3.3.2 通知渠道实现

```go
// internal/service/notify/dingtalk.go
type DingtalkSender struct {
    webhookURL string
    secret     string
}

func (s *DingtalkSender) Send(ctx context.Context, msg *NotifyMessage) error {
    // 1. 生成签名
    timestamp := time.Now().Unix() * 1000
    sign := generateDingtalkSign(s.secret, timestamp)
    
    // 2. 构建请求
    payload := map[string]interface{}{
        "msgtype": "markdown",
        "markdown": map[string]string{
            "title": fmt.Sprintf("【%s】%s", msg.Level, msg.RuleName),
            "text":  msg.Content,
        },
        "at": map[string]interface{}{
            "isAtAll": msg.IsAtAll,
        },
    }
    
    // 3. 发送请求
    url := fmt.Sprintf("%s&timestamp=%d&sign=%s", s.webhookURL, timestamp, sign)
    return postJSON(ctx, url, payload)
}

// internal/service/notify/email.go
type EmailSender struct {
    smtpHost string
    smtpPort int
    username string
    password string
    from     string
}

func (s *EmailSender) Send(ctx context.Context, msg *NotifyMessage) error {
    // 构建邮件内容
    subject := fmt.Sprintf("【%s】%s", msg.Level, msg.RuleName)
    body := buildEmailBody(msg)
    
    // 发送邮件
    return smtp.SendMail(
        fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort),
        s.username, s.password,
        s.from, msg.Recipients,
        []byte(buildEmailMsg(s.from, msg.Recipients, subject, body)),
    )
}
```

### 3.3.3 任务清单

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T3.3.1 | 钉钉通知实现 | 4h |
| T3.3.2 | 邮件通知实现 | 4h |
| T3.3.3 | Slack 通知实现 | 2h |
| T3.3.4 | SMS 通知实现 | 2h |
| T3.3.5 | Webhook 通知实现 | 2h |
| T3.3.6 | 通知模板定制 | 4h |

## 3.4 告警事件与统计 (第4周)

### 3.4.1 告警事件处理

```go
// internal/service/alert_event.go
type AlertEventService struct {
    n9eClient  *N9EClient
    notifySvc  *NotifyService
    eventRepo  *AlertEventRepository
}

// 从 N9E 拉取告警事件
func (s *AlertEventService) SyncEvents(ctx context.Context) error {
    events, err := s.n9eClient.GetFiringEvents(ctx)
    if err != nil {
        return err
    }
    
    for _, event := range events {
        // 转换为内部模型
        alertEvent := convertToInternalEvent(event)
        
        // 存储到数据库
        s.eventRepo.Upsert(ctx, alertEvent)
        
        // 发送通知 (首次触发)
        if !alertEvent.Notified {
            channels := getChannelsForRule(alertEvent.RuleID)
            s.notifySvc.SendAlert(ctx, alertEvent, channels)
            alertEvent.Notified = true
            s.eventRepo.Update(ctx, alertEvent)
        }
    }
    
    // 处理已恢复的告警
    resolvedEvents := s.eventRepo.GetFiringEvents(ctx)
    for _, event := range resolvedEvents {
        if !isFiringInN9E(event.N9EEventID) {
            event.Status = "resolved"
            event.EndTime = time.Now()
            s.eventRepo.Update(ctx, event)
        }
    }
    
    return nil
}
```

### 3.4.2 告警统计 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/alerts/stats/summary | 告警概览 |
| GET | /api/v1/alerts/stats/trend | 告警趋势 |
| GET | /api/v1/alerts/stats/by-level | 按级别统计 |
| GET | /api/v1/alerts/stats/by-rule | 按规则统计 |

### 3.4.3 任务清单

| 任务 | 描述 | 预估工时 |
|------|------|---------|
| T3.4.1 | 告警事件同步 (定时任务) | 6h |
| T3.4.2 | 告警确认/关闭功能 | 4h |
| T3.4.3 | 告警统计接口 | 6h |
| T3.4.4 | 告警收敛逻辑 | 4h |
| T3.4.5 | 告警历史查询 | 4h |

## 3.5 阶段三验收标准

- [ ] 实例 CRUD 功能正常
- [ ] 实例扩缩容功能正常
- [ ] 告警规则 CRUD 功能正常
- [ ] 告警规则同步到 N9E 正常
- [ ] 告警事件存储和查询正常
- [ ] 告警通知 (钉钉/邮件) 发送正常
- [ ] 告警统计功能正常
- [ ] 告警确认/关闭功能正常