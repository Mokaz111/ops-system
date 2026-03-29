package repository

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AlertEventRepository 告警事件持久化。
type AlertEventRepository struct {
	db *gorm.DB
}

func NewAlertEventRepository(db *gorm.DB) *AlertEventRepository {
	return &AlertEventRepository{db: db}
}

// AlertEventListFilter 列表筛选。
type AlertEventListFilter struct {
	TenantID  *uuid.UUID
	RuleID    *uuid.UUID
	Level     string
	Status    string
	StartTime *time.Time
	EndTime   *time.Time
	Offset    int
	Limit     int
}

// Create 创建告警事件。
func (r *AlertEventRepository) Create(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// GetByID 按 ID。
func (r *AlertEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.WithContext(ctx).First(&event, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// Update 更新。
func (r *AlertEventRepository) Update(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}

// List 分页列表。
func (r *AlertEventRepository) List(ctx context.Context, f AlertEventListFilter) ([]model.AlertEvent, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AlertEvent{})
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	if f.RuleID != nil {
		q = q.Where("rule_id = ?", *f.RuleID)
	}
	if f.Level != "" {
		q = q.Where("level = ?", f.Level)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.StartTime != nil {
		q = q.Where("created_at >= ?", *f.StartTime)
	}
	if f.EndTime != nil {
		q = q.Where("created_at <= ?", *f.EndTime)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AlertEvent
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListFiring 获取所有 firing 状态事件。
func (r *AlertEventRepository) ListFiring(ctx context.Context, tenantID uuid.UUID) ([]model.AlertEvent, error) {
	var list []model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, "firing").
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

// Upsert 按 rule_id + tenant_id 插入或更新。
func (r *AlertEventRepository) Upsert(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "rule_id"}, {Name: "tenant_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"rule_name", "level", "status", "start_time", "end_time", "details", "notified"}),
		}).
		Create(event).Error
}

// CountByStatus 按状态计数。
func (r *AlertEventRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&n).Error
	return n, err
}

// LevelStats 按级别统计结果。
type LevelStats struct {
	Level string `json:"level"`
	Count int64  `json:"count"`
}

// StatsByLevel 按级别统计。
func (r *AlertEventRepository) StatsByLevel(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]LevelStats, error) {
	var rows []LevelStats
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("level, count(*) as count").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, start, end).
		Group("level").
		Find(&rows).Error
	return rows, err
}

// RuleStats 按规则统计结果。
type RuleStats struct {
	RuleID   uuid.UUID `json:"rule_id"`
	RuleName string    `json:"rule_name"`
	Count    int64     `json:"count"`
}

// StatsByRule Top N 规则统计。
func (r *AlertEventRepository) StatsByRule(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit int) ([]RuleStats, error) {
	var rows []RuleStats
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("rule_id, rule_name, count(*) as count").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, start, end).
		Group("rule_id, rule_name").
		Order("count DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// TrendPoint 趋势数据点。
type TrendPoint struct {
	Bucket time.Time `json:"bucket"`
	Count  int64     `json:"count"`
}

// Trend 时间分桶趋势。
func (r *AlertEventRepository) Trend(ctx context.Context, tenantID uuid.UUID, start, end time.Time, interval string) ([]TrendPoint, error) {
	var rows []TrendPoint
	err := r.db.WithContext(ctx).Raw(
		`SELECT date_trunc(?, created_at) AS bucket, count(*) AS count
		 FROM alert_events
		 WHERE tenant_id = ? AND created_at BETWEEN ? AND ?
		 GROUP BY bucket ORDER BY bucket`,
		interval, tenantID, start, end,
	).Scan(&rows).Error
	return rows, err
}
