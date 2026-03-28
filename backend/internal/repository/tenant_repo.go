package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRepository 租户持久化。
type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// TenantListFilter 列表筛选。
type TenantListFilter struct {
	DeptID       *uuid.UUID
	TemplateType string
	Status       string
	Keyword      string
	Offset       int
	Limit        int
}

// Create 创建租户。
func (r *TenantRepository) Create(ctx context.Context, t *model.Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByID 按 ID。
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByDeptID 按部门（一部门最多一租户）。
func (r *TenantRepository) GetByDeptID(ctx context.Context, deptID uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.WithContext(ctx).Where("dept_id = ?", deptID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByVMUserID 按 VMuser 标识。
func (r *TenantRepository) GetByVMUserID(ctx context.Context, vmuserID string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.db.WithContext(ctx).Where("vmuser_id = ?", vmuserID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// Update 更新。
func (r *TenantRepository) Update(ctx context.Context, t *model.Tenant) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// Delete 删除。
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Tenant{}, "id = ?", id).Error
}

// List 分页列表。
func (r *TenantRepository) List(ctx context.Context, f TenantListFilter) ([]model.Tenant, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Tenant{})
	if f.DeptID != nil {
		q = q.Where("dept_id = ?", *f.DeptID)
	}
	if f.TemplateType != "" {
		q = q.Where("template_type = ?", f.TemplateType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("tenant_name ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Tenant
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CountByDeptID 绑定到该部门的租户数量。
func (r *TenantRepository) CountByDeptID(ctx context.Context, deptID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Tenant{}).Where("dept_id = ?", deptID).Count(&n).Error
	return n, err
}
