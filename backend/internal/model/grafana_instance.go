package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GrafanaInstance Grafana 实例连接信息（平台统一管理）。
// source: platform（Zone 自动部署）/ external（管理员手动登记）。
type GrafanaInstance struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Source        string         `json:"source" gorm:"type:varchar(20);default:external;index"` // platform / external
	ZoneID        *uuid.UUID     `json:"zone_id" gorm:"type:uuid;index"`
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
