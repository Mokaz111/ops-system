package repository

import (
	"context"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ZoneRepository 可用区仓储。
type ZoneRepository struct {
	db *gorm.DB
}

func NewZoneRepository(db *gorm.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

func (r *ZoneRepository) Create(ctx context.Context, m *model.Zone) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *ZoneRepository) Update(ctx context.Context, m *model.Zone) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *ZoneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Zone{}, "id = ?", id).Error
}

func (r *ZoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Zone, error) {
	var m model.Zone
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *ZoneRepository) GetBySlug(ctx context.Context, slug string) (*model.Zone, error) {
	var m model.Zone
	err := r.db.WithContext(ctx).First(&m, "slug = ?", slug).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// ZoneListFilter 列表筛选。
type ZoneListFilter struct {
	Status string
	Offset int
	Limit  int
}

func (r *ZoneRepository) List(ctx context.Context, f ZoneListFilter) ([]model.Zone, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Zone{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	var rows []model.Zone
	if err := q.Order("created_at desc").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CountActiveInstances 统计某 Zone 下的活跃实例数。
func (r *ZoneRepository) CountActiveInstances(ctx context.Context, zoneID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Instance{}).
		Where("zone_id = ? AND status IN ('creating','deploying','running','degraded')", zoneID).
		Count(&count).Error
	return count, err
}
