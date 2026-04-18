package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScaleEvent 记录一次实例伸缩操作的审计。
//
// 每次 ScaleService.Scale 调用都会写入一条，无论成功/失败，用于：
//   - 审计（谁在何时做了什么伸缩）
//   - 诊断（CR 直接 patch vs helm upgrade 的效果、失败原因）
//   - 界面回放（InstanceDetail · 伸缩历史）
//
// 审计记录不支持软删除，以保证不可篡改。清理走冷表 / 归档策略。
type ScaleEvent struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID   uuid.UUID `json:"instance_id" gorm:"type:uuid;not null;index:idx_scale_event_instance_time,priority:1"`
	InstanceName string    `json:"instance_name" gorm:"type:varchar(255)"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	ScaleType    string    `json:"scale_type" gorm:"type:varchar(20);index"`        // horizontal / vertical / storage
	Method       string    `json:"method" gorm:"type:varchar(32);index"`            // cr_patch / helm_upgrade / k8s_native / rejected
	Replicas     *int32    `json:"replicas"`
	CPU          string    `json:"cpu" gorm:"type:varchar(32)"`
	Memory       string    `json:"memory" gorm:"type:varchar(32)"`
	Storage      string    `json:"storage" gorm:"type:varchar(32)"`
	Status       string    `json:"status" gorm:"type:varchar(20);index"` // success / failed
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	Operator     string    `json:"operator" gorm:"type:varchar(100)"`
	CreatedAt    time.Time `json:"created_at" gorm:"index:idx_scale_event_instance_time,priority:2,sort:desc"`
}

// TableName 表名。
func (ScaleEvent) TableName() string {
	return "ops_scale_events"
}

// BeforeCreate 生成主键。
func (e *ScaleEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
