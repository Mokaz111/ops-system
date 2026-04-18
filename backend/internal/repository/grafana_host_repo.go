package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GrafanaHostRepository Grafana 主机注册表持久化。
type GrafanaHostRepository struct {
	db *gorm.DB
}

func NewGrafanaHostRepository(db *gorm.DB) *GrafanaHostRepository {
	return &GrafanaHostRepository{db: db}
}

func (r *GrafanaHostRepository) Create(ctx context.Context, m *model.GrafanaHost) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *GrafanaHostRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.GrafanaHost, error) {
	var m model.GrafanaHost
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GrafanaHostRepository) Update(ctx context.Context, m *model.GrafanaHost) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GrafanaHostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.GrafanaHost{}, "id = ?", id).Error
}

// GrafanaHostListFilter 列表筛选条件。
type GrafanaHostListFilter struct {
	Scope    string
	TenantID *uuid.UUID
	Offset   int
	Limit    int
}

func (r *GrafanaHostRepository) List(ctx context.Context, f GrafanaHostListFilter) ([]model.GrafanaHost, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.GrafanaHost{})
	if f.Scope != "" {
		q = q.Where("scope = ?", f.Scope)
	}
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.GrafanaHost
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}
