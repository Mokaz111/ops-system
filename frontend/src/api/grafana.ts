import api from './index';
import type { ApiResponse, GrafanaDashboard, GrafanaDatasource, GrafanaHealthStatus, GrafanaOrg, GrafanaOrgUser, GrafanaPlugin } from '../types/api';

function instPath(instanceId: string, suffix: string) {
  return `/grafana/instances/${instanceId}${suffix}`;
}

export const grafanaAPI = {
  // ── Orgs ──
  listOrgs: (instanceId: string) =>
    api.get<ApiResponse<GrafanaOrg[]>>(instPath(instanceId, '/orgs')),

  createOrg: (instanceId: string, name: string) =>
    api.post<ApiResponse<{ org_id: number }>>(instPath(instanceId, '/orgs'), { name }),

  deleteOrg: (instanceId: string, orgId: number) =>
    api.delete<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}`)),

  updateOrg: (instanceId: string, orgId: number, name: string) =>
    api.put<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}`), { name }),

  // ── Org users ──
  listOrgUsers: (instanceId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaOrgUser[]>>(instPath(instanceId, `/orgs/${orgId}/users`)),

  addOrgUser: (instanceId: string, orgId: number, data: { login_or_email: string; role: string }) =>
    api.post<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/users`), data),

  removeOrgUser: (instanceId: string, orgId: number, userId: number) =>
    api.delete<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/users/${userId}`)),

  // ── Datasources ──
  listDatasources: (instanceId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaDatasource[]>>(instPath(instanceId, `/orgs/${orgId}/datasources`)),

  createDatasource: (instanceId: string, orgId: number, data: Partial<GrafanaDatasource>) =>
    api.post<ApiResponse<GrafanaDatasource>>(instPath(instanceId, `/orgs/${orgId}/datasources`), data),

  updateDatasource: (instanceId: string, orgId: number, dsId: number, data: Partial<GrafanaDatasource>) =>
    api.put<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/datasources/${dsId}`), data),

  deleteDatasource: (instanceId: string, orgId: number, dsId: number) =>
    api.delete<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/datasources/${dsId}`)),

  testDatasource: (instanceId: string, orgId: number, data: { name: string; type: string; url: string; access?: string }) =>
    api.post<ApiResponse<Record<string, unknown>>>(instPath(instanceId, `/orgs/${orgId}/datasources/test`), data),

  // ── Dashboards ──
  listDashboards: (instanceId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaDashboard[]>>(instPath(instanceId, `/orgs/${orgId}/dashboards`)),

  getDashboard: (instanceId: string, orgId: number, uid: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(instanceId, `/orgs/${orgId}/dashboards/${uid}`)),

  deleteDashboard: (instanceId: string, orgId: number, uid: string) =>
    api.delete<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/dashboards/${uid}`)),

  importDashboard: (instanceId: string, orgId: number, jsonData: object) =>
    api.post<ApiResponse<null>>(instPath(instanceId, `/orgs/${orgId}/dashboards/import`), jsonData),

  // ── Plugins ──
  listPlugins: (instanceId: string) =>
    api.get<ApiResponse<GrafanaPlugin[]>>(instPath(instanceId, '/plugins')),

  installPlugin: (instanceId: string, pluginId: string, version?: string) =>
    api.post<ApiResponse<null>>(instPath(instanceId, `/plugins/${pluginId}/install`), version ? { version } : {}),

  uninstallPlugin: (instanceId: string, pluginId: string) =>
    api.delete<ApiResponse<null>>(instPath(instanceId, `/plugins/${pluginId}`)),

  // ── Health & Admin ──
  healthCheck: (instanceId: string) =>
    api.get<ApiResponse<GrafanaHealthStatus>>(instPath(instanceId, '/health')),

  adminStats: (instanceId: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(instanceId, '/admin/stats')),

  adminSettings: (instanceId: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(instanceId, '/admin/settings')),
};
