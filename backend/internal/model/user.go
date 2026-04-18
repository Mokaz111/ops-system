package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 平台用户。
//
// username 只在活跃行内唯一；软删除后，同名用户可以重新创建。若未来需要
// "保留历史用户名防止冒名"，应改成独立审计表而不是依赖 uniqueIndex。
type User struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Username     string         `json:"username" gorm:"type:varchar(255);not null;uniqueIndex:uk_user_username_active,where:deleted_at IS NULL"`
	PasswordHash string         `json:"-" gorm:"type:varchar(255);not null"`
	Email        string         `json:"email" gorm:"type:varchar(255)"`
	Phone        string         `json:"phone" gorm:"type:varchar(50)"`
	DeptID       *uuid.UUID     `json:"dept_id" gorm:"type:uuid;index"`
	TenantID     *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;index"`
	Role         string         `json:"role" gorm:"type:varchar(20);default:user"` // admin/user
	Status       string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (User) TableName() string {
	return "ops_users"
}

// BeforeCreate 生成主键。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
