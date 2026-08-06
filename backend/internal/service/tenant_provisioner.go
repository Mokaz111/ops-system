package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ops-system/backend/internal/config"
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
	vmClusters *repository.VMClusterRepository
	audit      *repository.AuditLogRepository
	routeBuild *vm.RouteBuilder
	operator   *vm.VMOperatorClient
	vmCfg      *config.VMConfig
	log        *zap.Logger
}

func NewWorkspaceProvisioner(
	tasks *repository.ProvisioningTaskRepository,
	routes *repository.VMRouteRepository,
	datasource *repository.DatasourceRepository,
	vmClusters *repository.VMClusterRepository,
	audit *repository.AuditLogRepository,
	routeBuild *vm.RouteBuilder,
	operator *vm.VMOperatorClient,
	vmCfg *config.VMConfig,
	log *zap.Logger,
) *WorkspaceProvisioner {
	if log == nil {
		log = zap.NewNop()
	}
	if routeBuild == nil {
		routeBuild = vm.NewRouteBuilder(nil)
	}
	return &WorkspaceProvisioner{
		tasks: tasks, routes: routes, datasource: datasource, vmClusters: vmClusters,
		audit: audit, routeBuild: routeBuild, operator: operator, vmCfg: vmCfg, log: log,
	}
}

func (p *WorkspaceProvisioner) ProvisionCreate(ctx context.Context, t *model.Workspace) error {
	if p == nil || t == nil {
		return nil
	}
	routes := p.routeBuild.BuildWorkspaceRoutes(t)
	poolTarget := vm.ResolvePoolTarget("", p.vmCfg)
	t.VMNamespace = routes.Namespace
	t.VMInsertURL = routes.InsertURL
	t.VMSelectURL = routes.SelectURL
	if t.IsolationLevel == "" {
		t.IsolationLevel = "shared"
	}

	var vmClusterID *uuid.UUID

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
				TenantID:    t.ID,
				VMClusterID: vmClusterID,
				RouteType:   item.typ,
				Path:        item.path,
				AuthType:    "basic",
				SecretRef:   routes.Namespace + "/" + t.VMUserID + "-auth",
				Status:      "active",
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
		return p.operator.ApplyWorkspaceUser(ctx, t, routes, poolTarget)
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

// ProvisionCreateWithPool 在指定 Zone 共享池上下文中开通 Workspace。
func (p *WorkspaceProvisioner) ProvisionCreateWithPool(ctx context.Context, t *model.Workspace, pool *model.VMCluster) error {
	if pool != nil && pool.TargetURL != "" {
		routes := p.routeBuild.BuildWorkspaceRoutes(t)
		if pool.Namespace != "" {
			routes.Namespace = pool.Namespace
			t.VMNamespace = pool.Namespace
		}
		t.VMInsertURL = routes.InsertURL
		t.VMSelectURL = routes.SelectURL
		if t.IsolationLevel == "" {
			t.IsolationLevel = "shared"
		}
		poolTarget := pool.TargetURL
		vmClusterID := pool.ID

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
					TenantID:    t.ID,
					VMClusterID: &vmClusterID,
					RouteType:   item.typ,
					Path:        item.path,
					AuthType:    "basic",
					SecretRef:   routes.Namespace + "/" + t.VMUserID + "-auth",
					Status:      "active",
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
			return p.operator.ApplyWorkspaceUser(ctx, t, routes, poolTarget)
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
	return p.ProvisionCreate(ctx, t)
}

func (p *WorkspaceProvisioner) ProvisionDelete(ctx context.Context, t *model.Workspace) error {
	if p == nil || t == nil {
		return nil
	}
	return p.runStep(ctx, t.ID, "tenant", "delete", "deprovision_vm_resources", nil, func() error {
		ns := t.VMNamespace
		if ns == "" {
			ns = p.routeBuild.BuildWorkspaceRoutes(t).Namespace
		}
		if p.operator != nil && p.operator.Enabled() && t.VMUserID != "" {
			if err := p.operator.DeleteVMUser(ctx, t.VMUserID, ns); err != nil {
				p.log.Warn("tenant_deprovision_vmuser_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
			}
		}
		if p.routes != nil {
			if err := p.routes.DeactivateByTenant(ctx, t.ID); err != nil {
				return fmt.Errorf("deactivate vm routes: %w", err)
			}
		}
		if p.datasource != nil {
			if err := p.datasource.DeactivateByTenant(ctx, t.ID); err != nil {
				return fmt.Errorf("deactivate datasources: %w", err)
			}
		}
		_ = p.auditCreate(ctx, &t.ID, "tenant.deprovisioned", "tenant", t.ID.String(), nil)
		return nil
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
