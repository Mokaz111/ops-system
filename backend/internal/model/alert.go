package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertRule 告警规则。
type AlertRule struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	RuleName    string         `json:"rule_name" gorm:"type:varchar(255);not null"`
	RuleType    string         `json:"rule_type" gorm:"type:varchar(50)"`    // metrics, logs
	Query       string         `json:"query" gorm:"type:text"`               // PromQL / LogQL
	Condition   string         `json:"condition" gorm:"type:jsonb"`          // {operator, threshold, for}
	Level       string         `json:"level" gorm:"type:varchar(20)"`        // critical, warning, info
	Channels    string         `json:"channels" gorm:"type:jsonb"`           // channel IDs
	Annotations string         `json:"annotations" gorm:"type:text"`
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	N9ERuleID   int64          `json:"n9e_rule_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (AlertRule) TableName() string {
	return "ops_alert_rules"
}

// BeforeCreate 生成主键。
func (a *AlertRule) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// AlertEvent 告警事件。
type AlertEvent struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	RuleID    uuid.UUID  `json:"rule_id" gorm:"type:uuid;index"`
	RuleName  string     `json:"rule_name" gorm:"type:varchar(255)"`
	Level     string     `json:"level" gorm:"type:varchar(20)"`           // critical, warning, info
	Status    string     `json:"status" gorm:"type:varchar(20)"`          // firing, resolved, acknowledged
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Details   string     `json:"details" gorm:"type:jsonb"`
	Notified  bool       `json:"notified" gorm:"default:false"`
	AckedBy   *uuid.UUID `json:"acked_by" gorm:"type:uuid"`
	AckedAt   *time.Time `json:"acked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName 表名。
func (AlertEvent) TableName() string {
	return "ops_alert_events"
}

// BeforeCreate 生成主键。
func (e *AlertEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// NotificationChannel 通知渠道。
type NotificationChannel struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ChannelName string         `json:"channel_name" gorm:"type:varchar(255);not null"`
	ChannelType string         `json:"channel_type" gorm:"type:varchar(50)"` // dingtalk, email, slack, sms, webhook
	Config      string         `json:"config" gorm:"type:jsonb"`
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (NotificationChannel) TableName() string {
	return "ops_notification_channels"
}

// BeforeCreate 生成主键。
func (c *NotificationChannel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
