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

// FindByInstanceTemplate 按 (instance_id, template_id) 查找活跃（未软删除）的安装记录。
//
// 活跃行在 Postgres 里由 partial unique index `uk_install_instance_tpl_active` 保证至多 1 条；
// 即使 status=uninstalled 也会占用该索引位（我们没有在卸载时 soft-delete 原行，便于保留审计链），
// 所以 Install/Upgrade/Reinstall 之前都要先经过这里查一下再决定是 create 还是 update。
func (r *IntegrationInstallationRepository) FindByInstanceTemplate(
	ctx context.Context,
	instanceID, templateID uuid.UUID,
) (*model.IntegrationInstallation, error) {
	var m model.IntegrationInstallation
	err := r.db.WithContext(ctx).
		Where("instance_id = ? AND template_id = ?", instanceID, templateID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// CreateWithRevision 在同一事务里创建安装记录 + 首次 revision，并把 last_revision_id 指回。
//
// 这是 Install 的首次安装路径；任何一步失败都会整体回滚，避免只插入 installation 却
// 丢失 revision（或反过来）导致审计链断裂。
func (r *IntegrationInstallationRepository) CreateWithRevision(
	ctx context.Context,
	m *model.IntegrationInstallation,
	rev *model.IntegrationInstallationRevision,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		rev.InstallationID = m.ID
		if err := tx.Create(rev).Error; err != nil {
			return err
		}
		m.LastRevisionID = &rev.ID
		return tx.Model(&model.IntegrationInstallation{}).
			Where("id = ?", m.ID).
			Update("last_revision_id", rev.ID).Error
	})
}

// UpdateWithRevision 在同一事务里更新安装记录 + 追加一条 revision，并把 last_revision_id 指到新 revision。
//
// 用于升级（Install on existing）和卸载：任何失败都会回滚，确保状态与审计保持一致。
func (r *IntegrationInstallationRepository) UpdateWithRevision(
	ctx context.Context,
	m *model.IntegrationInstallation,
	rev *model.IntegrationInstallationRevision,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rev.InstallationID = m.ID
		if err := tx.Create(rev).Error; err != nil {
			return err
		}
		m.LastRevisionID = &rev.ID
		return tx.Save(m).Error
	})
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

// CountActiveByTemplateID 统计某模板（无版本区分）仍在活跃使用中的安装记录数。
// 用于模板本体删除前的引用检查。
func (r *IntegrationInstallationRepository) CountActiveByTemplateID(
	ctx context.Context,
	templateID uuid.UUID,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.IntegrationInstallation{}).
		Where("template_id = ?", templateID).
		Where("status NOT IN ?", []string{"uninstalled", "uninstall_failed"}).
		Count(&total).Error
	return total, err
}

// CountActiveByInstanceID 统计某实例上仍活跃（未卸载）的安装记录数。
// 用于实例删除前的引用检查，避免级联遗留 k8s / grafana 资源。
func (r *IntegrationInstallationRepository) CountActiveByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.IntegrationInstallation{}).
		Where("instance_id = ?", instanceID).
		Where("status NOT IN ?", []string{"uninstalled", "uninstall_failed"}).
		Count(&total).Error
	return total, err
}
