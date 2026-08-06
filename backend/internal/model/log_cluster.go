package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LogCluster Zone 级共享日志管道（VictoriaLogs + Kafka + Aggregator）。
type LogCluster struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string         `json:"name" gorm:"type:varchar(255);not null"`
	BackendType  string         `json:"backend_type" gorm:"type:varchar(50);not null;default:victorialogs"`
	ZoneID       *uuid.UUID     `json:"zone_id" gorm:"type:uuid;index"`
	ClusterID    *uuid.UUID     `json:"cluster_id" gorm:"type:uuid;index"`
	ReleaseName  string         `json:"release_name" gorm:"type:varchar(100)"`
	Namespace    string         `json:"namespace" gorm:"type:varchar(100)"`
	InsertURL    string         `json:"insert_url" gorm:"type:text"`
	SelectURL    string         `json:"select_url" gorm:"type:text"`
	KafkaBrokers string         `json:"kafka_brokers" gorm:"type:text"`
	KafkaTopic   string         `json:"kafka_topic" gorm:"type:varchar(255)"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (LogCluster) TableName() string { return "ops_log_clusters" }

func (l *LogCluster) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
