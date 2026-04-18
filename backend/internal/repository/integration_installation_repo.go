package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IntegrationInstallationRepository 安装记录持久化。
type IntegrationInstallationRepository struct {
	db *gorm.DB
}

func NewIntegrationInstallationRepository(db *gorm.DB) *IntegrationInstallationRepository {
	return &IntegrationInstallationRepository{db: db}
}

func (r *IntegrationInstallationRepository) Create(ctx context.Context, m *model.IntegrationInstallation) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *IntegrationInstallationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.IntegrationInstallation, error) {
	var m model.IntegrationInstallation
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *IntegrationInstallationRepository) Update(ctx context.Context, m *model.IntegrationInstallation) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *IntegrationInstallationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.IntegrationInstallation{}, "id = ?", id).Error
}

// IntegrationInstallationListFilter 列表筛选条件。
type IntegrationInstallationListFilter struct {
	TenantID   *uuid.UUID
	InstanceID *uuid.UUID
	TemplateID *uuid.UUID
	Status     string
	Offset     int
	Limit      int
}

func (r *IntegrationInstallationRepository) List(ctx context.Context, f IntegrationInstallationListFilter) ([]model.IntegrationInstallation, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.IntegrationInstallation{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.InstanceID != nil {
		q = q.Where("instance_id = ?", *f.InstanceID)
	}
	if f.TemplateID != nil {
		q = q.Where("template_id = ?", *f.TemplateID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.IntegrationInstallation
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

// CreateRevision 写入一次变更快照。
func (r *IntegrationInstallationRepository) CreateRevision(ctx context.Context, rev *model.IntegrationInstallationRevision) error {
	return r.db.WithContext(ctx).Create(rev).Error
}

// ListRevisions 按安装记录列出变更历史。
func (r *IntegrationInstallationRepository) ListRevisions(ctx context.Context, installationID uuid.UUID) ([]model.IntegrationInstallationRevision, error) {
	var list []model.IntegrationInstallationRevision
	err := r.db.WithContext(ctx).Where("installation_id = ?", installationID).Order("created_at DESC").Find(&list).Error
	return list, err
}

// CountActiveByTemplateVersion 统计仍在使用（未卸载）指定模板版本的安装记录数。
// 用于模板版本删除前的引用检查。
func (r *IntegrationInstallationRepository) CountActiveByTemplateVersion(
	ctx context.Context,
	templateID uuid.UUID,
	version string,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.IntegrationInstallation{}).
		Where("template_id = ? AND template_version = ?", templateID, version).
		Where("status NOT IN ?", []string{"uninstalled", "uninstall_failed"}).
		Count(&total).Error
	return total, err
}
