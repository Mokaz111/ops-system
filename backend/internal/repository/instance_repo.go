package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstanceRepository struct {
	db *gorm.DB
}

func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

func (r *InstanceRepository) Create(ctx context.Context, i *model.Instance) error {
	return r.db.WithContext(ctx).Create(i).Error
}

func (r *InstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Instance, error) {
	var i model.Instance
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&i).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &i, err
}

func (r *InstanceRepository) Update(ctx context.Context, i *model.Instance) error {
	return r.db.WithContext(ctx).Save(i).Error
}

func (r *InstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Instance{}).Error
}

type InstanceListFilter struct {
	TenantID     *uuid.UUID
	InstanceType string
	Status       string
	Keyword      string
	Offset       int
	Limit        int
}

func (r *InstanceRepository) List(ctx context.Context, f InstanceListFilter) ([]model.Instance, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Instance{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.InstanceType != "" {
		q = q.Where("instance_type = ?", f.InstanceType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("instance_name ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Instance
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

func (r *InstanceRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Instance{}).Where("tenant_id = ?", tenantID).Count(&n).Error
	return n, err
}

func (r *InstanceRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.Instance, error) {
	var list []model.Instance
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *InstanceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", id).Update("status", status).Error
}
