package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/logstore"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 日志实例相关业务错误。
var (
	ErrLogInstanceNotFound = errors.New("log instance not found")
	ErrLogInstanceName     = errors.New("instance_name required")
	ErrLogZoneRequired     = errors.New("zone_id required")
	ErrZoneLogsNotReady    = errors.New("zone log pipeline is not ready")
)

// LogInstanceService 日志工作空间业务。
type LogInstanceService struct {
	repo          *repository.LogInstanceRepository
	logClusterRepo *repository.LogClusterRepository
	zoneRepo      *repository.ZoneRepository
	logSetRepo    *repository.LogSetRepository
	logsCfg       *config.LogsConfig
}

func NewLogInstanceService(
	repo *repository.LogInstanceRepository,
	logClusterRepo *repository.LogClusterRepository,
	zoneRepo *repository.ZoneRepository,
	logSetRepo *repository.LogSetRepository,
	logsCfg *config.LogsConfig,
) *LogInstanceService {
	return &LogInstanceService{
		repo: repo, logClusterRepo: logClusterRepo, zoneRepo: zoneRepo,
		logSetRepo: logSetRepo, logsCfg: logsCfg,
	}
}

// CreateLogInstanceRequest 创建日志工作空间请求。
type CreateLogInstanceRequest struct {
	TenantID      uuid.UUID `json:"tenant_id" binding:"required"`
	ZoneID        string    `json:"zone_id" binding:"required"`
	InstanceName  string    `json:"instance_name" binding:"required"`
	Namespace     string    `json:"namespace"`
	ReleaseName   string    `json:"release_name"`
	BackendType   string    `json:"backend_type"`
	RetentionDays int       `json:"retention_days"`
	Spec          string    `json:"spec"`
}

// LogQueryRequest 日志查询请求。
type LogQueryRequest struct {
	Query string     `json:"query"`
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
	Limit int        `json:"limit"`
}

// Create 创建日志工作空间并登记默认 LogSet。
func (s *LogInstanceService) Create(ctx context.Context, req *CreateLogInstanceRequest) (*model.LogInstance, error) {
	if req.InstanceName == "" {
		return nil, ErrLogInstanceName
	}
	zoneID, err := uuid.Parse(strings.TrimSpace(req.ZoneID))
	if err != nil {
		return nil, ErrLogZoneRequired
	}
	if s.zoneRepo != nil {
		z, zerr := s.zoneRepo.GetByID(ctx, zoneID)
		if zerr != nil || z == nil {
			return nil, ErrZoneNotFound
		}
	}
	pool, err := s.logClusterRepo.GetActiveByZone(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, ErrZoneLogsNotReady
	}

	backendType := strings.TrimSpace(req.BackendType)
	if backendType == "" {
		backendType = logstore.BackendVictoriaLogs
	}
	m := &model.LogInstance{
		TenantID:      req.TenantID,
		ZoneID:        &zoneID,
		BackendType:   backendType,
		InstanceName:  req.InstanceName,
		Namespace:     req.Namespace,
		ReleaseName:   req.ReleaseName,
		Endpoint:      pool.SelectURL,
		RetentionDays: req.RetentionDays,
		Spec:          req.Spec,
		Status:        "active",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	if s.logSetRepo != nil {
		ls := &model.LogSet{
			TenantID:    req.TenantID,
			Name:        req.InstanceName,
			DisplayName: req.InstanceName,
			Component:   "logs",
			Description: "default log set for " + req.InstanceName,
			Status:      "active",
			Labels:      `{}`,
		}
		if err := s.logSetRepo.Create(ctx, ls); err != nil {
			return m, fmt.Errorf("log instance created but log set failed: %w", err)
		}
	}
	return m, nil
}

// Query 查询日志（强制租户过滤）。
func (s *LogInstanceService) Query(ctx context.Context, id uuid.UUID, req *LogQueryRequest) (*logstore.QueryResult, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrLogInstanceNotFound
	}
	if m.ZoneID == nil {
		return nil, ErrZoneLogsNotReady
	}
	pool, err := s.logClusterRepo.GetActiveByZone(ctx, *m.ZoneID)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, ErrZoneLogsNotReady
	}

	timeout := 15 * time.Second
	if s.logsCfg != nil && s.logsCfg.HTTPTimeoutSeconds > 0 {
		timeout = time.Duration(s.logsCfg.HTTPTimeoutSeconds) * time.Second
	}
	store, err := logstore.New(m.BackendType, logstore.Config{
		SelectURL: pool.SelectURL,
		InsertURL: pool.InsertURL,
		Timeout:   timeout,
	})
	if err != nil {
		return nil, err
	}
	if req == nil {
		req = &LogQueryRequest{}
	}
	// 未指定时间范围时默认查近 15 分钟，避免全时间段扫描拖垮存储。
	if req.Start == nil && req.End == nil {
		end := time.Now().UTC()
		start := end.Add(-15 * time.Minute)
		req.Start, req.End = &start, &end
	}
	zoneSlug := ""
	if s.zoneRepo != nil {
		z, _ := s.zoneRepo.GetByID(ctx, *m.ZoneID)
		if z != nil {
			zoneSlug = z.Slug
		}
	}
	return store.Query(ctx, logstore.QueryRequest{
		TenantFilter: logstore.TenantFilter{
			TenantID: m.TenantID.String(),
			Zone:     zoneSlug,
		},
		RawQuery: req.Query,
		Start:    req.Start,
		End:      req.End,
		Limit:    req.Limit,
	})
}

// Get 查询。
func (s *LogInstanceService) Get(ctx context.Context, id uuid.UUID) (*model.LogInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrLogInstanceNotFound
	}
	return m, nil
}

// UpdateLogInstanceRequest 更新请求。
type UpdateLogInstanceRequest struct {
	InstanceName  string `json:"instance_name"`
	RetentionDays int    `json:"retention_days"`
	Spec          string `json:"spec"`
	Status        string `json:"status"`
}

// Update 更新。
func (s *LogInstanceService) Update(ctx context.Context, id uuid.UUID, req *UpdateLogInstanceRequest) (*model.LogInstance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrLogInstanceNotFound
	}
	if req.InstanceName != "" {
		m.InstanceName = req.InstanceName
	}
	if req.RetentionDays > 0 {
		m.RetentionDays = req.RetentionDays
	}
	if req.Spec != "" {
		m.Spec = req.Spec
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete 删除。
func (s *LogInstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrLogInstanceNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *LogInstanceService) List(ctx context.Context, tenantID *uuid.UUID, keyword string, page, pageSize int) ([]model.LogInstance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.LogInstanceListFilter{
		TenantID: tenantID,
		Keyword:  keyword,
		Offset:   offset,
		Limit:    pageSize,
	})
}
