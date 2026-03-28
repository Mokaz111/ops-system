package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DepartmentRepository 部门持久化。
type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) Create(ctx context.Context, d *model.Department) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DepartmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Department, error) {
	var d model.Department
	err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *DepartmentRepository) Update(ctx context.Context, d *model.Department) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *DepartmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Department{}, "id = ?", id).Error
}

// List 分页列表，按创建时间倒序。
func (r *DepartmentRepository) List(ctx context.Context, offset, limit int) ([]model.Department, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.Department{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Department
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListAll 全量（用于部门树）。
func (r *DepartmentRepository) ListAll(ctx context.Context) ([]model.Department, error) {
	var rows []model.Department
	err := r.db.WithContext(ctx).Order("dept_name ASC").Find(&rows).Error
	return rows, err
}

// CountChildren 直接子部门数量。
func (r *DepartmentRepository) CountChildren(ctx context.Context, parentID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Department{}).Where("parent_id = ?", parentID).Count(&n).Error
	return n, err
}
