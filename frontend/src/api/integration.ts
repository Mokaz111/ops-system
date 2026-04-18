import api from './index';
import type { AxiosRequestConfig } from 'axios';
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
  // 第二个 config 参数主要用于传 AbortSignal，让调用方能在 useEffect cleanup 里把请求取消，
  // 避免组件 unmount / 过滤条件高频切换时旧请求覆盖新结果。
  listCategories: (config?: AxiosRequestConfig) =>
    api.get<ApiResponse<IntegrationCategory[]>>('/integrations/categories', config),

  listTemplates: (
    params?: PaginationParams & { category?: string; component?: string; keyword?: string },
    config?: AxiosRequestConfig,
  ) =>
    api.get<ApiResponse<PaginatedResponse<IntegrationTemplate>>>('/integrations/templates', { ...config, params }),

  getTemplate: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<IntegrationTemplate>>(`/integrations/templates/${id}`, config),

  createTemplate: (data: CreateTemplateRequest) =>
    api.post<ApiResponse<IntegrationTemplate>>('/integrations/templates', data),

  updateTemplate: (id: string, data: UpdateTemplateRequest) =>
    api.put<ApiResponse<IntegrationTemplate>>(`/integrations/templates/${id}`, data),

  deleteTemplate: (id: string) =>
    api.delete<ApiResponse<null>>(`/integrations/templates/${id}`),

  listVersions: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<IntegrationTemplateVersion[]>>(`/integrations/templates/${id}/versions`, config),

  createVersion: (id: string, data: CreateVersionRequest) =>
    api.post<ApiResponse<IntegrationTemplateVersion>>(`/integrations/templates/${id}/versions`, data),

  deleteVersion: (id: string, version: string) =>
    api.delete<ApiResponse<null>>(`/integrations/templates/${id}/versions/${encodeURIComponent(version)}`),

  installPlan: (data: InstallRequest) =>
    api.post<ApiResponse<InstallPlanResponse>>('/integrations/install/plan', data),

  install: (data: InstallRequest) =>
    api.post<ApiResponse<InstallResponse>>('/integrations/install', data),

  listInstallations: (
    params?: PaginationParams & { tenant_id?: string; instance_id?: string; template_id?: string; status?: string },
    config?: AxiosRequestConfig,
  ) =>
    api.get<ApiResponse<PaginatedResponse<IntegrationInstallation>>>('/integrations/installations', { ...config, params }),

  getInstallation: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<IntegrationInstallation>>(`/integrations/installations/${id}`, config),

  listInstallationRevisions: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<IntegrationInstallationRevision[]>>(`/integrations/installations/${id}/revisions`, config),

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
 * 在历次 revision（后端按 created_at DESC 返回）中取最新一次 install / upgrade / reinstall 的 AppliedRef。
 *
 * action=uninstall 的 revision 不作为"当前已应用资源"判断依据；
 * reinstall 是 stage-5 INS-1 引入的新 action（"卸载后再装回来"），
 * 资源语义与 install 等价，必须纳入，否则重装后这里会返回空数组。
 */
export function latestAppliedRefs(revisions: IntegrationInstallationRevision[]): AppliedRef[] {
  for (const r of revisions) {
    if (r.action === 'install' || r.action === 'upgrade' || r.action === 'reinstall') {
      return parseAppliedResources(r.applied_resources);
    }
  }
  return [];
}
