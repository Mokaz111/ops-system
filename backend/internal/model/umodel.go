package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity UModel EntitySet 实例（运维对象）。
type Entity struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	EntityType  string         `json:"entity_type" gorm:"type:varchar(50);not null;index"` // service, k8s_cluster, namespace, workload
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	DisplayName string         `json:"display_name" gorm:"type:varchar(255)"`
	Labels      string         `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Entity) TableName() string { return "ops_entities" }

func (e *Entity) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// MetricSet UModel MetricSet 元数据（指标集定义）。
type MetricSet struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	DisplayName string         `json:"display_name" gorm:"type:varchar(255)"`
	Component   string         `json:"component" gorm:"type:varchar(100);index"`
	Description string         `json:"description" gorm:"type:text"`
	Labels      string         `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (MetricSet) TableName() string { return "ops_metric_sets" }

func (m *MetricSet) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// LogSet UModel LogSet 元数据（日志集定义）。
type LogSet struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	DisplayName string         `json:"display_name" gorm:"type:varchar(255)"`
	Component   string         `json:"component" gorm:"type:varchar(100);index"`
	Description string         `json:"description" gorm:"type:text"`
	Labels      string         `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (LogSet) TableName() string { return "ops_log_sets" }

func (l *LogSet) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// DataLink Entity 与 TelemetryDataSet 的关联（UModel DataLink）。
type DataLink struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID     uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	EntityID     uuid.UUID      `json:"entity_id" gorm:"type:uuid;not null;index"`
	TargetType   string         `json:"target_type" gorm:"type:varchar(50);not null"` // metric_set, log_set, trace_set
	TargetID     uuid.UUID      `json:"target_id" gorm:"type:uuid;not null;index"`
	RelationType string         `json:"relation_type" gorm:"type:varchar(50);default:observes"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (DataLink) TableName() string { return "ops_data_links" }

func (d *DataLink) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
