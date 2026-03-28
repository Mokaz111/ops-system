package handler

import (
	"time"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
)

type userPublic struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	DeptID    *uuid.UUID `json:"dept_id"`
	TenantID  *uuid.UUID `json:"tenant_id"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func toUserPublic(u *model.User) userPublic {
	return userPublic{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     u.Phone,
		DeptID:    u.DeptID,
		TenantID:  u.TenantID,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
