package vm

import (
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
)

// PoolEndpoints 共享 VM 池内部/外部端点。
type PoolEndpoints struct {
	Namespace   string
	ReleaseName string
	VMAuthURL   string // 集群内 vmauth Service URL（VMUser targetRef）
	SelectURL   string // 对外 select 根（通常 vmauth + path）
	InsertURL   string // 对外 insert 根
}

// BuildSharedPoolEndpoints 根据 Helm release 构造共享池 URL。
// victoria-metrics-k8s-stack 默认 Service：{release}-victoria-metrics-auth:8427
func BuildSharedPoolEndpoints(namespace, releaseName, vmauthBaseURL string) PoolEndpoints {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "monitoring"
	}
	release := strings.TrimSpace(releaseName)
	if release == "" {
		release = "vm-shared-stack"
	}
	target := fmt.Sprintf("http://%s-victoria-metrics-auth.%s.svc:8427", release, ns)
	base := strings.TrimRight(strings.TrimSpace(vmauthBaseURL), "/")
	return PoolEndpoints{
		Namespace:   ns,
		ReleaseName: release,
		VMAuthURL:   target,
		SelectURL:   base,
		InsertURL:   base,
	}
}

// PoolTargetFromConfig 从配置构造默认共享池端点（无 Zone 登记时的回退）。
func PoolTargetFromConfig(cfg *config.VMConfig) PoolEndpoints {
	ns := "monitoring"
	release := "vm-shared-stack"
	targetOverride := ""
	if cfg != nil {
		if strings.TrimSpace(cfg.SharedPoolNamespace) != "" {
			ns = strings.TrimSpace(cfg.SharedPoolNamespace)
		}
		if strings.TrimSpace(cfg.SharedPoolRelease) != "" {
			release = strings.TrimSpace(cfg.SharedPoolRelease)
		}
		targetOverride = strings.TrimSpace(cfg.SharedPoolTargetURL)
	}
	ep := BuildSharedPoolEndpoints(ns, release, cfg.VMAuthBaseURL)
	if targetOverride != "" {
		ep.VMAuthURL = targetOverride
	}
	return ep
}

// ResolvePoolTarget 优先使用已登记的 VMCluster，否则回退配置。
func ResolvePoolTarget(clusterTargetURL string, cfg *config.VMConfig) string {
	if strings.TrimSpace(clusterTargetURL) != "" {
		return strings.TrimSpace(clusterTargetURL)
	}
	return PoolTargetFromConfig(cfg).VMAuthURL
}
