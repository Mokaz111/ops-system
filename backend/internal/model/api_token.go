package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIToken 平台 API Token（哈希存储，明文仅生成时返回一次）。
type APIToken struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	TokenPrefix string         `json:"token_prefix" gorm:"type:varchar(16);not null;index"`
	TokenHash   string         `json:"-" gorm:"type:varchar(255);not null"`
	Scope       string         `json:"scope" gorm:"type:varchar(50);default:read_write"` // read / read_write
	ExpiresAt   *time.Time     `json:"expires_at"`
	LastUsedAt  *time.Time     `json:"last_used_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (APIToken) TableName() string { return "ops_api_tokens" }

func (t *APIToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
