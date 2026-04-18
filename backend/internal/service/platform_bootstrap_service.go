package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/k8s"
)

var (
	ErrHelmOperatorNotConfigured = errors.New("helm operator is not configured")
	ErrInvalidNamespace          = errors.New("invalid namespace")
	ErrInvalidReleaseName        = errors.New("invalid release_name")
)

const (
	defaultSharedStackNamespace = "monitoring"
	defaultSharedStackRelease   = "vm-shared-stack"
	defaultSharedStackChart     = "vm/victoria-metrics-k8s-stack"
)

var dns1123NamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

type InitSharedClusterRequest struct {
	DryRun      bool
	Namespace   string
	ReleaseName string
}

type InitSharedClusterPlan struct {
	DryRun      bool                   `json:"dry_run"`
	Namespace   string                 `json:"namespace"`
	ReleaseName string                 `json:"release_name"`
	Chart       string                 `json:"chart"`
	Action      string                 `json:"action"`
	Values      map[string]interface{} `json:"values"`
}

type PlatformBootstrapService struct {
	helmClient *helm.Client
	k8sClient  *k8s.Client // 可选；用于安装完成后刷新 RESTMapper 缓存。
}

func NewPlatformBootstrapService(helmClient *helm.Client, k8sClient *k8s.Client) *PlatformBootstrapService {
	return &PlatformBootstrapService{helmClient: helmClient, k8sClient: k8sClient}
}

// InitSharedVMStack 初始化或升级全局共享监控集群（admin 手动触发）。
func (s *PlatformBootstrapService) InitSharedVMStack(
	ctx context.Context,
	req *InitSharedClusterRequest,
) (*InitSharedClusterPlan, error) {
	if req == nil {
		req = &InitSharedClusterRequest{}
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = defaultSharedStackNamespace
	}
	release := strings.TrimSpace(req.ReleaseName)
	if release == "" {
		release = defaultSharedStackRelease
	}
	if !isDNS1123Name(ns) {
		return nil, ErrInvalidNamespace
	}
	if !isDNS1123Name(release) {
		return nil, ErrInvalidReleaseName
	}

	values := map[string]interface{}{
		// 使用 vm/victoria-metrics-k8s-stack 统一安装共享监控栈，并启用内置 Grafana。
		"grafana": map[string]interface{}{
			"enabled": true,
		},
	}
	plan := &InitSharedClusterPlan{
		DryRun:      req.DryRun,
		Namespace:   ns,
		ReleaseName: release,
		Chart:       defaultSharedStackChart,
		Action:      "install_or_upgrade",
		Values:      values,
	}
	if req.DryRun {
		return plan, nil
	}
	if s.helmClient == nil {
		return nil, ErrHelmOperatorNotConfigured
	}
	if err := s.helmClient.InstallOrUpgrade(ctx, release, defaultSharedStackChart, ns, values); err != nil {
		return nil, err
	}
	// VictoriaMetrics operator 会注册 VMAgent/VMRule/VMPodScrape 等 CRD；
	// 刷新缓存，后续 Install 走 CompositeApplier 才能正确解析 GVR。
	if s.k8sClient != nil {
		s.k8sClient.InvalidateMapperCache()
	}
	return plan, nil
}

func isDNS1123Name(v string) bool {
	if len(v) < 1 || len(v) > 63 {
		return false
	}
	return dns1123NamePattern.MatchString(v)
}
