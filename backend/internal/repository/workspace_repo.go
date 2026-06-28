package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceRepository 工作空间持久化。
type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// WorkspaceListFilter 列表筛选。
type WorkspaceListFilter struct {
	TemplateType string
	Status       string
	Keyword      string
	Offset       int
	Limit        int
}

// Create 创建工作空间。
func (r *WorkspaceRepository) Create(ctx context.Context, w *model.Workspace) error {
	return r.db.WithContext(ctx).Create(w).Error
}

// GetByID 按 ID。
func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Workspace, error) {
	var w model.Workspace
	err := r.db.WithContext(ctx).First(&w, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// GetBySlug 按 slug。
func (r *WorkspaceRepository) GetBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	var w model.Workspace
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// GetByVMUserID 按 VMuser 标识。
func (r *WorkspaceRepository) GetByVMUserID(ctx context.Context, vmuserID string) (*model.Workspace, error) {
	var w model.Workspace
	err := r.db.WithContext(ctx).Where("vm_user_id = ?", vmuserID).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// Update 更新。
func (r *WorkspaceRepository) Update(ctx context.Context, w *model.Workspace) error {
	return r.db.WithContext(ctx).Save(w).Error
}

// Delete 删除。
func (r *WorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Workspace{}, "id = ?", id).Error
}

// List 分页列表。
func (r *WorkspaceRepository) List(ctx context.Context, f WorkspaceListFilter) ([]model.Workspace, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Workspace{})
	if f.TemplateType != "" {
		q = q.Where("template_type = ?", f.TemplateType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("workspace_name ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Workspace
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
