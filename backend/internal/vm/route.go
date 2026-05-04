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
	baseURL string
}

func NewRouteBuilder(cfg *config.VMConfig) *RouteBuilder {
	base := ""
	if cfg != nil {
		base = cfg.VMAuthBaseURL
	}
	return &RouteBuilder{baseURL: strings.TrimRight(strings.TrimSpace(base), "/")}
}

func (b *RouteBuilder) BuildTenantRoutes(t *model.Tenant) RouteSet {
	if t == nil {
		return RouteSet{}
	}
	ns := strings.TrimSpace(t.VMNamespace)
	if ns == "" {
		ns = "ops-tenant-" + strings.ToLower(strings.ReplaceAll(t.VMUserID, "_", "-"))
	}
	return RouteSet{
		InsertURL: InsertURL(b.baseURL, t.VMUserID),
		SelectURL: SelectURL(b.baseURL, t.VMUserID),
		RuleName:  "tenant-" + strings.ToLower(strings.ReplaceAll(t.VMUserID, "_", "-")),
		Namespace: ns,
	}
}
