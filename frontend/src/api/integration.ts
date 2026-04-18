import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface IntegrationTemplate {
  id: string;
  name: string;
  display_name: string;
  category: string;
  component: string;
  description: string;
  icon: string;
  latest_version: string;
  tags: string;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IntegrationTemplateVersion {
  id: string;
  template_id: string;
  version: string;
  collector_spec: string;
  alert_spec: string;
  dashboard_spec: string;
  variables: string;
  changelog: string;
  signature: string;
  created_at: string;
}

export interface IntegrationInstallation {
  id: string;
  template_id: string;
  template_version: string;
  instance_id: string;
  tenant_id: string;
  grafana_host_id: string | null;
  grafana_org_id: number;
  installed_parts: string;
  variables: string;
  status: string;
  installed_by: string;
  last_revision_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface IntegrationInstallationRevision {
  id: string;
  installation_id: string;
  version: string;
  action: string;
  spec_diff: string;
  applied_resources: string;
  operator: string;
  status: string;
  error_message: string;
  created_at: string;
}

export interface IntegrationCategory {
  key: string;
  label: string;
}

export interface RenderedResource {
  part: string;
  kind: string;
  apiVersion?: string;
  name: string;
  yaml?: string;
  dashboard?: string;
}

export interface AppliedRef {
  part: string;
  target: string;
  apiVersion?: string;
  kind?: string;
  namespace?: string;
  name?: string;
  uid?: string;
  grafana_org?: number;
  grafana_host_id?: string;
  cluster_id?: string;
  action?: string;
  status?: string;
  error?: string;
}

export interface PreflightIssue {
  part: string;
  apiVersion?: string;
  kind?: string;
  name?: string;
  reason: string;
  error?: string;
}

export interface InstallRequest {
  template_id: string;
  template_version: string;
  instance_id: string;
  tenant_id: string;
  grafana_host_id?: string;
  grafana_org_id?: number;
  installed_parts?: string[];
  values?: Record<string, string>;
  force?: boolean;
}

export interface InstallPlanResponse {
  rendered: RenderedResource[];
  preflight?: PreflightIssue[];
}

export interface InstallResponse {
  installation?: IntegrationInstallation | null;
  rendered: RenderedResource[];
  applied?: AppliedRef[];
  preflight?: PreflightIssue[];
  status: string;
}

export interface CreateTemplateRequest {
  name: string;
  display_name?: string;
  category?: string;
  component?: string;
  description?: string;
  icon?: string;
  tags?: string[];
}

export interface UpdateTemplateRequest {
  display_name?: string;
  category?: string;
  component?: string;
  description?: string;
  icon?: string;
  tags?: string[];
  status?: string;
}

export interface CreateVersionRequest {
  version: string;
  collector_spec?: string;
  alert_spec?: string;
  dashboard_spec?: string;
  variables?: string;
  changelog?: string;
}

export const integrationAPI = {
  listCategories: () =>
    api.get<ApiResponse<IntegrationCategory[]>>('/integrations/categories'),

  listTemplates: (params?: PaginationParams & { category?: string; component?: string; keyword?: string }) =>
    api.get<ApiResponse<PaginatedResponse<IntegrationTemplate>>>('/integrations/templates', { params }),

  getTemplate: (id: string) =>
    api.get<ApiResponse<IntegrationTemplate>>(`/integrations/templates/${id}`),

  createTemplate: (data: CreateTemplateRequest) =>
    api.post<ApiResponse<IntegrationTemplate>>('/integrations/templates', data),

  updateTemplate: (id: string, data: UpdateTemplateRequest) =>
    api.put<ApiResponse<IntegrationTemplate>>(`/integrations/templates/${id}`, data),

  deleteTemplate: (id: string) =>
    api.delete<ApiResponse<null>>(`/integrations/templates/${id}`),

  listVersions: (id: string) =>
    api.get<ApiResponse<IntegrationTemplateVersion[]>>(`/integrations/templates/${id}/versions`),

  createVersion: (id: string, data: CreateVersionRequest) =>
    api.post<ApiResponse<IntegrationTemplateVersion>>(`/integrations/templates/${id}/versions`, data),

  deleteVersion: (id: string, version: string) =>
    api.delete<ApiResponse<null>>(`/integrations/templates/${id}/versions/${encodeURIComponent(version)}`),

  installPlan: (data: InstallRequest) =>
    api.post<ApiResponse<InstallPlanResponse>>('/integrations/install/plan', data),

  install: (data: InstallRequest) =>
    api.post<ApiResponse<InstallResponse>>('/integrations/install', data),

  listInstallations: (params?: PaginationParams & { tenant_id?: string; instance_id?: string; template_id?: string; status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<IntegrationInstallation>>>('/integrations/installations', { params }),

  getInstallation: (id: string) =>
    api.get<ApiResponse<IntegrationInstallation>>(`/integrations/installations/${id}`),

  listInstallationRevisions: (id: string) =>
    api.get<ApiResponse<IntegrationInstallationRevision[]>>(`/integrations/installations/${id}/revisions`),

  uninstall: (id: string) =>
    api.delete<ApiResponse<null>>(`/integrations/installations/${id}`),
};

/**
 * 从 revision.applied_resources (JSON 字符串) 解析出 AppliedRef 列表；失败则返回空数组。
 */
export function parseAppliedResources(raw: string | null | undefined): AppliedRef[] {
  if (!raw) return [];
  try {
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return [];
    return arr as AppliedRef[];
  } catch {
    return [];
  }
}

/**
 * 在历次 revision（后端按 created_at DESC 返回）中取最新一次 install/upgrade 的 AppliedRef。
 * action=uninstall 的 revision 不作为"当前已应用资源"的判断依据。
 */
export function latestAppliedRefs(revisions: IntegrationInstallationRevision[]): AppliedRef[] {
  for (const r of revisions) {
    if (r.action === 'install' || r.action === 'upgrade') {
      return parseAppliedResources(r.applied_resources);
    }
  }
  return [];
}
