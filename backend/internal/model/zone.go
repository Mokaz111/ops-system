package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Zone 可用区，一个 Zone 对应一个 K8s 监控集群。
type Zone struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Slug        string         `json:"slug" gorm:"type:varchar(50);not null;uniqueIndex:uk_zone_slug_active,where:deleted_at IS NULL"`
	DisplayName string         `json:"display_name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	ClusterID   uuid.UUID      `json:"cluster_id" gorm:"type:uuid;not null;index"`
	Endpoint    string         `json:"endpoint" gorm:"type:varchar(500)"`
	Labels      string         `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Capacity    string         `json:"capacity" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:creating"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Zone) TableName() string { return "ops_zones" }

func (z *Zone) BeforeCreate(tx *gorm.DB) error {
	if z.ID == uuid.Nil {
		z.ID = uuid.New()
	}
	return nil
}
