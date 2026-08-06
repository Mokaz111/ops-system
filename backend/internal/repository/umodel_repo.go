package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EntityRepository struct {
	db *gorm.DB
}

func NewEntityRepository(db *gorm.DB) *EntityRepository { return &EntityRepository{db: db} }

type EntityListFilter struct {
	TenantID   uuid.UUID
	EntityType string
	Keyword    string
	Offset     int
	Limit      int
}

func (r *EntityRepository) Create(ctx context.Context, e *model.Entity) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *EntityRepository) Update(ctx context.Context, e *model.Entity) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *EntityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Entity, error) {
	var e model.Entity
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *EntityRepository) List(ctx context.Context, f EntityListFilter) ([]model.Entity, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Entity{}).Where("tenant_id = ?", f.TenantID)
	if f.EntityType != "" {
		q = q.Where("entity_type = ?", f.EntityType)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR display_name ILIKE ?", kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Entity
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *EntityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Entity{}, "id = ?", id).Error
}

type MetricSetRepository struct {
	db *gorm.DB
}

func NewMetricSetRepository(db *gorm.DB) *MetricSetRepository { return &MetricSetRepository{db: db} }

type MetricSetListFilter struct {
	TenantID  uuid.UUID
	Component string
	Keyword   string
	Offset    int
	Limit     int
}

func (r *MetricSetRepository) Create(ctx context.Context, m *model.MetricSet) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MetricSetRepository) Update(ctx context.Context, m *model.MetricSet) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *MetricSetRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MetricSet, error) {
	var m model.MetricSet
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MetricSetRepository) List(ctx context.Context, f MetricSetListFilter) ([]model.MetricSet, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MetricSet{}).Where("tenant_id = ?", f.TenantID)
	if f.Component != "" {
		q = q.Where("component = ?", f.Component)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR display_name ILIKE ?", kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.MetricSet
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MetricSetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.MetricSet{}, "id = ?", id).Error
}

type LogSetRepository struct {
	db *gorm.DB
}

func NewLogSetRepository(db *gorm.DB) *LogSetRepository { return &LogSetRepository{db: db} }

type LogSetListFilter struct {
	TenantID  uuid.UUID
	Component string
	Keyword   string
	Offset    int
	Limit     int
}

func (r *LogSetRepository) Create(ctx context.Context, l *model.LogSet) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *LogSetRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogSet, error) {
	var l model.LogSet
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (r *LogSetRepository) List(ctx context.Context, f LogSetListFilter) ([]model.LogSet, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LogSet{}).Where("tenant_id = ?", f.TenantID)
	if f.Component != "" {
		q = q.Where("component = ?", f.Component)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR display_name ILIKE ?", kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.LogSet
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *LogSetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.LogSet{}, "id = ?", id).Error
}

type DataLinkRepository struct {
	db *gorm.DB
}

func NewDataLinkRepository(db *gorm.DB) *DataLinkRepository { return &DataLinkRepository{db: db} }

func (r *DataLinkRepository) Create(ctx context.Context, d *model.DataLink) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DataLinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DataLink, error) {
	var d model.DataLink
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *DataLinkRepository) ListByEntity(ctx context.Context, tenantID, entityID uuid.UUID) ([]model.DataLink, error) {
	var items []model.DataLink
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entity_id = ? AND status = ?", tenantID, entityID, "active").
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *DataLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.DataLink{}, "id = ?", id).Error
}
