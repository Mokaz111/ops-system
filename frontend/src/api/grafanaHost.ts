import api from './index';
import type { AxiosRequestConfig } from 'axios';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface GrafanaHost {
  id: string;
  name: string;
  scope: 'platform' | 'tenant';
  tenant_id: string | null;
  url: string;
  admin_user: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export const grafanaHostAPI = {
  list: (params?: PaginationParams & { scope?: string; tenant_id?: string }, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<PaginatedResponse<GrafanaHost>>>('/grafana/instances', { ...config, params }),

  get: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<GrafanaHost>>(`/grafana/instances/${id}`, config),

  create: (data: {
    name: string;
    scope: 'platform' | 'tenant';
    tenant_id?: string;
    url: string;
    admin_user?: string;
    admin_password?: string;
    admin_token?: string;
  }) => api.post<ApiResponse<GrafanaHost>>('/grafana/instances', data),

  update: (id: string, data: Partial<GrafanaHost> & { admin_password?: string; admin_token?: string }) =>
    api.put<ApiResponse<GrafanaHost>>(`/grafana/instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/grafana/instances/${id}`),

  login: (id: string) =>
    api.post<ApiResponse<{ url: string; user: string; password: string }>>(`/grafana/instances/${id}/login`),
};
