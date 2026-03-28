package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Department 部门。
type Department struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	DeptName     string         `json:"dept_name" gorm:"type:varchar(255);not null"`
	ParentID     *uuid.UUID     `json:"parent_id" gorm:"type:uuid;index"`
	TenantID     *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;uniqueIndex"`
	LeaderUserID *uuid.UUID     `json:"leader_user_id" gorm:"type:uuid"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Department) TableName() string {
	return "departments"
}

// BeforeCreate 生成主键。
func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
