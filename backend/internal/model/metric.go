package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Metric 指标库条目。
//
// (Component, Name) 组合唯一：Prometheus 指标名（如 up / requests_total）在不同
// component 下完全可能重名；早期的全局 uniqueIndex(name) 与真实数据冲突。
// 同时保留对 name 的非唯一索引，用于按名称关键字模糊搜索。
type Metric struct {
	ID                    uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name                  string         `json:"name" gorm:"type:varchar(255);not null;index;uniqueIndex:uk_metric_component_name,priority:2"`
	MetricType            string         `json:"metric_type" gorm:"type:varchar(20)"` // counter/gauge/histogram/summary
	Unit                  string         `json:"unit" gorm:"type:varchar(50)"`
	Component             string         `json:"component" gorm:"type:varchar(100);index;uniqueIndex:uk_metric_component_name,priority:1"`
	DescriptionCN         string         `json:"description_cn" gorm:"type:text"`
	DescriptionEN         string         `json:"description_en" gorm:"type:text"`
	Labels                string         `json:"labels" gorm:"type:jsonb"`   // [{name, description}]
	Examples              string         `json:"examples" gorm:"type:jsonb"` // [PromQL/MetricsQL]
	SourceTemplateID      *uuid.UUID     `json:"source_template_id" gorm:"type:uuid;index"`
	SourceTemplateVersion string         `json:"source_template_version" gorm:"type:varchar(50)"`
	ManualOverride        bool           `json:"manual_override" gorm:"default:false"`
	Tags                  string         `json:"tags" gorm:"type:jsonb"` // []string
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Metric) TableName() string {
	return "ops_metrics"
}

// BeforeCreate 生成主键。
func (m *Metric) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// MetricTemplateMapping 指标与模版的出现关联。
type MetricTemplateMapping struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	MetricID           uuid.UUID `json:"metric_id" gorm:"type:uuid;index;not null"`
	TemplateID         uuid.UUID `json:"template_id" gorm:"type:uuid;index;not null"`
	TemplateVersion    string    `json:"template_version" gorm:"type:varchar(50)"`
	AppearsInCollector bool      `json:"appears_in_collector"`
	AppearsInDashboard bool      `json:"appears_in_dashboard"`
	AppearsInAlert     bool      `json:"appears_in_alert"`
	DashboardPanels    string    `json:"dashboard_panels" gorm:"type:jsonb"` // [{dashboard_uid, panel_id, expr}]
	CreatedAt          time.Time `json:"created_at"`
}

// TableName 表名。
func (MetricTemplateMapping) TableName() string {
	return "ops_metric_template_mappings"
}

// BeforeCreate 生成主键。
func (m *MetricTemplateMapping) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
