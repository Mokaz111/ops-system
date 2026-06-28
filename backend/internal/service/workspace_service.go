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
	ErrWorkspaceNotFound          = errors.New("workspace not found")
	ErrWorkspaceNameRequired      = errors.New("workspace_name required")
	ErrWorkspaceSlugConflict      = errors.New("workspace slug already exists")
	ErrInvalidTemplateType        = errors.New("invalid template_type")
	ErrQuotaConfigNotJSON         = errors.New("quota_config must be valid JSON object")
	ErrWorkspaceHasInstances      = errors.New("workspace has instances")
	ErrWorkspaceProvisionFailed   = errors.New("workspace provisioning failed")
	ErrWorkspaceDeprovisionFailed = errors.New("workspace deprovision failed")
)

var allowedTemplateTypes = map[string]struct{}{
	"shared":            {},
	"dedicated_single":  {},
	"dedicated_cluster": {},
}

type CreateWorkspaceRequest struct {
	WorkspaceName     string
	TemplateType      string
	QuotaConfig       string
	GrafanaInstanceID *uuid.UUID
}

type UpdateWorkspaceRequest struct {
	WorkspaceName     string
	TemplateType      string
	QuotaConfig       string
	Status            string
	GrafanaInstanceID *uuid.UUID
}

// WorkspaceService 工作空间业务（不直接依赖 N9E，告警由 N9E 独立管理）。
type WorkspaceService struct {
	workspace       *repository.WorkspaceRepository
	inst            *repository.InstanceRepository
	vmSync          *vm.SyncService
	vmQuery         *vm.QueryClient
	provisioner     *WorkspaceProvisioner
	grafanaResolver func(ctx context.Context, instanceID *uuid.UUID) (*grafana.Client, error)
	grafanaSvc      *GrafanaService
	orch            *OrchestratorService
	log             *zap.Logger
}

func NewWorkspaceService(
	tenant *repository.WorkspaceRepository,
	inst *repository.InstanceRepository,
	vmSync *vm.SyncService,
	vmQuery *vm.QueryClient,
	provisioner *WorkspaceProvisioner,
	grafanaResolver func(ctx context.Context, instanceID *uuid.UUID) (*grafana.Client, error),
	grafanaSvc *GrafanaService,
	orch *OrchestratorService,
	log *zap.Logger,
) *WorkspaceService {
	return &WorkspaceService{
		workspace: tenant, inst: inst,
		vmSync: vmSync, vmQuery: vmQuery, provisioner: provisioner,
		grafanaResolver: grafanaResolver, grafanaSvc: grafanaSvc, orch: orch, log: log,
	}
}

func (s *WorkspaceService) resolveGrafana(ctx context.Context, instanceID *uuid.UUID) *grafana.Client {
	if s.grafanaResolver == nil {
		return nil
	}
	client, err := s.grafanaResolver(ctx, instanceID)
	if err != nil {
		if s.log != nil {
			s.log.Warn("grafana_resolver_failed", zap.Error(err))
		}
		return nil
	}
	return client
}

func (s *WorkspaceService) InsertURL(vmuserID string) string {
	if s.vmSync == nil {
		return ""
	}
	return s.vmSync.InsertURL(vmuserID)
}

// Create 创建工作空间：校验 → 生成 vmuser → 落库 → 外部系统编排。
func (s *WorkspaceService) Create(ctx context.Context, req *CreateWorkspaceRequest) (*model.Workspace, error) {
	if req.WorkspaceName == "" {
		return nil, ErrWorkspaceNameRequired
	}
	if req.TemplateType == "" || !allowedTemplateType(req.TemplateType) {
		return nil, ErrInvalidTemplateType
	}
	if err := validateQuotaJSON(req.QuotaConfig); err != nil {
		return nil, err
	}

	// Check slug uniqueness.
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.WorkspaceName), " ", "-"))
	existing, _ := s.workspace.GetBySlug(ctx, slug)
	if existing != nil {
		return nil, ErrWorkspaceSlugConflict
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

	w := &model.Workspace{
		WorkspaceName:     strings.TrimSpace(req.WorkspaceName),
		Slug:              slug,
		VMUserID:          vmuserID,
		VMUserKey:         vmKey,
		TemplateType:      req.TemplateType,
		QuotaConfig:       quotaCfg,
		IsolationLevel:    isolationLevelForTemplate(req.TemplateType),
		GrafanaInstanceID: req.GrafanaInstanceID,
		Status:            "creating",
	}
	if err := s.workspace.Create(ctx, w); err != nil {
		return nil, err
	}

	if s.provisioner != nil {
		if err := s.provisioner.ProvisionCreate(ctx, w); err != nil {
			s.markStatus(ctx, w, "failed")
			return nil, ErrWorkspaceProvisionFailed
		}
		if err := s.workspace.Update(ctx, w); err != nil {
			s.markStatus(ctx, w, "failed")
			return nil, err
		}
	}

	if s.vmSync != nil {
		if err := s.vmSync.OnWorkspaceCreated(ctx, w); err != nil {
			s.markStatus(ctx, w, "failed")
			return nil, ErrWorkspaceProvisionFailed
		}
	}

	// Create Grafana Org if grafana is enabled (non-fatal: workspace still usable without Grafana).
	if s.grafanaSvc != nil {
		orgID, err := s.grafanaSvc.CreateOrgForWorkspace(ctx, w.ID)
		if err != nil {
			s.log.Warn("workspace_grafana_org_create_failed",
				zap.String("workspace_id", w.ID.String()),
				zap.Error(err))
		} else {
			w.GrafanaOrgID = orgID
			_ = s.workspace.Update(ctx, w)
		}
	}

	if s.orch != nil {
		if err := s.orch.DeployWorkspace(ctx, w); err != nil {
			if s.log != nil {
				s.log.Warn("orchestrator_deploy_failed", zap.Error(err), zap.String("workspace_id", w.ID.String()))
			}
		}
	}
	w.Status = "active"
	if err := s.workspace.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func isolationLevelForTemplate(templateType string) string {
	switch templateType {
	case "dedicated_single", "dedicated_cluster":
		return "dedicated"
	default:
		return "shared"
	}
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

func (s *WorkspaceService) allocVMUserID(ctx context.Context) (string, error) {
	for i := 0; i < 8; i++ {
		id := "vmuser-" + uuid.New().String()
		exist, err := s.workspace.GetByVMUserID(ctx, id)
		if err != nil {
			return "", err
		}
		if exist == nil {
			return id, nil
		}
	}
	return "", errors.New("failed to allocate vmuser_id")
}

func (s *WorkspaceService) Get(ctx context.Context, id uuid.UUID) (*model.Workspace, error) {
	w, err := s.workspace.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWorkspaceNotFound
	}
	return w, nil
}

func (s *WorkspaceService) List(ctx context.Context, page, pageSize int, templateType, status, keyword string) ([]model.Workspace, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.workspace.List(ctx, repository.WorkspaceListFilter{
		TemplateType: templateType,
		Status:       status,
		Keyword:      keyword,
		Offset:       offset,
		Limit:        pageSize,
	})
}

func (s *WorkspaceService) Update(ctx context.Context, id uuid.UUID, req *UpdateWorkspaceRequest) (*model.Workspace, error) {
	w, err := s.workspace.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWorkspaceNotFound
	}
	if req.WorkspaceName == "" {
		return nil, ErrWorkspaceNameRequired
	}
	if req.TemplateType != "" && !allowedTemplateType(req.TemplateType) {
		return nil, ErrInvalidTemplateType
	}
	if err := validateQuotaJSON(req.QuotaConfig); err != nil {
		return nil, err
	}
	w.WorkspaceName = strings.TrimSpace(req.WorkspaceName)
	if req.TemplateType != "" {
		w.TemplateType = req.TemplateType
	}
	if req.QuotaConfig != "" {
		w.QuotaConfig = req.QuotaConfig
	}
	if req.Status != "" {
		w.Status = req.Status
	}
	if req.GrafanaInstanceID != nil {
		w.GrafanaInstanceID = req.GrafanaInstanceID
	}
	if err := s.workspace.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Delete 删除工作空间（无实例挂载时）。
func (s *WorkspaceService) Delete(ctx context.Context, id uuid.UUID) error {
	w, err := s.workspace.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrWorkspaceNotFound
	}
	n, err := s.inst.CountByWorkspaceID(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrWorkspaceHasInstances
	}
	w.Status = "deprovisioning"
	if err := s.workspace.Update(ctx, w); err != nil {
		return err
	}

	// Clean up Grafana Org.
	if s.grafanaSvc != nil && w.GrafanaOrgID > 0 {
		if err := s.grafanaSvc.DeleteOrg(ctx, w.GrafanaOrgID); err != nil {
			s.log.Warn("workspace_grafana_org_delete_failed",
				zap.String("workspace_id", w.ID.String()),
				zap.Int64("org_id", w.GrafanaOrgID),
				zap.Error(err))
		}
	}

	if s.provisioner != nil {
		if err := s.provisioner.ProvisionDelete(ctx, w); err != nil {
			s.markStatus(ctx, w, "failed")
			return ErrWorkspaceDeprovisionFailed
		}
	}
	if s.vmSync != nil {
		if err := s.vmSync.OnWorkspaceDeleted(ctx, w); err != nil {
			s.markStatus(ctx, w, "deprovision_failed")
			return ErrWorkspaceDeprovisionFailed
		}
	}
	if s.orch != nil {
		if err := s.orch.DeleteWorkspace(ctx, w); err != nil {
			if s.log != nil {
				s.log.Warn("orchestrator_delete_failed", zap.Error(err), zap.String("workspace_id", w.ID.String()))
			}
			s.markStatus(ctx, w, "deprovision_failed")
			return ErrWorkspaceDeprovisionFailed
		}
	}
	return s.workspace.Delete(ctx, id)
}

func (s *WorkspaceService) markStatus(ctx context.Context, w *model.Workspace, status string) {
	if w == nil {
		return
	}
	w.Status = status
	if err := s.workspace.Update(ctx, w); err != nil && s.log != nil {
		s.log.Warn("workspace_update_status_failed", zap.String("workspace_id", w.ID.String()), zap.String("status", status), zap.Error(err))
	}
}

type WorkspaceMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	SeriesCount        int64   `json:"series_count"`
	IngestQPS          float64 `json:"ingest_qps"`
	Note               string  `json:"note"`
}

func (s *WorkspaceService) GetMetrics(ctx context.Context, id uuid.UUID) (*WorkspaceMetrics, error) {
	w, err := s.workspace.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWorkspaceNotFound
	}
	if s.vmQuery == nil || !s.vmQuery.Enabled() {
		return &WorkspaceMetrics{Note: "victoriametrics query client is not configured"}, nil
	}
	series, _ := s.vmQuery.Scalar(ctx, w, `count({__name__!=""})`)
	ingest, _ := s.vmQuery.Scalar(ctx, w, `sum(rate(vm_rows_inserted_total[5m]))`)
	return &WorkspaceMetrics{
		SeriesCount: int64(series),
		IngestQPS:   ingest,
		Note:        "queried from Workspace-scoped VictoriaMetrics endpoint",
	}, nil
}
