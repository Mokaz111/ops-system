package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClusterRepository K8s 集群注册表仓储。
type ClusterRepository struct {
	db *gorm.DB
}

// NewClusterRepository 构造。
func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

// Create 新增。
func (r *ClusterRepository) Create(ctx context.Context, m *model.Cluster) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新。
func (r *ClusterRepository) Update(ctx context.Context, m *model.Cluster) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete 软删。
func (r *ClusterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Cluster{}, "id = ?", id).Error
}

// GetByID 按 ID 查。
func (r *ClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Cluster, error) {
	var m model.Cluster
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// ClusterListFilter 列表筛选。
type ClusterListFilter struct {
	Status string
	Offset int
	Limit  int
}

// List 分页列出。
func (r *ClusterRepository) List(ctx context.Context, f ClusterListFilter) ([]model.Cluster, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Cluster{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	var rows []model.Cluster
	if err := q.Order("created_at desc").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
