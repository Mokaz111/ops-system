package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertRuleRepository 告警规则持久化。
type AlertRuleRepository struct {
	db *gorm.DB
}

func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

// AlertRuleListFilter 列表筛选。
type AlertRuleListFilter struct {
	TenantID *uuid.UUID
	RuleType string
	Level    string
	Enabled  *bool
	Keyword  string
	Offset   int
	Limit    int
}

// Create 创建告警规则。
func (r *AlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 按 ID。
func (r *AlertRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.WithContext(ctx).First(&rule, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// Update 更新。
func (r *AlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 软删除。
func (r *AlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.AlertRule{}, "id = ?", id).Error
}

// List 分页列表。
func (r *AlertRuleRepository) List(ctx context.Context, f AlertRuleListFilter) ([]model.AlertRule, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AlertRule{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.RuleType != "" {
		q = q.Where("rule_type = ?", f.RuleType)
	}
	if f.Level != "" {
		q = q.Where("level = ?", f.Level)
	}
	if f.Enabled != nil {
		q = q.Where("enabled = ?", *f.Enabled)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("rule_name ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AlertRule
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListByTenantID 按租户。
func (r *AlertRuleRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.AlertRule, error) {
	var list []model.AlertRule
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&list).Error
	return list, err
}

// CountByTenantID 按租户计数。
func (r *AlertRuleRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("tenant_id = ?", tenantID).Count(&n).Error
	return n, err
}
