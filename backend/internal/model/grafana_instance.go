package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GrafanaInstance Grafana 纳管实例注册表（平台共享 or 租户自带）。
type GrafanaInstance struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Scope         string         `json:"scope" gorm:"type:varchar(20);index"` // platform / tenant
	TenantID      *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;index"`
	URL           string         `json:"url" gorm:"type:varchar(500)"`
	AdminUser     string         `json:"admin_user" gorm:"type:varchar(100)"`
	AdminPassword string         `json:"-" gorm:"type:varchar(255)"`
	AdminTokenEnc string         `json:"-" gorm:"type:text"`
	Status        string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (GrafanaInstance) TableName() string {
	return "ops_grafana_instances"
}

// BeforeCreate 生成主键。
func (g *GrafanaInstance) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
