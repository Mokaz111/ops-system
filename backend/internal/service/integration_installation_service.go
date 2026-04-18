package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ops-system/backend/internal/integration"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 安装记录相关业务错误。
var (
	ErrIntegrationInstallationNotFound = errors.New("integration installation not found")
)

// IntegrationInstallationService 接入中心安装记录业务。
// M3：Install/Plan 支持真实 Apply（如果 applier.Enabled()），否则退化为 B 方案仅渲染。
type IntegrationInstallationService struct {
	repo         *repository.IntegrationInstallationRepository
	templateRepo *repository.IntegrationTemplateRepository
	instanceRepo *repository.InstanceRepository
	renderer     *integration.Renderer
	applier      integration.Applier
}

func NewIntegrationInstallationService(
	repo *repository.IntegrationInstallationRepository,
	templateRepo *repository.IntegrationTemplateRepository,
	instanceRepo *repository.InstanceRepository,
	renderer *integration.Renderer,
	applier integration.Applier,
) *IntegrationInstallationService {
	if applier == nil {
		applier = integration.NewNoopApplier()
	}
	return &IntegrationInstallationService{
		repo:         repo,
		templateRepo: templateRepo,
		instanceRepo: instanceRepo,
		renderer:     renderer,
		applier:      applier,
	}
}

// InstallRequest 安装请求。
type InstallRequest struct {
	TemplateID      uuid.UUID         `json:"template_id" binding:"required"`
	TemplateVersion string            `json:"template_version" binding:"required"`
	InstanceID      uuid.UUID         `json:"instance_id" binding:"required"`
	TenantID        uuid.UUID         `json:"tenant_id" binding:"required"`
	GrafanaHostID   *uuid.UUID        `json:"grafana_host_id"`
	GrafanaOrgID    int64             `json:"grafana_org_id"`
	InstalledParts  []string          `json:"installed_parts"`
	Values          map[string]string `json:"values"`
	Force           bool              `json:"force"` // 忽略 preflight 失败继续下发
}

// PlanResult Plan（dry-run）返回。
type PlanResult struct {
	Rendered  []integration.RenderedResource `json:"rendered"`
	Preflight []integration.PreflightIssue   `json:"preflight"`
}

// InstallResult 安装返回（含渲染结果、preflight 与 apply 状态）。
type InstallResult struct {
	Installation *model.IntegrationInstallation `json:"installation,omitempty"`
	Rendered     []integration.RenderedResource `json:"rendered"`
	Applied      []integration.AppliedRef       `json:"applied,omitempty"`
	Preflight    []integration.PreflightIssue   `json:"preflight,omitempty"`
	Status       string                         `json:"status"` // success / partial / failed / rendered / preflight_failed
}

// Plan 仅渲染不落库（dry-run 预览），并返回 preflight 结果。
func (s *IntegrationInstallationService) Plan(ctx context.Context, req *InstallRequest) (*PlanResult, error) {
	spec, _, _, err := s.loadSpec(ctx, req.TemplateID, req.TemplateVersion)
	if err != nil {
		return nil, err
	}
	renderCtx, err := s.buildRenderContext(ctx, req)
	if err != nil {
		return nil, err
	}
	rendered, err := s.renderer.Render(integration.RenderInput{
		Spec:   spec,
		Values: req.Values,
		Ctx:    renderCtx,
		Parts:  req.InstalledParts,
	})
	if err != nil {
		return nil, err
	}
	clusterID := s.lookupClusterID(ctx, req.InstanceID)
	issues := s.applier.Preflight(ctx, rendered, integration.ApplyOptions{
		DefaultNamespace: renderCtx.Namespace,
		GrafanaOrgID:     req.GrafanaOrgID,
		GrafanaHostID:    req.GrafanaHostID,
		ClusterID:        clusterID,
	})
	return &PlanResult{Rendered: rendered, Preflight: issues}, nil
}

// lookupClusterID 从 instance 读 cluster_id；失败返回 nil 表示平台默认集群。
func (s *IntegrationInstallationService) lookupClusterID(ctx context.Context, instanceID uuid.UUID) *uuid.UUID {
	if s.instanceRepo == nil {
		return nil
	}
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil || inst == nil {
		return nil
	}
	return inst.ClusterID
}

// Install 渲染 + Apply + 持久化。
// 若 applier 为 NoopApplier（k8s/grafana 不可用）则退化为 B 方案仅记录 rendered 状态。
func (s *IntegrationInstallationService) Install(ctx context.Context, operator string, req *InstallRequest) (*InstallResult, error) {
	spec, tpl, version, err := s.loadSpec(ctx, req.TemplateID, req.TemplateVersion)
	if err != nil {
		return nil, err
	}
	_ = tpl
	_ = version

	renderCtx, err := s.buildRenderContext(ctx, req)
	if err != nil {
		return nil, err
	}
	rendered, err := s.renderer.Render(integration.RenderInput{
		Spec:   spec,
		Values: req.Values,
		Ctx:    renderCtx,
		Parts:  req.InstalledParts,
	})
	if err != nil {
		return nil, err
	}

	applyOpts := integration.ApplyOptions{
		DefaultNamespace: renderCtx.Namespace,
		GrafanaOrgID:     req.GrafanaOrgID,
		GrafanaHostID:    req.GrafanaHostID,
		ClusterID:        s.lookupClusterID(ctx, req.InstanceID),
	}
	issues := s.applier.Preflight(ctx, rendered, applyOpts)
	if len(issues) > 0 && !req.Force {
		return &InstallResult{
			Rendered:  rendered,
			Preflight: issues,
			Status:    "preflight_failed",
		}, nil
	}
	applied, applyErr := s.applier.Apply(ctx, rendered, applyOpts)
	overallStatus, errorMsg := summarizeApply(s.applier, applied, applyErr)

	valuesJSON := "{}"
	if b, mErr := json.Marshal(req.Values); mErr == nil {
		valuesJSON = string(b)
	}

	m := &model.IntegrationInstallation{
		TemplateID:      req.TemplateID,
		TemplateVersion: req.TemplateVersion,
		InstanceID:      req.InstanceID,
		TenantID:        req.TenantID,
		GrafanaHostID:   req.GrafanaHostID,
		GrafanaOrgID:    req.GrafanaOrgID,
		InstalledParts:  marshalJSONStringArray(collectParts(rendered)),
		Variables:       valuesJSON,
		Status:          overallStatus,
		InstalledBy:     operator,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	appliedJSON := "[]"
	if b, mErr := json.Marshal(applied); mErr == nil {
		appliedJSON = string(b)
	}
	rev := &model.IntegrationInstallationRevision{
		InstallationID:   m.ID,
		Version:          req.TemplateVersion,
		Action:           "install",
		SpecDiff:         valuesJSON,
		AppliedResources: appliedJSON,
		Operator:         operator,
		Status:           overallStatus,
		ErrorMessage:     errorMsg,
	}
	if err := s.repo.CreateRevision(ctx, rev); err != nil {
		return nil, err
	}
	m.LastRevisionID = &rev.ID
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return &InstallResult{
		Installation: m,
		Rendered:     rendered,
		Applied:      applied,
		Preflight:    issues,
		Status:       overallStatus,
	}, nil
}

// summarizeApply 根据 Apply 结果判断整体状态。
// 规则：
//   - applier 为 Noop → rendered
//   - 所有 applied.Status == success → success
//   - 部分 failed → partial
//   - 全部 failed → failed
func summarizeApply(applier integration.Applier, applied []integration.AppliedRef, applyErr error) (string, string) {
	if applier == nil || !applier.Enabled() {
		return "rendered", ""
	}
	total := len(applied)
	if total == 0 {
		if applyErr != nil {
			return "failed", applyErr.Error()
		}
		return "success", ""
	}
	successCount := 0
	failedMsgs := make([]string, 0)
	for _, r := range applied {
		if r.Status == "success" {
			successCount++
		} else if r.Status == "failed" {
			failedMsgs = append(failedMsgs, r.Kind+"/"+r.Name+": "+r.Error)
		}
	}
	errorMsg := ""
	if len(failedMsgs) > 0 {
		errorMsg = strings.Join(failedMsgs, "; ")
	}
	switch {
	case successCount == total:
		return "success", ""
	case successCount == 0:
		return "failed", errorMsg
	default:
		return "partial", errorMsg
	}
}

// loadSpec 取出 spec 结构体 + template / version。
func (s *IntegrationInstallationService) loadSpec(ctx context.Context, templateID uuid.UUID, version string) (integration.TemplateSpec, *model.IntegrationTemplate, *model.IntegrationTemplateVersion, error) {
	tpl, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return integration.TemplateSpec{}, nil, nil, err
	}
	if tpl == nil {
		return integration.TemplateSpec{}, nil, nil, ErrIntegrationTemplateNotFound
	}
	v, err := s.templateRepo.GetVersion(ctx, templateID, version)
	if err != nil {
		return integration.TemplateSpec{}, nil, nil, err
	}
	if v == nil {
		return integration.TemplateSpec{}, nil, nil, ErrIntegrationVersionNotFound
	}
	spec, err := integration.ParseSpec(v.CollectorSpec, v.AlertSpec, v.DashboardSpec, v.Variables)
	if err != nil {
		return integration.TemplateSpec{}, nil, nil, err
	}
	return spec, tpl, v, nil
}

// buildRenderContext 从目标 VM 实例信息构造渲染上下文。
func (s *IntegrationInstallationService) buildRenderContext(ctx context.Context, req *InstallRequest) (integration.RenderContext, error) {
	rc := integration.RenderContext{
		TenantID:     req.TenantID.String(),
		InstanceID:   req.InstanceID.String(),
		GrafanaOrgID: req.GrafanaOrgID,
	}
	if s.instanceRepo == nil {
		return rc, nil
	}
	inst, err := s.instanceRepo.GetByID(ctx, req.InstanceID)
	if err != nil {
		return rc, err
	}
	if inst != nil {
		rc.InstanceName = inst.InstanceName
		rc.Namespace = inst.Namespace
		rc.VMAgentURL = inst.URL
	}
	return rc, nil
}

func collectParts(rs []integration.RenderedResource) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rs {
		if r.Part == "" || seen[r.Part] {
			continue
		}
		seen[r.Part] = true
		out = append(out, r.Part)
	}
	return out
}

// Get 查询安装记录。
func (s *IntegrationInstallationService) Get(ctx context.Context, id uuid.UUID) (*model.IntegrationInstallation, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrIntegrationInstallationNotFound
	}
	return m, nil
}

// Uninstall 卸载：加载最新 revision 的 applied_resources，调用 applier.Delete 并写审计。
func (s *IntegrationInstallationService) Uninstall(ctx context.Context, id uuid.UUID, operator string) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrIntegrationInstallationNotFound
	}

	// 合并所有 install/upgrade revisions 中成功 apply 的资源，按 (target, kind, ns, name, uid)
	// 去重后反向删除；这样升级过多次也不会漏掉早期 revision 中已创建但后续 revision 未覆盖的资源。
	var refs []integration.AppliedRef
	revisions, err := s.repo.ListRevisions(ctx, id)
	if err == nil {
		seen := map[string]bool{}
		for _, r := range revisions {
			if r.Action != "install" && r.Action != "upgrade" {
				continue
			}
			if strings.TrimSpace(r.AppliedResources) == "" {
				continue
			}
			var batch []integration.AppliedRef
			if jErr := json.Unmarshal([]byte(r.AppliedResources), &batch); jErr != nil {
				continue
			}
			for _, ref := range batch {
				if ref.Status == "failed" {
					continue
				}
				key := ref.Target + "|" + ref.APIVersion + "|" + ref.Kind + "|" + ref.Namespace + "|" + ref.Name + "|" + ref.UID
				if seen[key] {
					continue
				}
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}

	var deleteErr error
	if len(refs) > 0 {
		deleteErr = s.applier.Delete(ctx, refs)
	}

	status := "uninstalled"
	errorMsg := ""
	if deleteErr != nil {
		status = "uninstall_failed"
		errorMsg = deleteErr.Error()
	}
	m.Status = status
	if err := s.repo.Update(ctx, m); err != nil {
		return err
	}
	appliedJSON := "[]"
	if b, mErr := json.Marshal(refs); mErr == nil {
		appliedJSON = string(b)
	}
	rev := &model.IntegrationInstallationRevision{
		InstallationID:   m.ID,
		Version:          m.TemplateVersion,
		Action:           "uninstall",
		AppliedResources: appliedJSON,
		Operator:         operator,
		Status:           status,
		ErrorMessage:     errorMsg,
	}
	return s.repo.CreateRevision(ctx, rev)
}

// List 分页列表。
func (s *IntegrationInstallationService) List(ctx context.Context, f repository.IntegrationInstallationListFilter, page, pageSize int) ([]model.IntegrationInstallation, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	f.Offset = (page - 1) * pageSize
	f.Limit = pageSize
	return s.repo.List(ctx, f)
}

// ListRevisions 列出安装记录的变更历史。
func (s *IntegrationInstallationService) ListRevisions(ctx context.Context, id uuid.UUID) ([]model.IntegrationInstallationRevision, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrIntegrationInstallationNotFound
	}
	return s.repo.ListRevisions(ctx, id)
}
