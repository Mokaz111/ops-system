package repository

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantMemberRepository struct {
	db *gorm.DB
}

func NewTenantMemberRepository(db *gorm.DB) *TenantMemberRepository {
	return &TenantMemberRepository{db: db}
}

func (r *TenantMemberRepository) Upsert(ctx context.Context, m *model.TenantMember) error {
	var existing model.TenantMember
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", m.TenantID, m.UserID).First(&existing).Error
	if err == nil {
		existing.Role = m.Role
		existing.Status = m.Status
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *TenantMemberRepository) GetActive(ctx context.Context, tenantID, userID uuid.UUID) (*model.TenantMember, error) {
	var m model.TenantMember
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, "active").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *TenantMemberRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.TenantMember, error) {
	var rows []model.TenantMember
	err := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "active").Order("created_at DESC").Find(&rows).Error
	return rows, err
}

type VMRouteRepository struct {
	db *gorm.DB
}

func NewVMRouteRepository(db *gorm.DB) *VMRouteRepository { return &VMRouteRepository{db: db} }

func (r *VMRouteRepository) Upsert(ctx context.Context, route *model.VMRoute) error {
	var existing model.VMRoute
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND route_type = ?", route.TenantID, route.RouteType).First(&existing).Error
	if err == nil {
		existing.VMClusterID = route.VMClusterID
		existing.Path = route.Path
		existing.AuthType = route.AuthType
		existing.SecretRef = route.SecretRef
		existing.Status = route.Status
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *VMRouteRepository) GetByTenantAndType(ctx context.Context, tenantID uuid.UUID, routeType string) (*model.VMRoute, error) {
	var route model.VMRoute
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND route_type = ? AND status = ?", tenantID, routeType, "active").First(&route).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &route, nil
}

type DatasourceRepository struct {
	db *gorm.DB
}

func NewDatasourceRepository(db *gorm.DB) *DatasourceRepository {
	return &DatasourceRepository{db: db}
}

func (r *DatasourceRepository) Upsert(ctx context.Context, ds *model.Datasource) error {
	var existing model.Datasource
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND provider = ? AND name = ?", ds.TenantID, ds.Provider, ds.Name).First(&existing).Error
	if err == nil {
		existing.Type = ds.Type
		existing.URL = ds.URL
		existing.AuthType = ds.AuthType
		existing.ExternalID = ds.ExternalID
		existing.Status = ds.Status
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(ds).Error
}

type ProvisioningTaskRepository struct {
	db *gorm.DB
}

func NewProvisioningTaskRepository(db *gorm.DB) *ProvisioningTaskRepository {
	return &ProvisioningTaskRepository{db: db}
}

func (r *ProvisioningTaskRepository) Create(ctx context.Context, t *model.ProvisioningTask) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *ProvisioningTaskRepository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.ProvisioningTask{}).Where("id = ?", id).Updates(map[string]any{
		"status":   "running",
		"attempts": gorm.Expr("attempts + 1"),
	}).Error
}

func (r *ProvisioningTaskRepository) MarkSucceeded(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.ProvisioningTask{}).Where("id = ?", id).Updates(map[string]any{
		"status":       "succeeded",
		"completed_at": &now,
		"last_error":   "",
	}).Error
}

func (r *ProvisioningTaskRepository) MarkFailed(ctx context.Context, id uuid.UUID, err error) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return r.db.WithContext(ctx).Model(&model.ProvisioningTask{}).Where("id = ?", id).Updates(map[string]any{
		"status":     "failed",
		"last_error": msg,
	}).Error
}

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

func (r *AuditLogRepository) Create(ctx context.Context, a *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(a).Error
}
