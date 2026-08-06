package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BusinessClusterRepository 业务集群仓储。
type BusinessClusterRepository struct {
	db *gorm.DB
}

func NewBusinessClusterRepository(db *gorm.DB) *BusinessClusterRepository {
	return &BusinessClusterRepository{db: db}
}

func (r *BusinessClusterRepository) Create(ctx context.Context, m *model.BusinessCluster) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *BusinessClusterRepository) Update(ctx context.Context, m *model.BusinessCluster) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *BusinessClusterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.BusinessCluster{}, "id = ?", id).Error
}

func (r *BusinessClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.BusinessCluster, error) {
	var m model.BusinessCluster
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// BusinessClusterListFilter 列表筛选。
type BusinessClusterListFilter struct {
	TenantID   string
	InstanceID string
	Offset     int
	Limit      int
}

func (r *BusinessClusterRepository) List(ctx context.Context, f BusinessClusterListFilter) ([]model.BusinessCluster, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.BusinessCluster{})
	if f.TenantID != "" {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if f.InstanceID != "" {
		q = q.Where("instance_id = ?", f.InstanceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	var rows []model.BusinessCluster
	if err := q.Order("created_at desc").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *BusinessClusterRepository) CountByInstanceID(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.BusinessCluster{}).Where("instance_id = ?", instanceID).Count(&n).Error
	return n, err
}
