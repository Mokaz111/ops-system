package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceMember 工作空间成员（多对多）。
type WorkspaceMember struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID      `json:"workspace_id" gorm:"type:uuid;not null;uniqueIndex:uk_ws_member,where:deleted_at IS NULL"`
	UserID      uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:uk_ws_member,where:deleted_at IS NULL"`
	Role        string         `json:"role" gorm:"type:varchar(20);not null;default:member"` // admin / member / viewer
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkspaceMember) TableName() string { return "ops_workspace_members" }

func (m *WorkspaceMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
