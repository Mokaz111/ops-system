package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository 用户持久化。
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// ListByDeptID 某部门下的用户分页列表。
func (r *UserRepository) ListByDeptID(ctx context.Context, deptID uuid.UUID, offset, limit int) ([]model.User, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.User{}).Where("dept_id = ?", deptID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.User
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
