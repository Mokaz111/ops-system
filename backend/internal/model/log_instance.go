package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LogInstance VictoriaLogs 日志实例。
type LogInstance struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID      uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	InstanceName  string         `json:"instance_name" gorm:"type:varchar(255);not null"`
	ReleaseName   string         `json:"release_name" gorm:"type:varchar(100)"`
	Namespace     string         `json:"namespace" gorm:"type:varchar(100)"`
	Endpoint      string         `json:"endpoint" gorm:"type:varchar(255)"`
	Token         string         `json:"token" gorm:"type:varchar(255)"`
	RetentionDays int            `json:"retention_days"`
	Spec          string         `json:"spec" gorm:"type:jsonb"`
	Status        string         `json:"status" gorm:"type:varchar(20);default:creating"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (LogInstance) TableName() string {
	return "ops_log_instances"
}

// BeforeCreate 生成主键。
func (l *LogInstance) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
