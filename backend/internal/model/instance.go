package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Instance 监控实例（指标/日志/可视化/告警等）。
type Instance struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID     uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	InstanceName string         `json:"instance_name" gorm:"type:varchar(255);not null"`
	InstanceType string         `json:"instance_type" gorm:"type:varchar(50)"`  // metrics, logs, visual, alert
	TemplateType string         `json:"template_type" gorm:"type:varchar(50)"`  // shared, dedicated_single, dedicated_cluster
	ReleaseName  string         `json:"release_name" gorm:"type:varchar(100)"`
	Namespace    string         `json:"namespace" gorm:"type:varchar(100)"`
	Spec         string         `json:"spec" gorm:"type:jsonb"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:creating"`
	URL          string         `json:"url"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Instance) TableName() string {
	return "instances"
}

// BeforeCreate 生成主键。
func (i *Instance) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
