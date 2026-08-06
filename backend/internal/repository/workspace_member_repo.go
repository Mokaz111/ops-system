package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceMemberRepository 工作空间成员持久化。
type WorkspaceMemberRepository struct {
	db *gorm.DB
}

func NewWorkspaceMemberRepository(db *gorm.DB) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

// Create 创建成员关系。
func (r *WorkspaceMemberRepository) Create(ctx context.Context, m *model.WorkspaceMember) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID 按 ID 查询。
func (r *WorkspaceMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkspaceMember, error) {
	var m model.WorkspaceMember
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Update 更新成员关系。
func (r *WorkspaceMemberRepository) Update(ctx context.Context, m *model.WorkspaceMember) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete 软删除成员关系。
func (r *WorkspaceMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.WorkspaceMember{}, "id = ?", id).Error
}

// ListByWorkspace 列出工作空间成员。
func (r *WorkspaceMemberRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]model.WorkspaceMember, error) {
	var list []model.WorkspaceMember
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Order("created_at ASC").
		Find(&list).Error
	return list, err
}

// GetByUserAndWorkspace 查询用户在指定工作空间的成员关系。
func (r *WorkspaceMemberRepository) GetByUserAndWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) (*model.WorkspaceMember, error) {
	var m model.WorkspaceMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// ListByUserID 列出用户所属的全部工作空间成员关系。
func (r *WorkspaceMemberRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.WorkspaceMember, error) {
	var list []model.WorkspaceMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&list).Error
	return list, err
}
