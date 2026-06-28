package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Workspace 工作空间（原 Tenant）—— 纯 SaaS 平台的组织单元。
//
// vmuser_id 仅在活跃行内唯一：partial unique index (WHERE deleted_at IS NULL)。
type Workspace struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceName     string         `json:"workspace_name" gorm:"type:varchar(255);not null"`
	Slug              string         `json:"slug" gorm:"type:varchar(120);index"`
	VMUserID          string         `json:"vmuser_id" gorm:"type:varchar(100);uniqueIndex:uk_ws_vmuser_active,where:deleted_at IS NULL"`
	VMUserKey         string         `json:"vmuser_key" gorm:"type:varchar(255)"`
	TemplateType      string         `json:"template_type" gorm:"type:varchar(50)"`
	QuotaConfig       string         `json:"quota_config" gorm:"type:jsonb"`
	IsolationLevel    string         `json:"isolation_level" gorm:"type:varchar(30);default:shared"`
	VMNamespace       string         `json:"vm_namespace" gorm:"type:varchar(120)"`
	VMSelectURL       string         `json:"vm_select_url" gorm:"type:text"`
	VMInsertURL       string         `json:"vm_insert_url" gorm:"type:text"`
	Status            string         `json:"status" gorm:"type:varchar(20);default:creating"`
	N9ETeamID         int64          `json:"n9e_team_id"`
	GrafanaOrgID      int64          `json:"grafana_org_id"`
	GrafanaInstanceID *uuid.UUID     `json:"grafana_instance_id" gorm:"type:uuid;index"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Workspace) TableName() string {
	return "ops_workspaces"
}

func (w *Workspace) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
