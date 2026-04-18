package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Department 部门。
//
// tenant_id 只在活跃行内唯一：软删除后允许同一 tenant 重新绑定到其它 dept。
// 注：普通的 uniqueIndex 在 Postgres 里对多个 NULL 互不相等，单字段仍允许多条
// tenant_id=NULL 的记录，因此此处的 WHERE 子句只影响"tenant_id 非空时的唯一性"。
type Department struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	DeptName     string         `json:"dept_name" gorm:"type:varchar(255);not null"`
	ParentID     *uuid.UUID     `json:"parent_id" gorm:"type:uuid;index"`
	TenantID     *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;uniqueIndex:uk_dept_tenant_active,where:deleted_at IS NULL"`
	LeaderUserID *uuid.UUID     `json:"leader_user_id" gorm:"type:uuid"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Department) TableName() string {
	return "ops_departments"
}

// BeforeCreate 生成主键。
func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
