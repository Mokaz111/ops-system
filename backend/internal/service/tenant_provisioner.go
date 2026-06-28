package service

import (
	"context"
	"encoding/json"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/vm"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WorkspaceProvisioner struct {
	tasks      *repository.ProvisioningTaskRepository
	routes     *repository.VMRouteRepository
	datasource *repository.DatasourceRepository
	audit      *repository.AuditLogRepository
	routeBuild *vm.RouteBuilder
	operator   *vm.VMOperatorClient
	log        *zap.Logger
}

func NewWorkspaceProvisioner(
	tasks *repository.ProvisioningTaskRepository,
	routes *repository.VMRouteRepository,
	datasource *repository.DatasourceRepository,
	audit *repository.AuditLogRepository,
	routeBuild *vm.RouteBuilder,
	operator *vm.VMOperatorClient,
	log *zap.Logger,
) *WorkspaceProvisioner {
	if log == nil {
		log = zap.NewNop()
	}
	if routeBuild == nil {
		routeBuild = vm.NewRouteBuilder(nil)
	}
	return &WorkspaceProvisioner{tasks: tasks, routes: routes, datasource: datasource, audit: audit, routeBuild: routeBuild, operator: operator, log: log}
}

func (p *WorkspaceProvisioner) ProvisionCreate(ctx context.Context, t *model.Workspace) error {
	if p == nil || t == nil {
		return nil
	}
	routes := p.routeBuild.BuildWorkspaceRoutes(t)
	t.VMNamespace = routes.Namespace
	t.VMInsertURL = routes.InsertURL
	t.VMSelectURL = routes.SelectURL
	if t.IsolationLevel == "" {
		t.IsolationLevel = "shared"
	}

	if err := p.runStep(ctx, t.ID, "tenant", "create", "vm_routes", routes, func() error {
		if p.routes == nil {
			return nil
		}
		for _, item := range []struct {
			typ  string
			path string
		}{
			{typ: "insert", path: routes.InsertURL},
			{typ: "select", path: routes.SelectURL},
			{typ: "rule", path: routes.RuleName},
		} {
			if err := p.routes.Upsert(ctx, &model.VMRoute{
				TenantID:  t.ID,
				RouteType: item.typ,
				Path:      item.path,
				AuthType:  "basic",
				SecretRef: routes.Namespace + "/" + t.VMUserID + "-auth",
				Status:    "active",
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := p.runStep(ctx, t.ID, "tenant", "create", "vm_operator_user", routes, func() error {
		if p.operator == nil {
			return nil
		}
		return p.operator.ApplyWorkspaceUser(ctx, t, routes)
	}); err != nil {
		return err
	}

	if err := p.runStep(ctx, t.ID, "tenant", "create", "default_datasource", routes, func() error {
		if p.datasource == nil || routes.SelectURL == "" {
			return nil
		}
		return p.datasource.Upsert(ctx, &model.Datasource{
			TenantID: t.ID,
			Provider: "api",
			Name:     "vm-" + t.VMUserID,
			Type:     "prometheus",
			URL:      routes.SelectURL,
			AuthType: "basic",
			Status:   "active",
		})
	}); err != nil {
		return err
	}

	_ = p.auditCreate(ctx, &t.ID, "tenant.provisioned", "tenant", t.ID.String(), routes)
	return nil
}

func (p *WorkspaceProvisioner) ProvisionDelete(ctx context.Context, t *model.Workspace) error {
	if p == nil || t == nil {
		return nil
	}
	return p.runStep(ctx, t.ID, "tenant", "delete", "mark_external_resources_deleted", nil, func() error {
		_ = p.auditCreate(ctx, &t.ID, "tenant.deprovisioned", "tenant", t.ID.String(), nil)
		return nil
	})
}

func (p *WorkspaceProvisioner) runStep(ctx context.Context, tenantID uuid.UUID, resource, action, step string, payload any, fn func() error) error {
	task := &model.ProvisioningTask{
		TenantID: tenantID,
		Resource: resource,
		Action:   action,
		Step:     step,
		Status:   "pending",
		Payload:  jsonString(payload),
	}
	if p.tasks != nil {
		if err := p.tasks.Create(ctx, task); err != nil {
			return err
		}
		_ = p.tasks.MarkRunning(ctx, task.ID)
	}
	err := fn()
	if err != nil {
		if p.tasks != nil {
			_ = p.tasks.MarkFailed(ctx, task.ID, err)
		}
		p.log.Warn("tenant_provisioning_step_failed", zap.String("step", step), zap.Error(err))
		return err
	}
	if p.tasks != nil {
		_ = p.tasks.MarkSucceeded(ctx, task.ID)
	}
	return nil
}

func (p *WorkspaceProvisioner) auditCreate(ctx context.Context, tenantID *uuid.UUID, action, resource, resourceID string, details any) error {
	if p.audit == nil {
		return nil
	}
	return p.audit.Create(ctx, &model.AuditLog{
		TenantID:   tenantID,
		ActorType:  "system",
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    jsonString(details),
	})
}

func jsonString(v any) string {
	if v == nil {
		return "{}"
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
