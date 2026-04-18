package integration

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/k8s"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GrafanaClientResolver 根据 host id 返回对应的 grafana.Client。
// 当 hostID 为 nil 时可返回平台默认 client。
// 返回的 client 可能为 nil（表示回退到默认），err 非空则视为不可用。
type GrafanaClientResolver func(ctx context.Context, hostID *uuid.UUID) (*grafana.Client, error)

// K8sClientResolver 根据 cluster id 返回对应的 k8s.Client。
// 当 clusterID 为 nil 时返回平台默认 client。
type K8sClientResolver func(ctx context.Context, clusterID *uuid.UUID) (*k8s.Client, error)

// AppliedRef 已经 Apply 成功的资源引用，供 Uninstall 时反向清理。
type AppliedRef struct {
	Part          string `json:"part"`                     // collector / vmrule / dashboard / n9e
	Target        string `json:"target"`                   // k8s / grafana
	APIVersion    string `json:"apiVersion,omitempty"`     // K8s 资源的 apiVersion
	Kind          string `json:"kind,omitempty"`           // K8s 资源的 Kind
	Namespace     string `json:"namespace,omitempty"`      // K8s namespace
	Name          string `json:"name,omitempty"`           // K8s 资源 / Grafana dashboard title
	UID           string `json:"uid,omitempty"`            // K8s UID / Grafana dashboard uid
	GrafanaOrg    int64  `json:"grafana_org,omitempty"`    // Grafana 组织 ID
	GrafanaHostID string `json:"grafana_host_id,omitempty"` // Grafana 主机 id（空=默认平台）
	ClusterID     string `json:"cluster_id,omitempty"`     // 目标集群 id（空=默认集群）
	Action        string `json:"action,omitempty"`         // created / updated / imported
	Status        string `json:"status,omitempty"`         // success / failed
	Error         string `json:"error,omitempty"`
}

// ApplyOptions 单次安装的应用选项。
type ApplyOptions struct {
	DefaultNamespace string
	GrafanaOrgID     int64
	GrafanaHostID    *uuid.UUID
	ClusterID        *uuid.UUID
}

// PreflightIssue 预检发现的问题。
type PreflightIssue struct {
	Part       string `json:"part"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name,omitempty"`
	Reason     string `json:"reason"` // crd_not_found / invalid_api_version / grafana_disabled / k8s_unavailable
	Error      string `json:"error,omitempty"`
}

// Applier 抽象；测试可替换为 fake。
type Applier interface {
	Preflight(ctx context.Context, rendered []RenderedResource, opts ApplyOptions) []PreflightIssue
	Apply(ctx context.Context, rendered []RenderedResource, opts ApplyOptions) ([]AppliedRef, error)
	Delete(ctx context.Context, refs []AppliedRef) error
	Enabled() bool
}

// CompositeApplier 将 K8s 资源与 Grafana dashboard 分别下发。
type CompositeApplier struct {
	defaultK8s      *k8s.Client
	k8sResolver     K8sClientResolver
	defaultGrafana  *grafana.Client
	grafanaResolver GrafanaClientResolver
	log             *zap.Logger
}

// NewCompositeApplier 构造 Applier；任意一端为空仍可运行（对应部件会被跳过并记录 skip）。
// grafanaResolver / k8sResolver 为 nil 时，始终使用默认 client。
func NewCompositeApplier(
	defaultK8s *k8s.Client,
	k8sResolver K8sClientResolver,
	defaultGrafana *grafana.Client,
	grafanaResolver GrafanaClientResolver,
	log *zap.Logger,
) *CompositeApplier {
	if log == nil {
		log = zap.NewNop()
	}
	return &CompositeApplier{
		defaultK8s:      defaultK8s,
		k8sResolver:     k8sResolver,
		defaultGrafana:  defaultGrafana,
		grafanaResolver: grafanaResolver,
		log:             log,
	}
}

// Enabled 只要有任意后端可用即认为可 Apply。
func (a *CompositeApplier) Enabled() bool {
	if a == nil {
		return false
	}
	return a.defaultK8s != nil || a.k8sResolver != nil ||
		(a.defaultGrafana != nil && a.defaultGrafana.Enabled()) || a.grafanaResolver != nil
}

// resolveGrafana 根据 hostID 挑选 client；失败或返回 nil 则回退到默认 client。
func (a *CompositeApplier) resolveGrafana(ctx context.Context, hostID *uuid.UUID) *grafana.Client {
	if a.grafanaResolver != nil {
		if cli, err := a.grafanaResolver(ctx, hostID); err == nil && cli != nil {
			return cli
		} else if err != nil {
			a.log.Warn("integration_grafana_resolver_failed", zap.Error(err))
		}
	}
	return a.defaultGrafana
}

// resolveK8s 根据 clusterID 挑选 client；失败或返回 nil 则回退到默认 client。
func (a *CompositeApplier) resolveK8s(ctx context.Context, clusterID *uuid.UUID) *k8s.Client {
	if a.k8sResolver != nil {
		if cli, err := a.k8sResolver(ctx, clusterID); err == nil && cli != nil {
			return cli
		} else if err != nil {
			a.log.Warn("integration_k8s_resolver_failed", zap.Error(err))
		}
	}
	return a.defaultK8s
}

// Preflight 校验 k8s CRD 是否注册、grafana 是否可用等；不抛异常，只返回问题列表。
func (a *CompositeApplier) Preflight(ctx context.Context, rendered []RenderedResource, opts ApplyOptions) []PreflightIssue {
	var issues []PreflightIssue
	k8sCli := a.resolveK8s(ctx, opts.ClusterID)
	// k8s 侧：按 (apiVersion, kind) 去重做 GVR 解析。
	if k8sCli != nil {
		seen := map[string]bool{}
		for _, r := range rendered {
			if r.Part == "dashboard" || r.Part == "n9e" {
				continue
			}
			if r.APIVersion == "" || r.Kind == "" {
				continue
			}
			key := r.APIVersion + "|" + r.Kind
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, _, err := k8sCli.ResolveGVRByString(r.APIVersion, r.Kind); err != nil {
				issues = append(issues, PreflightIssue{
					Part:       r.Part,
					APIVersion: r.APIVersion,
					Kind:       r.Kind,
					Reason:     "crd_not_found",
					Error:      err.Error(),
				})
			}
		}
	} else {
		// 若渲染结果包含 k8s 资源但 k8s client 缺失，也记录。
		for _, r := range rendered {
			if r.Part == "dashboard" || r.Part == "n9e" {
				continue
			}
			issues = append(issues, PreflightIssue{
				Part:       r.Part,
				APIVersion: r.APIVersion,
				Kind:       r.Kind,
				Name:       r.Name,
				Reason:     "k8s_unavailable",
				Error:      "k8s client not configured",
			})
			break
		}
	}
	// grafana 侧：若有 dashboard 但 grafana 未配置或 orgID 缺失。
	hasDashboard := false
	for _, r := range rendered {
		if r.Part == "dashboard" {
			hasDashboard = true
			break
		}
	}
	if hasDashboard {
		cli := a.resolveGrafana(ctx, opts.GrafanaHostID)
		if cli == nil || !cli.Enabled() {
			issues = append(issues, PreflightIssue{
				Part:   "dashboard",
				Reason: "grafana_disabled",
				Error:  "grafana client not configured or disabled",
			})
		} else if opts.GrafanaOrgID <= 0 {
			issues = append(issues, PreflightIssue{
				Part:   "dashboard",
				Reason: "grafana_org_missing",
				Error:  "grafana_org_id required for dashboard apply",
			})
		}
	}
	return issues
}

// Apply 将渲染结果分派到对应后端。
// 返回的 AppliedRef 列表中 Status 可能为 failed；调用方决定是否作为整体失败。
func (a *CompositeApplier) Apply(ctx context.Context, rendered []RenderedResource, opts ApplyOptions) ([]AppliedRef, error) {
	refs := make([]AppliedRef, 0, len(rendered))
	var firstErr error
	for _, r := range rendered {
		switch r.Part {
		case "dashboard":
			ref := a.applyDashboard(ctx, r, opts)
			refs = append(refs, ref)
			if ref.Status == "failed" && firstErr == nil {
				firstErr = fmt.Errorf("dashboard %s apply failed: %s", r.Name, ref.Error)
			}
		case "n9e":
			refs = append(refs, AppliedRef{
				Part:   r.Part,
				Target: "n9e",
				Name:   r.Name,
				Status: "success",
				Action: "skipped",
				Error:  "n9e apply not implemented",
			})
		default:
			ref := a.applyK8s(ctx, r, opts)
			refs = append(refs, ref)
			if ref.Status == "failed" && firstErr == nil {
				firstErr = fmt.Errorf("k8s %s/%s apply failed: %s", r.Kind, r.Name, ref.Error)
			}
		}
	}
	return refs, firstErr
}

func (a *CompositeApplier) applyK8s(ctx context.Context, r RenderedResource, opts ApplyOptions) AppliedRef {
	ref := AppliedRef{
		Part:       r.Part,
		Target:     "k8s",
		APIVersion: r.APIVersion,
		Kind:       r.Kind,
		Name:       r.Name,
	}
	if opts.ClusterID != nil {
		ref.ClusterID = opts.ClusterID.String()
	}
	cli := a.resolveK8s(ctx, opts.ClusterID)
	if cli == nil {
		ref.Status = "failed"
		ref.Error = "k8s client not configured"
		return ref
	}
	yaml := strings.TrimSpace(r.YAML)
	if yaml == "" {
		ref.Status = "failed"
		ref.Error = "empty yaml"
		return ref
	}
	results, err := cli.ApplyYAML(ctx, yaml, opts.DefaultNamespace)
	if err != nil {
		ref.Status = "failed"
		ref.Error = err.Error()
		a.log.Warn("integration_apply_k8s_failed",
			zap.String("part", r.Part),
			zap.String("kind", r.Kind),
			zap.String("name", r.Name),
			zap.Error(err))
		return ref
	}
	if len(results) > 0 {
		res := results[0]
		ref.APIVersion = res.APIVersion
		ref.Kind = res.Kind
		ref.Namespace = res.Namespace
		ref.Name = res.Name
		ref.UID = res.UID
		ref.Action = res.Action
	}
	ref.Status = "success"
	return ref
}

func (a *CompositeApplier) applyDashboard(ctx context.Context, r RenderedResource, opts ApplyOptions) AppliedRef {
	ref := AppliedRef{
		Part:       r.Part,
		Target:     "grafana",
		Name:       r.Name,
		GrafanaOrg: opts.GrafanaOrgID,
	}
	if opts.GrafanaHostID != nil {
		ref.GrafanaHostID = opts.GrafanaHostID.String()
	}
	cli := a.resolveGrafana(ctx, opts.GrafanaHostID)
	if cli == nil || !cli.Enabled() {
		ref.Status = "failed"
		ref.Error = "grafana client not configured or disabled"
		return ref
	}
	if opts.GrafanaOrgID <= 0 {
		ref.Status = "failed"
		ref.Error = "grafana_org_id required for dashboard apply"
		return ref
	}
	body := r.Dashboard
	if strings.TrimSpace(body) == "" {
		body = r.YAML
	}
	if strings.TrimSpace(body) == "" {
		ref.Status = "failed"
		ref.Error = "empty dashboard body"
		return ref
	}
	out, err := cli.ImportDashboardJSONWithResult(ctx, opts.GrafanaOrgID, []byte(body))
	if err != nil {
		ref.Status = "failed"
		ref.Error = err.Error()
		a.log.Warn("integration_apply_dashboard_failed",
			zap.Int64("org_id", opts.GrafanaOrgID),
			zap.String("name", r.Name),
			zap.Error(err))
		return ref
	}
	if out != nil {
		ref.UID = out.UID
	}
	ref.Action = "imported"
	ref.Status = "success"
	return ref
}

// Delete 按 AppliedRef 反向清理（失败记录但继续）。
func (a *CompositeApplier) Delete(ctx context.Context, refs []AppliedRef) error {
	var firstErr error
	for _, ref := range refs {
		if ref.Status == "failed" {
			continue
		}
		switch ref.Target {
		case "grafana":
			if ref.UID == "" {
				continue
			}
			var hostID *uuid.UUID
			if ref.GrafanaHostID != "" {
				if id, err := uuid.Parse(ref.GrafanaHostID); err == nil {
					hostID = &id
				}
			}
			cli := a.resolveGrafana(ctx, hostID)
			if cli == nil || !cli.Enabled() {
				continue
			}
			if err := cli.DeleteDashboardByUID(ctx, ref.GrafanaOrg, ref.UID); err != nil {
				a.log.Warn("integration_delete_dashboard_failed",
					zap.String("uid", ref.UID),
					zap.Error(err))
				if firstErr == nil {
					firstErr = err
				}
			}
		case "k8s":
			var clusterID *uuid.UUID
			if ref.ClusterID != "" {
				if id, err := uuid.Parse(ref.ClusterID); err == nil {
					clusterID = &id
				}
			}
			cli := a.resolveK8s(ctx, clusterID)
			if cli == nil {
				continue
			}
			if err := cli.DeleteByGVK(ctx, ref.APIVersion, ref.Kind, ref.Namespace, ref.Name); err != nil {
				a.log.Warn("integration_delete_k8s_failed",
					zap.String("kind", ref.Kind),
					zap.String("name", ref.Name),
					zap.Error(err))
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	return firstErr
}

// NoopApplier 当 k8s / grafana 不可用时保留 B 方案行为。
type NoopApplier struct{}

// NewNoopApplier 构造 Applier 的无操作实现。
func NewNoopApplier() *NoopApplier { return &NoopApplier{} }

// Preflight 在 Noop 模式下不返回问题（视为仅渲染，不做真实下发）。
func (NoopApplier) Preflight(_ context.Context, _ []RenderedResource, _ ApplyOptions) []PreflightIssue {
	return nil
}

// Apply 不做任何事，返回渲染结果对应的 rendered 状态。
func (NoopApplier) Apply(_ context.Context, rendered []RenderedResource, _ ApplyOptions) ([]AppliedRef, error) {
	refs := make([]AppliedRef, 0, len(rendered))
	for _, r := range rendered {
		target := "k8s"
		if r.Part == "dashboard" {
			target = "grafana"
		}
		refs = append(refs, AppliedRef{
			Part:       r.Part,
			Target:     target,
			APIVersion: r.APIVersion,
			Kind:       r.Kind,
			Name:       r.Name,
			Action:     "rendered",
			Status:     "success",
		})
	}
	return refs, nil
}

// Delete 不做任何事。
func (NoopApplier) Delete(_ context.Context, _ []AppliedRef) error { return nil }

// Enabled 总是返回 false，便于上层识别此实现。
func (NoopApplier) Enabled() bool { return false }
