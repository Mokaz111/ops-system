package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GrafanaHost Grafana 主机注册表（平台共享 or 租户自带）。
type GrafanaHost struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Scope         string         `json:"scope" gorm:"type:varchar(20);index"` // platform / tenant
	TenantID      *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;index"`
	URL           string         `json:"url" gorm:"type:varchar(500)"`
	AdminUser     string         `json:"admin_user" gorm:"type:varchar(100)"`
	AdminTokenEnc string         `json:"-" gorm:"type:text"`
	Status        string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (GrafanaHost) TableName() string {
	return "ops_grafana_hosts"
}

// BeforeCreate 生成主键。
func (g *GrafanaHost) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
