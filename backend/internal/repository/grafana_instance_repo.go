package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GrafanaInstanceRepository Grafana 纳管实例注册表持久化。
type GrafanaInstanceRepository struct {
	db *gorm.DB
}

func NewGrafanaInstanceRepository(db *gorm.DB) *GrafanaInstanceRepository {
	return &GrafanaInstanceRepository{db: db}
}

func (r *GrafanaInstanceRepository) Create(ctx context.Context, m *model.GrafanaInstance) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *GrafanaInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.GrafanaInstance, error) {
	var m model.GrafanaInstance
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GrafanaInstanceRepository) Update(ctx context.Context, m *model.GrafanaInstance) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GrafanaInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.GrafanaInstance{}, "id = ?", id).Error
}

// GrafanaInstanceListFilter 列表筛选条件。
type GrafanaInstanceListFilter struct {
	Source string
	ZoneID *uuid.UUID
	Offset int
	Limit  int
}

func (r *GrafanaInstanceRepository) List(ctx context.Context, f GrafanaInstanceListFilter) ([]model.GrafanaInstance, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.GrafanaInstance{})
	if f.Source != "" {
		q = q.Where("source = ?", f.Source)
	}
	if f.ZoneID != nil {
		q = q.Where("zone_id = ?", *f.ZoneID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.GrafanaInstance
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}
