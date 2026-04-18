package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant 租户。
//
// dept_id / vmuser_id 只在活跃行内唯一——软删除后，DeptID 或 VMUserID
// 可以被重新分配。普通的 uniqueIndex + 软删除会让 DB 在"重建租户"时
// 因残留行抛唯一键冲突，因此使用 partial unique index（WHERE deleted_at IS NULL）。
type Tenant struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantName   string         `json:"tenant_name" gorm:"type:varchar(255);not null"`
	DeptID       uuid.UUID      `json:"dept_id" gorm:"type:uuid;not null;uniqueIndex:uk_tenant_dept_active,where:deleted_at IS NULL"`
	VMUserID     string         `json:"vmuser_id" gorm:"type:varchar(100);uniqueIndex:uk_tenant_vmuser_active,where:deleted_at IS NULL"`
	VMUserKey    string         `json:"vmuser_key" gorm:"type:varchar(255)"`
	TemplateType string         `json:"template_type" gorm:"type:varchar(50)"` // shared/dedicated_single/dedicated_cluster
	QuotaConfig  string         `json:"quota_config" gorm:"type:jsonb"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:creating"`
	N9ETeamID    int64          `json:"n9e_team_id"`
	GrafanaOrgID int64          `json:"grafana_org_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Tenant) TableName() string {
	return "ops_tenants"
}

// BeforeCreate 生成主键。
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
