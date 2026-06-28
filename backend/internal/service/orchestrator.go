package service

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"

	"go.uber.org/zap"
)

// OrchestratorService 租户监控栈 Helm + 命名空间编排（§2.6）。
type OrchestratorService struct {
	cfg *config.Config
	hc  *helm.Client
	kc  *k8s.Client
	log *zap.Logger
}

// NewOrchestratorService 当 orchestration.enabled=false 时返回 (nil, nil)。
func NewOrchestratorService(cfg *config.Config, log *zap.Logger) (*OrchestratorService, error) {
	if !cfg.Orchestration.Enabled {
		return nil, nil
	}
	if !cfg.Kubernetes.InCluster && strings.TrimSpace(cfg.Kubernetes.Kubeconfig) == "" {
		return nil, fmt.Errorf("orchestration enabled but kubernetes.kubeconfig is empty (incluster=false)")
	}
	hc, err := helm.NewClient(cfg.Kubernetes.Kubeconfig)
	if err != nil {
		return nil, err
	}
	kc, err := k8s.NewClient(cfg.Kubernetes.Kubeconfig, cfg.Kubernetes.InCluster)
	if err != nil {
		return nil, err
	}
	return &OrchestratorService{cfg: cfg, hc: hc, kc: kc, log: log}, nil
}

// TenantNamespace 租户专用命名空间名（DNS-1123，≤63）。
func WorkspaceNamespace(cfg *config.Config, t *model.Workspace) string {
	p := cfg.Orchestration.NamespacePrefix
	if p == "" {
		p = "ops"
	}
	compact := strings.ReplaceAll(t.ID.String(), "-", "")
	return fmt.Sprintf("%s-%s", p, compact)
}

// ReleaseName Helm release 名（与历史部署一致，便于卸载）。
func ReleaseName(t *model.Workspace) string {
	compact := strings.ReplaceAll(t.ID.String(), "-", "")
	return "ops-" + compact
}

func (s *OrchestratorService) chartRef(tt string) string {
	switch tt {
	case "shared":
		return strings.TrimSpace(s.cfg.Helm.Charts.Shared)
	case "dedicated_single":
		return strings.TrimSpace(s.cfg.Helm.Charts.DedicatedSingle)
	case "dedicated_cluster":
		return strings.TrimSpace(s.cfg.Helm.Charts.DedicatedCluster)
	default:
		return ""
	}
}

func (s *OrchestratorService) valuesFilesForTemplate(tt string) []string {
	switch tt {
	case "shared":
		return []string{"vm-shared.yaml", "vl.yaml", "n9e-edge.yaml"}
	case "dedicated_single":
		return []string{"vm-single.yaml", "vl.yaml", "grafana.yaml", "n9e-edge.yaml"}
	case "dedicated_cluster":
		return []string{"vm-cluster.yaml", "vl.yaml", "grafana.yaml", "n9e-edge.yaml"}
	default:
		return nil
	}
}

func (s *OrchestratorService) mergeEmbeddedValues(t *model.Workspace, ns string) (map[string]interface{}, error) {
	merged := map[string]interface{}{}
	for _, f := range s.valuesFilesForTemplate(t.TemplateType) {
		m, err := helm.LoadValuesYAML(f)
		if err != nil {
			return nil, fmt.Errorf("load values %s: %w", f, err)
		}
		merged = helm.MergeValues(merged, m)
	}
	// 校验 dedicated 模板 Grafana 凭据：envsubst 后 admin_password 不得为空或仍是字面量占位符。
	if t.TemplateType == "dedicated_single" || t.TemplateType == "dedicated_cluster" {
		if err := validateGrafanaAdminPassword(merged); err != nil {
			return nil, err
		}
	}
	overlay := map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":        t.ID.String(),
			"name":      t.WorkspaceName,
			"vmuser_id": t.VMUserID,
			"template":  t.TemplateType,
		},
		"ops": map[string]interface{}{
			"namespace": ns,
		},
	}
	return helm.MergeValues(merged, overlay), nil
}

// validateGrafanaAdminPassword 校验合并后的 values 中 Grafana admin_password 已被环境变量替换：
// 既不能为空，也不能仍是字面量占位符 ${GRAFANA_ADMIN_PASSWORD}。
func validateGrafanaAdminPassword(merged map[string]interface{}) error {
	g, ok := merged["grafana"].(map[string]interface{})
	if !ok {
		return nil
	}
	ini, ok := g["grafana.ini"].(map[string]interface{})
	if !ok {
		return nil
	}
	sec, ok := ini["security"].(map[string]interface{})
	if !ok {
		return nil
	}
	pw, _ := sec["admin_password"].(string)
	if strings.TrimSpace(pw) == "" || pw == "${GRAFANA_ADMIN_PASSWORD}" {
		return fmt.Errorf("grafana admin_password not configured: set GRAFANA_ADMIN_PASSWORD env before deploying dedicated grafana")
	}
	return nil
}

// DeployWorkspace 创建命名空间、配额、可选 NetworkPolicy，并按模板执行 Helm install/upgrade。
func (s *OrchestratorService) DeployWorkspace(ctx context.Context, t *model.Workspace) error {
	ns := WorkspaceNamespace(s.cfg, t)
	labels := map[string]string{
		"ops-system/tenant-id": t.ID.String(),
		"ops-system/vmuser-id": t.VMUserID,
	}
	if err := s.kc.EnsureNamespace(ctx, ns, labels); err != nil {
		return fmt.Errorf("ensure namespace: %w", err)
	}
	if err := s.kc.ApplyResourceQuota(ctx, ns, t.QuotaConfig); err != nil {
		return fmt.Errorf("resource quota: %w", err)
	}
	if s.cfg.Orchestration.ApplyNetworkPolicy {
		if err := s.kc.ApplyDefaultNetworkPolicy(ctx, ns); err != nil {
			return fmt.Errorf("network policy: %w", err)
		}
	}

	chart := s.chartRef(t.TemplateType)
	if chart == "" {
		if s.log != nil {
			s.log.Info("orchestrator_skip_helm_empty_chart",
				zap.String("tenant_id", t.ID.String()),
				zap.String("template_type", t.TemplateType))
		}
		return nil
	}

	vals, err := s.mergeEmbeddedValues(t, ns)
	if err != nil {
		return err
	}
	name := ReleaseName(t)
	if s.log != nil {
		s.log.Info("orchestrator_helm_install_or_upgrade",
			zap.String("release", name),
			zap.String("namespace", ns),
			zap.String("chart", chart))
	}
	return s.hc.InstallOrUpgrade(ctx, name, chart, ns, vals)
}

// DeleteWorkspace 卸载 Helm release 并删除命名空间。
func (s *OrchestratorService) DeleteWorkspace(ctx context.Context, t *model.Workspace) error {
	ns := WorkspaceNamespace(s.cfg, t)
	name := ReleaseName(t)

	chart := s.chartRef(t.TemplateType)
	if chart != "" {
		if err := s.hc.UninstallRelease(ctx, name, ns); err != nil {
			if s.log != nil {
				s.log.Warn("orchestrator_helm_uninstall", zap.Error(err),
					zap.String("release", name), zap.String("namespace", ns))
			}
		}
	} else if s.log != nil {
		s.log.Info("orchestrator_skip_helm_uninstall_empty_chart", zap.String("tenant_id", t.ID.String()))
	}

	if err := s.kc.DeleteNamespace(ctx, ns); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}
