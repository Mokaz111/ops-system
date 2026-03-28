package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceRepository 实例持久化（当前仅租户删除校验等）。
type InstanceRepository struct {
	db *gorm.DB
}

func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// CountByTenantID 某租户下实例数量。
func (r *InstanceRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Instance{}).Where("tenant_id = ?", tenantID).Count(&n).Error
	return n, err
}
