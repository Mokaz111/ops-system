import api from './index';
import type { AxiosRequestConfig } from 'axios';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface GrafanaInstance {
  id: string;
  name: string;
  source: 'platform' | 'external';
  zone_id: string | null;
  url: string;
  admin_user: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export const grafanaInstanceAPI = {
  list: (params?: PaginationParams & { source?: string; zone_id?: string }, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<PaginatedResponse<GrafanaInstance>>>('/grafana/instances', { ...config, params }),

  get: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<GrafanaInstance>>(`/grafana/instances/${id}`, config),

  create: (data: {
    name: string;
    source: 'platform' | 'external';
    zone_id?: string;
    url: string;
    admin_user?: string;
    admin_password?: string;
    admin_token?: string;
  }) => api.post<ApiResponse<GrafanaInstance>>('/grafana/instances', data),

  update: (id: string, data: Partial<GrafanaInstance> & { admin_password?: string; admin_token?: string }) =>
    api.put<ApiResponse<GrafanaInstance>>(`/grafana/instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/grafana/instances/${id}`),

  login: (id: string, redirect?: string) =>
    api.post<ApiResponse<{ proxyUrl: string }>>(`/grafana/instances/${id}/login`, null, {
      params: redirect ? { redirect } : undefined,
    }),
};
