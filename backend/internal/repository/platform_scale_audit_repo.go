package repository

import (
	"context"
	"time"

	"ops-system/backend/internal/model"

	"gorm.io/gorm"
)

type PlatformScaleAuditRepository struct {
	db *gorm.DB
}

func NewPlatformScaleAuditRepository(db *gorm.DB) *PlatformScaleAuditRepository {
	return &PlatformScaleAuditRepository{db: db}
}

func (r *PlatformScaleAuditRepository) Create(ctx context.Context, row *model.PlatformScaleAudit) error {
	return r.db.WithContext(ctx).Create(row).Error
}

type PlatformScaleAuditListFilter struct {
	TargetID  string
	Status    string
	Operator  string
	StartTime *time.Time
	EndTime   *time.Time
	Offset    int
	Limit     int
}

func (r *PlatformScaleAuditRepository) List(ctx context.Context, f PlatformScaleAuditListFilter) ([]model.PlatformScaleAudit, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.PlatformScaleAudit{})
	if f.TargetID != "" {
		q = q.Where("target_id = ?", f.TargetID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Operator != "" {
		q = q.Where("username ILIKE ?", "%"+f.Operator+"%")
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
	var rows []model.PlatformScaleAudit
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
