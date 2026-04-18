package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MetricRepository 指标持久化。
type MetricRepository struct {
	db *gorm.DB
}

func NewMetricRepository(db *gorm.DB) *MetricRepository {
	return &MetricRepository{db: db}
}

func (r *MetricRepository) Create(ctx context.Context, m *model.Metric) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MetricRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Metric, error) {
	var m model.Metric
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MetricRepository) GetByName(ctx context.Context, name string) (*model.Metric, error) {
	var m model.Metric
	err := r.db.WithContext(ctx).First(&m, "name = ?", name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MetricRepository) Update(ctx context.Context, m *model.Metric) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *MetricRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Metric{}, "id = ?", id).Error
}

// MetricListFilter 列表筛选条件。
type MetricListFilter struct {
	Component  string
	TemplateID *uuid.UUID
	Keyword    string
	Offset     int
	Limit      int
}

func (r *MetricRepository) List(ctx context.Context, f MetricListFilter) ([]model.Metric, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Metric{})
	if f.Component != "" {
		q = q.Where("component = ?", f.Component)
	}
	if f.TemplateID != nil {
		q = q.Where("source_template_id = ?", *f.TemplateID)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR description_cn ILIKE ? OR description_en ILIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Metric
	err := q.Order("name ASC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

// UpsertMapping 幂等写入指标-模版映射。
func (r *MetricRepository) UpsertMapping(ctx context.Context, m *model.MetricTemplateMapping) error {
	return r.db.WithContext(ctx).
		Where("metric_id = ? AND template_id = ? AND template_version = ?", m.MetricID, m.TemplateID, m.TemplateVersion).
		Assign(m).FirstOrCreate(m).Error
}

// ListMappingsByMetric 指标维度查询关联模版。
func (r *MetricRepository) ListMappingsByMetric(ctx context.Context, metricID uuid.UUID) ([]model.MetricTemplateMapping, error) {
	var list []model.MetricTemplateMapping
	err := r.db.WithContext(ctx).Where("metric_id = ?", metricID).Find(&list).Error
	return list, err
}

// ListMappingsByTemplate 模版维度查询关联指标。
func (r *MetricRepository) ListMappingsByTemplate(ctx context.Context, templateID uuid.UUID) ([]model.MetricTemplateMapping, error) {
	var list []model.MetricTemplateMapping
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&list).Error
	return list, err
}
