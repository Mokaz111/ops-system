package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlatformScaleAudit 平台级扩容审计日志。
type PlatformScaleAudit struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID       string    `json:"user_id" gorm:"type:varchar(100);index"`
	Username     string    `json:"username" gorm:"type:varchar(255);index"`
	Role         string    `json:"role" gorm:"type:varchar(50);index"`
	ClientIP     string    `json:"client_ip" gorm:"type:varchar(64)"`
	TargetID     string    `json:"target_id" gorm:"type:varchar(100);index"`
	DryRun       bool      `json:"dry_run"`
	Status       string    `json:"status" gorm:"type:varchar(20);index"` // success/failed/replayed
	SpecPatch    string    `json:"spec_patch" gorm:"type:jsonb"`
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}

func (PlatformScaleAudit) TableName() string {
	return "ops_platform_scale_audits"
}

func (a *PlatformScaleAudit) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
