package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"
	"ops-system/backend/pkg/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantNameRequired  = errors.New("tenant_name required")
	ErrDeptNotFound        = errors.New("department not found")
	ErrDeptHasTenant       = errors.New("department already has a tenant")
	ErrInvalidTemplateType = errors.New("invalid template_type")
	ErrQuotaConfigNotJSON  = errors.New("quota_config must be valid JSON object")
	ErrTenantHasInstances  = errors.New("tenant has instances")
)

var allowedTemplateTypes = map[string]struct{}{
	"shared":           {},
	"dedicated_single": {},
	"dedicated_cluster": {},
}

type CreateTenantRequest struct {
	TenantName   string
	DeptID       uuid.UUID
	TemplateType string
	QuotaConfig  string
}

type UpdateTenantRequest struct {
	TenantName   string
	TemplateType string
	QuotaConfig  string
	Status       string
}

// TenantService 租户业务（不直接依赖 N9E，告警由 N9E 独立管理）。
type TenantService struct {
	dept    *repository.DepartmentRepository
	tenant  *repository.TenantRepository
	inst    *repository.InstanceRepository
	vmSync  *vm.SyncService
	grafana *grafana.Client
	orch    *OrchestratorService
	log     *zap.Logger
}

func NewTenantService(
	dept *repository.DepartmentRepository,
	tenant *repository.TenantRepository,
	inst *repository.InstanceRepository,
	vmSync *vm.SyncService,
	grafanaClient *grafana.Client,
	orch *OrchestratorService,
	log *zap.Logger,
) *TenantService {
	return &TenantService{
		dept: dept, tenant: tenant, inst: inst,
		vmSync: vmSync, grafana: grafanaClient, orch: orch, log: log,
	}
}

func (s *TenantService) InsertURL(vmuserID string) string {
	if s.vmSync == nil {
		return ""
	}
	return s.vmSync.InsertURL(vmuserID)
}

// Create 创建租户：校验 → 生成 vmuser → 落库 → VM 同步 → Grafana 组织 → K8s 编排。
func (s *TenantService) Create(ctx context.Context, req *CreateTenantRequest) (*model.Tenant, error) {
	if req.TenantName == "" {
		return nil, ErrTenantNameRequired
	}
	if req.TemplateType == "" || !allowedTemplateType(req.TemplateType) {
		return nil, ErrInvalidTemplateType
	}
	if err := validateQuotaJSON(req.QuotaConfig); err != nil {
		return nil, err
	}

	d, err := s.dept.GetByID(ctx, req.DeptID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrDeptNotFound
	}

	existing, err := s.tenant.GetByDeptID(ctx, req.DeptID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDeptHasTenant
	}

	vmuserID, err := s.allocVMUserID(ctx)
	if err != nil {
		return nil, err
	}
	vmKey, err := utils.RandomHex(32)
	if err != nil {
		return nil, err
	}

	quotaCfg := strings.TrimSpace(req.QuotaConfig)
	if quotaCfg == "" {
		quotaCfg = "{}"
	}

	t := &model.Tenant{
		TenantName:   strings.TrimSpace(req.TenantName),
		DeptID:       req.DeptID,
		VMUserID:     vmuserID,
		VMUserKey:    vmKey,
		TemplateType: req.TemplateType,
		QuotaConfig:  quotaCfg,
		Status:       "active",
	}
	if err := s.tenant.Create(ctx, t); err != nil {
		return nil, err
	}

	if s.vmSync != nil {
		s.vmSync.OnTenantCreated(ctx, t)
	}
	if s.grafana != nil && s.grafana.Enabled() {
		if err := s.grafana.SyncTenantOnCreate(ctx, t); err == nil {
			_ = s.tenant.Update(ctx, t)
		}
	}
	if s.orch != nil {
		if err := s.orch.DeployTenant(ctx, t); err != nil && s.log != nil {
			s.log.Warn("orchestrator_deploy_failed", zap.Error(err), zap.String("tenant_id", t.ID.String()))
		}
	}
	return t, nil
}

func allowedTemplateType(s string) bool {
	_, ok := allowedTemplateTypes[s]
	return ok
}

func validateQuotaJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ErrQuotaConfigNotJSON
	}
	return nil
}

func (s *TenantService) allocVMUserID(ctx context.Context) (string, error) {
	for i := 0; i < 8; i++ {
		id := "vmuser-" + uuid.New().String()
		exist, err := s.tenant.GetByVMUserID(ctx, id)
		if err != nil {
			return "", err
		}
		if exist == nil {
			return id, nil
		}
	}
	return "", errors.New("failed to allocate vmuser_id")
}

func (s *TenantService) Get(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	t, err := s.tenant.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	return t, nil
}

func (s *TenantService) List(ctx context.Context, page, pageSize int, deptID *uuid.UUID, templateType, status, keyword string) ([]model.Tenant, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.tenant.List(ctx, repository.TenantListFilter{
		DeptID:       deptID,
		TemplateType: templateType,
		Status:       status,
		Keyword:      keyword,
		Offset:       offset,
		Limit:        pageSize,
	})
}

func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req *UpdateTenantRequest) (*model.Tenant, error) {
	t, err := s.tenant.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	if req.TenantName == "" {
		return nil, ErrTenantNameRequired
	}
	if req.TemplateType != "" && !allowedTemplateType(req.TemplateType) {
		return nil, ErrInvalidTemplateType
	}
	if err := validateQuotaJSON(req.QuotaConfig); err != nil {
		return nil, err
	}
	t.TenantName = strings.TrimSpace(req.TenantName)
	if req.TemplateType != "" {
		t.TemplateType = req.TemplateType
	}
	if req.QuotaConfig != "" {
		t.QuotaConfig = req.QuotaConfig
	}
	if req.Status != "" {
		t.Status = req.Status
	}
	if err := s.tenant.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete 删除租户（无实例挂载时）。
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	t, err := s.tenant.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTenantNotFound
	}
	n, err := s.inst.CountByTenantID(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrTenantHasInstances
	}
	if s.grafana != nil {
		s.grafana.SyncTenantOnDelete(ctx, t)
	}
	if s.vmSync != nil {
		s.vmSync.OnTenantDeleted(ctx, t)
	}
	if s.orch != nil {
		if err := s.orch.DeleteTenant(ctx, t); err != nil && s.log != nil {
			s.log.Warn("orchestrator_delete_failed", zap.Error(err), zap.String("tenant_id", t.ID.String()))
		}
	}
	return s.tenant.Delete(ctx, id)
}

type TenantMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	SeriesCount        int64   `json:"series_count"`
	IngestQPS          float64 `json:"ingest_qps"`
	Note               string  `json:"note"`
}

func (s *TenantService) GetMetrics(ctx context.Context, id uuid.UUID) (*TenantMetrics, error) {
	t, err := s.tenant.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}
	return &TenantMetrics{
		Note: "placeholder; connect VictoriaMetrics / cluster metrics in later phase",
	}, nil
}
