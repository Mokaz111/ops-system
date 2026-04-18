package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IntegrationInstallation 单个 VM 监控实例上对模版的安装记录。
type IntegrationInstallation struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TemplateID      uuid.UUID      `json:"template_id" gorm:"type:uuid;index;not null"`
	TemplateVersion string         `json:"template_version" gorm:"type:varchar(50)"`
	InstanceID      uuid.UUID      `json:"instance_id" gorm:"type:uuid;index;not null"`
	TenantID        uuid.UUID      `json:"tenant_id" gorm:"type:uuid;index;not null"`
	GrafanaHostID   *uuid.UUID     `json:"grafana_host_id" gorm:"type:uuid;index"`
	GrafanaOrgID    int64          `json:"grafana_org_id"`
	InstalledParts  string         `json:"installed_parts" gorm:"type:jsonb"` // ["collector","vmrule","dashboard"]
	Variables       string         `json:"variables" gorm:"type:jsonb"`
	Status          string         `json:"status" gorm:"type:varchar(20);default:pending"` // pending/success/failed/uninstalled
	InstalledBy     string         `json:"installed_by" gorm:"type:varchar(100)"`
	LastRevisionID  *uuid.UUID     `json:"last_revision_id" gorm:"type:uuid"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (IntegrationInstallation) TableName() string {
	return "ops_integration_installations"
}

// BeforeCreate 生成主键。
func (i *IntegrationInstallation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// IntegrationInstallationRevision 安装/升级/回滚/卸载每一次变更的审计快照。
type IntegrationInstallationRevision struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	InstallationID   uuid.UUID `json:"installation_id" gorm:"type:uuid;index;not null"`
	Version          string    `json:"version" gorm:"type:varchar(50)"`
	Action           string    `json:"action" gorm:"type:varchar(20)"` // install/upgrade/rollback/uninstall
	SpecDiff         string    `json:"spec_diff" gorm:"type:jsonb"`
	AppliedResources string    `json:"applied_resources" gorm:"type:jsonb"` // [{gvk, namespace, name}]
	Operator         string    `json:"operator" gorm:"type:varchar(100)"`
	Status           string    `json:"status" gorm:"type:varchar(20)"` // success/failed
	ErrorMessage     string    `json:"error_message" gorm:"type:text"`
	CreatedAt        time.Time `json:"created_at"`
}

// TableName 表名。
func (IntegrationInstallationRevision) TableName() string {
	return "ops_integration_installation_revisions"
}

// BeforeCreate 生成主键。
func (r *IntegrationInstallationRevision) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
