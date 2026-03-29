package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationChannelRepository 通知渠道持久化。
type NotificationChannelRepository struct {
	db *gorm.DB
}

func NewNotificationChannelRepository(db *gorm.DB) *NotificationChannelRepository {
	return &NotificationChannelRepository{db: db}
}

// NotificationChannelListFilter 列表筛选。
type NotificationChannelListFilter struct {
	TenantID    *uuid.UUID
	ChannelType string
	Keyword     string
	Offset      int
	Limit       int
}

// Create 创建通知渠道。
func (r *NotificationChannelRepository) Create(ctx context.Context, ch *model.NotificationChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

// GetByID 按 ID。
func (r *NotificationChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error) {
	var ch model.NotificationChannel
	err := r.db.WithContext(ctx).First(&ch, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

// Update 更新。
func (r *NotificationChannelRepository) Update(ctx context.Context, ch *model.NotificationChannel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

// Delete 软删除。
func (r *NotificationChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.NotificationChannel{}, "id = ?", id).Error
}

// List 分页列表。
func (r *NotificationChannelRepository) List(ctx context.Context, f NotificationChannelListFilter) ([]model.NotificationChannel, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.NotificationChannel{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.ChannelType != "" {
		q = q.Where("channel_type = ?", f.ChannelType)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("channel_name ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.NotificationChannel
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListByTenantID 查询某租户所有渠道。
func (r *NotificationChannelRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.NotificationChannel, error) {
	var rows []model.NotificationChannel
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// GetByIDs 批量按 ID 查询。
func (r *NotificationChannelRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]model.NotificationChannel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.NotificationChannel
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}
