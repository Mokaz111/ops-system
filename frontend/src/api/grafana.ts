import api from './index';
import type { ApiResponse, GrafanaDatasource, GrafanaOrg, GrafanaOrgUser } from '../types/api';

export const grafanaAPI = {
  listOrgs: () =>
    api.get<ApiResponse<GrafanaOrg[]>>('/grafana/orgs'),

  createOrg: (name: string) =>
    api.post<ApiResponse<GrafanaOrg>>('/grafana/orgs', { name }),

  deleteOrg: (id: number) =>
    api.delete<ApiResponse<null>>(`/grafana/orgs/${id}`),

  listOrgUsers: (orgId: number) =>
    api.get<ApiResponse<GrafanaOrgUser[]>>(`/grafana/orgs/${orgId}/users`),

  addOrgUser: (orgId: number, data: { loginOrEmail: string; role: string }) =>
    api.post<ApiResponse<null>>(`/grafana/orgs/${orgId}/users`, data),

  removeOrgUser: (orgId: number, userId: number) =>
    api.delete<ApiResponse<null>>(`/grafana/orgs/${orgId}/users/${userId}`),

  listDatasources: (orgId: number) =>
    api.get<ApiResponse<GrafanaDatasource[]>>(`/grafana/orgs/${orgId}/datasources`),

  createDatasource: (orgId: number, data: Partial<GrafanaDatasource>) =>
    api.post<ApiResponse<GrafanaDatasource>>(`/grafana/orgs/${orgId}/datasources`, data),

  deleteDatasource: (orgId: number, dsId: number) =>
    api.delete<ApiResponse<null>>(`/grafana/orgs/${orgId}/datasources/${dsId}`),

  importDashboard: (orgId: number, data: { dashboard: object; overwrite?: boolean }) =>
    api.post<ApiResponse<null>>(`/grafana/orgs/${orgId}/dashboards/import`, data),
};
