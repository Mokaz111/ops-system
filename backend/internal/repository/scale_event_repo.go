package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScaleEventRepository 伸缩审计事件存储。
type ScaleEventRepository struct {
	db *gorm.DB
}

func NewScaleEventRepository(db *gorm.DB) *ScaleEventRepository {
	return &ScaleEventRepository{db: db}
}

// Create 写入一次事件。
func (r *ScaleEventRepository) Create(ctx context.Context, e *model.ScaleEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// ScaleEventListFilter 列表筛选。
type ScaleEventListFilter struct {
	InstanceID *uuid.UUID
	TenantID   *uuid.UUID
	ScaleType  string
	Status     string
	Offset     int
	Limit      int
}

// List 按 created_at DESC 分页查询。
func (r *ScaleEventRepository) List(ctx context.Context, f ScaleEventListFilter) ([]model.ScaleEvent, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ScaleEvent{})
	if f.InstanceID != nil {
		q = q.Where("instance_id = ?", *f.InstanceID)
	}
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.ScaleType != "" {
		q = q.Where("scale_type = ?", f.ScaleType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ScaleEvent
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}
