package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrFirstUserAlreadyExists = errors.New("first user already exists")

// UserRepository 用户持久化。
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// UserListFilter 列表筛选。
type UserListFilter struct {
	DeptID   *uuid.UUID
	TenantID *uuid.UUID
	Role     string
	Status   string
	Keyword  string
	Offset   int
	Limit    int
}

// Create 创建用户。
func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByID 按 ID。
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetByUsername 按用户名。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// Update 更新。
func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

// Delete 删除。
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

// List 分页列表。
func (r *UserRepository) List(ctx context.Context, f UserListFilter) ([]model.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.User{})
	if f.DeptID != nil {
		q = q.Where("dept_id = ?", *f.DeptID)
	}
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.Role != "" {
		q = q.Where("role = ?", f.Role)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("username ILIKE ? OR email ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.User
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListByDeptID 某部门下的用户分页列表。
func (r *UserRepository) ListByDeptID(ctx context.Context, deptID uuid.UUID, offset, limit int) ([]model.User, int64, error) {
	return r.List(ctx, UserListFilter{DeptID: &deptID, Offset: offset, Limit: limit})
}

// Count 用户总数。
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&n).Error
	return n, err
}

// CreateFirstUser 原子创建首个用户（并发安全）。
func (r *UserRepository) CreateFirstUser(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// PostgreSQL 表级锁，避免并发 bootstrap 同时通过空表检查。
		if err := tx.Exec("LOCK TABLE ops_users IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Model(&model.User{}).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrFirstUserAlreadyExists
		}
		return tx.Create(u).Error
	})
}
