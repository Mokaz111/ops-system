package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IntegrationTemplate 接入中心模版（逻辑本体）。
//
// name 只在活跃行内唯一（partial unique index）；软删除模板后可以再用同名重建，
// 否则相同的 schema 缺陷会让"下架后重建"失败。
type IntegrationTemplate struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null;uniqueIndex:uk_integration_tpl_name_active,where:deleted_at IS NULL"`
	DisplayName   string         `json:"display_name" gorm:"type:varchar(255)"`
	Category      string         `json:"category" gorm:"type:varchar(50);index"`   // monitor/db/middleware/infra/log/cloud
	Component     string         `json:"component" gorm:"type:varchar(100);index"` // node/mysql/redis/kafka...
	Description   string         `json:"description" gorm:"type:text"`
	Icon          string         `json:"icon" gorm:"type:varchar(255)"`
	LatestVersion string         `json:"latest_version" gorm:"type:varchar(50)"`
	Tags          string         `json:"tags" gorm:"type:jsonb"` // []string
	Status        string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedBy     string         `json:"created_by" gorm:"type:varchar(100)"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (IntegrationTemplate) TableName() string {
	return "ops_integration_templates"
}

// BeforeCreate 生成主键。
func (t *IntegrationTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// IntegrationTemplateVersion 模版版本快照。
// (TemplateID, Version) 组合唯一，避免同一模版被重复写入同一版本号。
type IntegrationTemplateVersion struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	TemplateID    uuid.UUID `json:"template_id" gorm:"type:uuid;not null;uniqueIndex:uk_tpl_version,priority:1"`
	Version       string    `json:"version" gorm:"type:varchar(50);not null;uniqueIndex:uk_tpl_version,priority:2"`
	CollectorSpec string    `json:"collector_spec" gorm:"type:jsonb"` // VMPodScrape/VMServiceScrape/VMAgent YAML 片段
	AlertSpec     string    `json:"alert_spec" gorm:"type:jsonb"`     // { vmrule, n9e, alert_targets: []string }
	DashboardSpec string    `json:"dashboard_spec" gorm:"type:jsonb"` // [Grafana dashboard JSON]
	Variables     string    `json:"variables" gorm:"type:jsonb"`      // 模版可配置变量定义
	Changelog     string    `json:"changelog" gorm:"type:text"`
	Signature     string    `json:"signature" gorm:"type:varchar(128)"`
	CreatedAt     time.Time `json:"created_at"`
}

// TableName 表名。
func (IntegrationTemplateVersion) TableName() string {
	return "ops_integration_template_versions"
}

// BeforeCreate 生成主键。
func (v *IntegrationTemplateVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
