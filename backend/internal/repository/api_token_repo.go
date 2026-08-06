package repository

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APITokenRepository API Token 持久化。
type APITokenRepository struct {
	db *gorm.DB
}

func NewAPITokenRepository(db *gorm.DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// Create 创建 Token 记录。
func (r *APITokenRepository) Create(ctx context.Context, t *model.APIToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByPrefix 按 token 前缀查找。
func (r *APITokenRepository) GetByPrefix(ctx context.Context, prefix string) (*model.APIToken, error) {
	var t model.APIToken
	err := r.db.WithContext(ctx).Where("token_prefix = ?", prefix).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByID 按 ID 查找。
func (r *APITokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.APIToken, error) {
	var t model.APIToken
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// ListByUser 列出用户的 Token。
func (r *APITokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	var list []model.APIToken
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

// Delete 软删除 Token。
func (r *APITokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.APIToken{}, "id = ?", id).Error
}

// UpdateLastUsed 更新最近使用时间。
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.APIToken{}).
		Where("id = ?", id).
		Update("last_used_at", at).Error
}
