package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Cluster K8s 集群注册表。未注册时所有 Instance 使用平台默认 kubeconfig（config.Kubernetes）。
//
// name 在活跃行内唯一（partial unique index），避免软删除后同名集群无法重建。
type Cluster struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name           string         `json:"name" gorm:"type:varchar(255);not null;uniqueIndex:uk_cluster_name_active,where:deleted_at IS NULL"`
	DisplayName    string         `json:"display_name" gorm:"type:varchar(255)"`
	Description    string         `json:"description" gorm:"type:text"`
	InCluster      bool           `json:"in_cluster"` // true = 使用 Pod 内 ServiceAccount
	Kubeconfig     string         `json:"-" gorm:"type:text"` // 原始 kubeconfig 文本，敏感字段
	KubeconfigPath string         `json:"kubeconfig_path" gorm:"type:varchar(500)"` // 本地路径（与 Kubeconfig 二选一）
	Status         string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名。
func (Cluster) TableName() string {
	return "ops_clusters"
}

// BeforeCreate 生成主键。
func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
