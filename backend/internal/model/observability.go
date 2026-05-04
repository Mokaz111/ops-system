package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PlatformRoleAdmin = "platform_admin"
	TenantRoleAdmin   = "tenant_admin"
	TenantRoleEditor  = "editor"
	TenantRoleViewer  = "viewer"
	TenantRoleAlert   = "alert_admin"
)

// TenantMember models tenant-scoped RBAC without replacing the legacy
// users.tenant_id field immediately. The legacy field remains a compatibility
// shortcut during migration.
type TenantMember struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index;uniqueIndex:uk_tenant_member_active,where:deleted_at IS NULL"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index;uniqueIndex:uk_tenant_member_active,where:deleted_at IS NULL"`
	Role      string         `json:"role" gorm:"type:varchar(50);not null"`
	Status    string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (TenantMember) TableName() string { return "ops_tenant_members" }

func (m *TenantMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ServiceAccount is used by collectors and automation that should not borrow
// a human user's password or session.
type ServiceAccount struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Role        string         `json:"role" gorm:"type:varchar(50);not null"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceAccount) TableName() string { return "ops_service_accounts" }

func (s *ServiceAccount) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// APIToken stores hashed tokens for users and service accounts.
type APIToken struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID         *uuid.UUID     `json:"tenant_id" gorm:"type:uuid;index"`
	UserID           *uuid.UUID     `json:"user_id" gorm:"type:uuid;index"`
	ServiceAccountID *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	Name             string         `json:"name" gorm:"type:varchar(255);not null"`
	TokenHash        string         `json:"-" gorm:"type:varchar(255);not null;index"`
	Scopes           string         `json:"scopes" gorm:"type:jsonb"`
	ExpiresAt        *time.Time     `json:"expires_at"`
	LastUsedAt       *time.Time     `json:"last_used_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (APIToken) TableName() string { return "ops_api_tokens" }

func (t *APIToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// VMCluster describes a VictoriaMetrics data plane target.
type VMCluster struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string         `json:"name" gorm:"type:varchar(255);not null;uniqueIndex:uk_vm_cluster_name_active,where:deleted_at IS NULL"`
	Mode      string         `json:"mode" gorm:"type:varchar(50);not null"` // shared, dedicated_single, dedicated_cluster
	Namespace string         `json:"namespace" gorm:"type:varchar(100)"`
	SelectURL string         `json:"select_url"`
	InsertURL string         `json:"insert_url"`
	VMAuthURL string         `json:"vmauth_url"`
	Status    string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (VMCluster) TableName() string { return "ops_vm_clusters" }

func (c *VMCluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// VMRoute is the tenant-specific write/query/rule routing contract.
type VMRoute struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	VMClusterID *uuid.UUID     `json:"vm_cluster_id" gorm:"type:uuid;index"`
	RouteType   string         `json:"route_type" gorm:"type:varchar(20);not null;index"` // insert, select, rule
	Path        string         `json:"path" gorm:"type:text;not null"`
	AuthType    string         `json:"auth_type" gorm:"type:varchar(20);default:basic"`
	SecretRef   string         `json:"secret_ref" gorm:"type:varchar(255)"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (VMRoute) TableName() string { return "ops_vm_routes" }

func (r *VMRoute) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// Datasource records tenant-facing query endpoints provisioned into Grafana or
// other UI/query systems.
type Datasource struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Provider   string         `json:"provider" gorm:"type:varchar(50);not null"` // grafana, api, n9e
	Name       string         `json:"name" gorm:"type:varchar(255);not null"`
	Type       string         `json:"type" gorm:"type:varchar(50);not null"` // prometheus, victoriametrics
	URL        string         `json:"url" gorm:"type:text;not null"`
	AuthType   string         `json:"auth_type" gorm:"type:varchar(20)"`
	ExternalID string         `json:"external_id" gorm:"type:varchar(100)"`
	Status     string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Datasource) TableName() string { return "ops_datasources" }

func (d *Datasource) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

type CollectorTarget struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClusterID  *uuid.UUID     `json:"cluster_id" gorm:"type:uuid;index"`
	Name       string         `json:"name" gorm:"type:varchar(255);not null"`
	TargetType string         `json:"target_type" gorm:"type:varchar(50);not null"`
	Spec       string         `json:"spec" gorm:"type:jsonb"`
	Status     string         `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CollectorTarget) TableName() string { return "ops_collector_targets" }

func (c *CollectorTarget) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type RecordingRule struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name      string         `json:"name" gorm:"type:varchar(255);not null"`
	Expr      string         `json:"expr" gorm:"type:text;not null"`
	Labels    string         `json:"labels" gorm:"type:jsonb"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (RecordingRule) TableName() string { return "ops_recording_rules" }

func (r *RecordingRule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type AlertReceiver struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name      string         `json:"name" gorm:"type:varchar(255);not null"`
	Type      string         `json:"type" gorm:"type:varchar(50);not null"`
	Config    string         `json:"config" gorm:"type:jsonb"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertReceiver) TableName() string { return "ops_alert_receivers" }

func (r *AlertReceiver) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ProvisioningTask is the durable lifecycle/outbox record used to make tenant
// provisioning observable and retryable.
type ProvisioningTask struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ResourceID  *uuid.UUID     `json:"resource_id" gorm:"type:uuid;index"`
	Resource    string         `json:"resource" gorm:"type:varchar(50);not null"`
	Action      string         `json:"action" gorm:"type:varchar(50);not null"`
	Step        string         `json:"step" gorm:"type:varchar(100);not null"`
	Status      string         `json:"status" gorm:"type:varchar(20);not null;index"`
	Attempts    int            `json:"attempts" gorm:"default:0"`
	LastError   string         `json:"last_error" gorm:"type:text"`
	Payload     string         `json:"payload" gorm:"type:jsonb"`
	NextRunAt   *time.Time     `json:"next_run_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ProvisioningTask) TableName() string { return "ops_provisioning_tasks" }

func (t *ProvisioningTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type AuditLog struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID   *uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	ActorID    *uuid.UUID `json:"actor_id" gorm:"type:uuid;index"`
	ActorType  string     `json:"actor_type" gorm:"type:varchar(50)"`
	Action     string     `json:"action" gorm:"type:varchar(100);not null;index"`
	Resource   string     `json:"resource" gorm:"type:varchar(100);not null"`
	ResourceID string     `json:"resource_id" gorm:"type:varchar(100)"`
	Details    string     `json:"details" gorm:"type:jsonb"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (AuditLog) TableName() string { return "ops_audit_logs" }

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
