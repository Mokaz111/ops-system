package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 平台用户。
type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Username     string     `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string     `json:"-" gorm:"type:varchar(255);not null"`
	Email        string     `json:"email" gorm:"type:varchar(255)"`
	Phone        string     `json:"phone" gorm:"type:varchar(50)"`
	DeptID       *uuid.UUID `json:"dept_id" gorm:"type:uuid;index"`
	TenantID     *uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	Role         string     `json:"role" gorm:"type:varchar(20);default:user"` // admin/user
	Status       string     `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 表名。
func (User) TableName() string {
	return "users"
}

// BeforeCreate 生成主键。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
