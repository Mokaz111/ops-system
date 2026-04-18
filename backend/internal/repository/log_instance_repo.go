package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LogInstanceRepository 日志实例持久化。
type LogInstanceRepository struct {
	db *gorm.DB
}

func NewLogInstanceRepository(db *gorm.DB) *LogInstanceRepository {
	return &LogInstanceRepository{db: db}
}

func (r *LogInstanceRepository) Create(ctx context.Context, m *model.LogInstance) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *LogInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogInstance, error) {
	var m model.LogInstance
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *LogInstanceRepository) Update(ctx context.Context, m *model.LogInstance) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *LogInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.LogInstance{}, "id = ?", id).Error
}

// LogInstanceListFilter 列表筛选条件。
type LogInstanceListFilter struct {
	TenantID *uuid.UUID
	Status   string
	Keyword  string
	Offset   int
	Limit    int
}

func (r *LogInstanceRepository) List(ctx context.Context, f LogInstanceListFilter) ([]model.LogInstance, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LogInstance{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Keyword != "" {
		q = q.Where("instance_name ILIKE ?", "%"+f.Keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.LogInstance
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}
