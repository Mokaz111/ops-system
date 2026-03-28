package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/n9e"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"
	"ops-system/backend/pkg/utils"

	"github.com/google/uuid"
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
	"shared":              {},
	"dedicated_single":    {},
	"dedicated_cluster":   {},
}

// CreateTenantRequest 创建租户（不含 VMuser Key 输出前的字段）。
type CreateTenantRequest struct {
	TenantName   string
	DeptID       uuid.UUID
	TemplateType string
	QuotaConfig  string // JSON 字符串，可为空
}

// UpdateTenantRequest 更新租户（不可改 vmuser_id / vmuser_key / dept_id）。
type UpdateTenantRequest struct {
	TenantName   string
	TemplateType string
	QuotaConfig  string
	Status       string
}

// TenantService 租户业务。
type TenantService struct {
	dept   *repository.DepartmentRepository
	tenant *repository.TenantRepository
	inst   *repository.InstanceRepository
	vmSync *vm.SyncService
	n9e      *n9e.Client
	grafana  *grafana.Client
}

func NewTenantService(
	dept *repository.DepartmentRepository,
	tenant *repository.TenantRepository,
	inst *repository.InstanceRepository,
	vmSync *vm.SyncService,
	n9eClient *n9e.Client,
	grafanaClient *grafana.Client,
) *TenantService {
	return &TenantService{dept: dept, tenant: tenant, inst: inst, vmSync: vmSync, n9e: n9eClient, grafana: grafanaClient}
}

// InsertURL 对外写入路径（vmauth /insert/{vmuser_id}）。
func (s *TenantService) InsertURL(vmuserID string) string {
	if s.vmSync == nil {
		return ""
	}
	return s.vmSync.InsertURL(vmuserID)
}

// Create 创建租户：校验部门、部门唯一租户、生成 vmuser_id/key；K8s/N9E/Grafana 同步在后续阶段接入。
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

	t := &model.Tenant{
		TenantName:   strings.TrimSpace(req.TenantName),
		DeptID:       req.DeptID,
		VMUserID:     vmuserID,
		VMUserKey:    vmKey,
		TemplateType: req.TemplateType,
		QuotaConfig:  req.QuotaConfig,
		Status:       "active",
	}
	if err := s.tenant.Create(ctx, t); err != nil {
		return nil, err
	}
	if s.vmSync != nil {
		s.vmSync.OnTenantCreated(ctx, t)
	}
	if s.n9e != nil && s.n9e.Enabled() {
		if err := s.n9e.SyncTenantOnCreate(ctx, t); err == nil {
			_ = s.tenant.Update(ctx, t)
		}
	}
	if s.grafana != nil && s.grafana.Enabled() {
		if err := s.grafana.SyncTenantOnCreate(ctx, t); err == nil {
			_ = s.tenant.Update(ctx, t)
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

// Get 详情。
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

// List 分页筛选。
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

// Update 更新。
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
	t.QuotaConfig = req.QuotaConfig
	if req.Status != "" {
		t.Status = req.Status
	}
	if err := s.tenant.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete 删除（无实例挂载）。
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
	if s.n9e != nil {
		s.n9e.SyncTenantOnDelete(ctx, t)
	}
	if s.vmSync != nil {
		s.vmSync.OnTenantDeleted(ctx, t)
	}
	return s.tenant.Delete(ctx, id)
}

// TenantMetrics 资源使用占位（后续对接 VictoriaMetrics / K8s metrics）。
type TenantMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	SeriesCount        int64   `json:"series_count"`
	IngestQPS          float64 `json:"ingest_qps"`
	Note               string  `json:"note"`
}

// GetMetrics 占位指标。
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
