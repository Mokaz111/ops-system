package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BusinessCluster 租户接入的业务集群（被监控目标）。
type BusinessCluster struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	InstanceID      uuid.UUID      `json:"instance_id" gorm:"type:uuid;not null;index"`
	Name            string         `json:"name" gorm:"type:varchar(255);not null"`
	DisplayName     string         `json:"display_name" gorm:"type:varchar(255)"`
	Kubeconfig      string         `json:"-" gorm:"type:text"`
	KubeconfigPath  string         `json:"kubeconfig_path" gorm:"type:varchar(500)"`
	AgentStatus     string         `json:"agent_status" gorm:"type:varchar(20);default:pending"`
	LogAgentStatus  string         `json:"log_agent_status" gorm:"type:varchar(20);default:pending"`
	LogInstanceID   *uuid.UUID     `json:"log_instance_id" gorm:"type:uuid;index"`
	Labels          string         `json:"labels" gorm:"type:jsonb;default:'{}'"`
	// MetricsCollectConfig VMAgent 采集配置 JSON（见 MetricsCollectConfig）。
	MetricsCollectConfig string `json:"metrics_collect_config" gorm:"type:jsonb;default:'{}'"`
	// LogsCollectConfig Vector 采集配置 JSON（见 LogsCollectConfig）。
	LogsCollectConfig string `json:"logs_collect_config" gorm:"type:jsonb;default:'{}'"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (BusinessCluster) TableName() string { return "ops_business_clusters" }

func (b *BusinessCluster) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
