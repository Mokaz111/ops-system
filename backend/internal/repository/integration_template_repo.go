package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IntegrationTemplateRepository 模版持久化。
type IntegrationTemplateRepository struct {
	db *gorm.DB
}

func NewIntegrationTemplateRepository(db *gorm.DB) *IntegrationTemplateRepository {
	return &IntegrationTemplateRepository{db: db}
}

func (r *IntegrationTemplateRepository) Create(ctx context.Context, m *model.IntegrationTemplate) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *IntegrationTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.IntegrationTemplate, error) {
	var m model.IntegrationTemplate
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *IntegrationTemplateRepository) GetByName(ctx context.Context, name string) (*model.IntegrationTemplate, error) {
	var m model.IntegrationTemplate
	err := r.db.WithContext(ctx).First(&m, "name = ?", name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *IntegrationTemplateRepository) Update(ctx context.Context, m *model.IntegrationTemplate) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *IntegrationTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.IntegrationTemplate{}, "id = ?", id).Error
}

// IntegrationTemplateListFilter 列表筛选条件。
type IntegrationTemplateListFilter struct {
	Category  string
	Component string
	Keyword   string
	Offset    int
	Limit     int
}

func (r *IntegrationTemplateRepository) List(ctx context.Context, f IntegrationTemplateListFilter) ([]model.IntegrationTemplate, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.IntegrationTemplate{})
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Component != "" {
		q = q.Where("component = ?", f.Component)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("name ILIKE ? OR display_name ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.IntegrationTemplate
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

// CreateVersion 创建新版本。
func (r *IntegrationTemplateRepository) CreateVersion(ctx context.Context, v *model.IntegrationTemplateVersion) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// ListVersions 列出模版下所有版本（按创建时间倒序）。
func (r *IntegrationTemplateRepository) ListVersions(ctx context.Context, templateID uuid.UUID) ([]model.IntegrationTemplateVersion, error) {
	var list []model.IntegrationTemplateVersion
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Order("created_at DESC").Find(&list).Error
	return list, err
}

// GetVersion 查询特定版本。
func (r *IntegrationTemplateRepository) GetVersion(ctx context.Context, templateID uuid.UUID, version string) (*model.IntegrationTemplateVersion, error) {
	var v model.IntegrationTemplateVersion
	err := r.db.WithContext(ctx).Where("template_id = ? AND version = ?", templateID, version).First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// DeleteVersion 按 (template_id, version) 删除一个版本。
func (r *IntegrationTemplateRepository) DeleteVersion(ctx context.Context, templateID uuid.UUID, version string) error {
	return r.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", templateID, version).
		Delete(&model.IntegrationTemplateVersion{}).Error
}
