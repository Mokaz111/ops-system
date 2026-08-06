package repository

import (
	"context"
	"errors"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LogClusterRepository struct {
	db *gorm.DB
}

func NewLogClusterRepository(db *gorm.DB) *LogClusterRepository {
	return &LogClusterRepository{db: db}
}

func (r *LogClusterRepository) UpsertShared(ctx context.Context, cluster *model.LogCluster) error {
	var existing model.LogCluster
	q := r.db.WithContext(ctx).Where("status = ?", "active")
	if cluster.ZoneID != nil {
		q = q.Where("zone_id = ?", *cluster.ZoneID)
	} else {
		q = q.Where("zone_id IS NULL")
	}
	err := q.First(&existing).Error
	if err == nil {
		existing.Name = cluster.Name
		existing.BackendType = cluster.BackendType
		existing.Namespace = cluster.Namespace
		existing.ReleaseName = cluster.ReleaseName
		existing.InsertURL = cluster.InsertURL
		existing.SelectURL = cluster.SelectURL
		existing.KafkaBrokers = cluster.KafkaBrokers
		existing.KafkaTopic = cluster.KafkaTopic
		existing.ClusterID = cluster.ClusterID
		existing.Status = cluster.Status
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *LogClusterRepository) GetActiveByZone(ctx context.Context, zoneID uuid.UUID) (*model.LogCluster, error) {
	var cluster model.LogCluster
	err := r.db.WithContext(ctx).
		Where("zone_id = ? AND status = ?", zoneID, "active").
		First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

func (r *LogClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LogCluster, error) {
	var cluster model.LogCluster
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}
