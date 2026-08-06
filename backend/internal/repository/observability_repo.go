package repository

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

func (r *VMRouteRepository) DeactivateByTenant(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.VMRoute{}).
		Where("tenant_id = ?", tenantID).
		Update("status", "inactive").Error
}

type VMClusterRepository struct {
	db *gorm.DB
}

func NewVMClusterRepository(db *gorm.DB) *VMClusterRepository {
	return &VMClusterRepository{db: db}
}

func (r *VMClusterRepository) UpsertShared(ctx context.Context, cluster *model.VMCluster) error {
	var existing model.VMCluster
	q := r.db.WithContext(ctx).Where("mode = ? AND status = ?", "shared", "active")
	if cluster.ZoneID != nil {
		q = q.Where("zone_id = ?", *cluster.ZoneID)
	} else {
		q = q.Where("zone_id IS NULL")
	}
	err := q.First(&existing).Error
	if err == nil {
		existing.Name = cluster.Name
		existing.Namespace = cluster.Namespace
		existing.ReleaseName = cluster.ReleaseName
		existing.SelectURL = cluster.SelectURL
		existing.InsertURL = cluster.InsertURL
		existing.VMAuthURL = cluster.VMAuthURL
		existing.TargetURL = cluster.TargetURL
		existing.ClusterID = cluster.ClusterID
		existing.Status = cluster.Status
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *VMClusterRepository) GetActiveSharedByZone(ctx context.Context, zoneID uuid.UUID) (*model.VMCluster, error) {
	var cluster model.VMCluster
	err := r.db.WithContext(ctx).
		Where("zone_id = ? AND mode = ? AND status = ?", zoneID, "shared", "active").
		First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

func (r *VMClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMCluster, error) {
	var cluster model.VMCluster
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

// ListActive 列出全部活跃 VM 集群。
func (r *VMClusterRepository) ListActive(ctx context.Context) ([]model.VMCluster, error) {
	var list []model.VMCluster
	err := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("created_at ASC").
		Find(&list).Error
	return list, err
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

func (r *DatasourceRepository) DeactivateByTenant(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Datasource{}).
		Where("tenant_id = ?", tenantID).
		Update("status", "inactive").Error
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

// AuditLogListFilter 审计日志列表筛选。
type AuditLogListFilter struct {
	Action   string
	Resource string
	ActorID  *uuid.UUID
	TenantID *uuid.UUID
	Offset   int
	Limit    int
}

// List 分页查询审计日志。
func (r *AuditLogRepository) List(ctx context.Context, f AuditLogListFilter) ([]model.AuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Resource != "" {
		q = q.Where("resource = ?", f.Resource)
	}
	if f.ActorID != nil {
		q = q.Where("actor_id = ?", *f.ActorID)
	}
	if f.TenantID != nil {
		q = q.Where("tenant_id = ?", *f.TenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AuditLog
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
