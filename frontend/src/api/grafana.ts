import api from './index';
import type { ApiResponse, GrafanaDashboard, GrafanaDatasource, GrafanaHealthStatus, GrafanaOrg, GrafanaOrgUser, GrafanaPlugin } from '../types/api';

function instPath(hostId: string, suffix: string) {
  return `/grafana/instances/${hostId}${suffix}`;
}

export const grafanaAPI = {
  // ── Orgs ──
  listOrgs: (hostId: string) =>
    api.get<ApiResponse<GrafanaOrg[]>>(instPath(hostId, '/orgs')),

  createOrg: (hostId: string, name: string) =>
    api.post<ApiResponse<{ org_id: number }>>(instPath(hostId, '/orgs'), { name }),

  deleteOrg: (hostId: string, orgId: number) =>
    api.delete<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}`)),

  // ── Org users ──
  listOrgUsers: (hostId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaOrgUser[]>>(instPath(hostId, `/orgs/${orgId}/users`)),

  addOrgUser: (hostId: string, orgId: number, data: { login_or_email: string; role: string }) =>
    api.post<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/users`), data),

  removeOrgUser: (hostId: string, orgId: number, userId: number) =>
    api.delete<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/users/${userId}`)),

  // ── Datasources ──
  listDatasources: (hostId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaDatasource[]>>(instPath(hostId, `/orgs/${orgId}/datasources`)),

  createDatasource: (hostId: string, orgId: number, data: Partial<GrafanaDatasource>) =>
    api.post<ApiResponse<GrafanaDatasource>>(instPath(hostId, `/orgs/${orgId}/datasources`), data),

  updateDatasource: (hostId: string, orgId: number, dsId: number, data: Partial<GrafanaDatasource>) =>
    api.put<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/datasources/${dsId}`), data),

  deleteDatasource: (hostId: string, orgId: number, dsId: number) =>
    api.delete<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/datasources/${dsId}`)),

  testDatasource: (hostId: string, orgId: number, data: { name: string; type: string; url: string; access?: string }) =>
    api.post<ApiResponse<Record<string, unknown>>>(instPath(hostId, `/orgs/${orgId}/datasources/test`), data),

  // ── Dashboards ──
  listDashboards: (hostId: string, orgId: number) =>
    api.get<ApiResponse<GrafanaDashboard[]>>(instPath(hostId, `/orgs/${orgId}/dashboards`)),

  getDashboard: (hostId: string, orgId: number, uid: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(hostId, `/orgs/${orgId}/dashboards/${uid}`)),

  deleteDashboard: (hostId: string, orgId: number, uid: string) =>
    api.delete<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/dashboards/${uid}`)),

  importDashboard: (hostId: string, orgId: number, jsonData: object) =>
    api.post<ApiResponse<null>>(instPath(hostId, `/orgs/${orgId}/dashboards/import`), jsonData),

  // ── Plugins ──
  listPlugins: (hostId: string) =>
    api.get<ApiResponse<GrafanaPlugin[]>>(instPath(hostId, '/plugins')),

  installPlugin: (hostId: string, pluginId: string, version?: string) =>
    api.post<ApiResponse<null>>(instPath(hostId, `/plugins/${pluginId}/install`), version ? { version } : {}),

  uninstallPlugin: (hostId: string, pluginId: string) =>
    api.delete<ApiResponse<null>>(instPath(hostId, `/plugins/${pluginId}`)),

  // ── Health & Admin ──
  healthCheck: (hostId: string) =>
    api.get<ApiResponse<GrafanaHealthStatus>>(instPath(hostId, '/health')),

  adminStats: (hostId: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(hostId, '/admin/stats')),

  adminSettings: (hostId: string) =>
    api.get<ApiResponse<Record<string, unknown>>>(instPath(hostId, '/admin/settings')),
};
