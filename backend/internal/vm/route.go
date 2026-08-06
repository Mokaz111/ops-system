package vm

import (
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
)

type RouteSet struct {
	InsertURL string
	SelectURL string
	RuleName  string
	Namespace string
}

type RouteBuilder struct {
	baseURL       string
	poolNamespace string
}

func NewRouteBuilder(cfg *config.VMConfig) *RouteBuilder {
	base := ""
	ns := "monitoring"
	if cfg != nil {
		base = cfg.VMAuthBaseURL
		if strings.TrimSpace(cfg.SharedPoolNamespace) != "" {
			ns = strings.TrimSpace(cfg.SharedPoolNamespace)
		}
	}
	return &RouteBuilder{
		baseURL:       strings.TrimRight(strings.TrimSpace(base), "/"),
		poolNamespace: ns,
	}
}

func (b *RouteBuilder) BuildWorkspaceRoutes(t *model.Workspace) RouteSet {
	if t == nil {
		return RouteSet{}
	}
	ns := strings.TrimSpace(t.VMNamespace)
	if ns == "" {
		ns = b.sharedPoolNamespace()
	}
	return RouteSet{
		InsertURL: InsertURL(b.baseURL, t.VMUserID),
		SelectURL: SelectURL(b.baseURL, t.VMUserID),
		RuleName:  "tenant-" + strings.ToLower(strings.ReplaceAll(t.VMUserID, "_", "-")),
		Namespace: ns,
	}
}

func (b *RouteBuilder) sharedPoolNamespace() string {
	if b.poolNamespace != "" {
		return b.poolNamespace
	}
	return "monitoring"
}
