import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface LogInstance {
  id: string;
  tenant_id: string;
  instance_name: string;
  release_name: string;
  namespace: string;
  endpoint: string;
  token: string;
  retention_days: number;
  spec: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface LogQueryResult {
  note?: string;
  results: unknown[];
}

export const logAPI = {
  list: (params?: PaginationParams & { tenant_id?: string; keyword?: string }) =>
    api.get<ApiResponse<PaginatedResponse<LogInstance>>>('/log-instances', { params }),

  get: (id: string) =>
    api.get<ApiResponse<LogInstance>>(`/log-instances/${id}`),

  create: (data: {
    tenant_id: string;
    instance_name: string;
    namespace?: string;
    release_name?: string;
    retention_days?: number;
    spec?: string;
  }) => api.post<ApiResponse<LogInstance>>('/log-instances', data),

  update: (id: string, data: Partial<LogInstance>) =>
    api.put<ApiResponse<LogInstance>>(`/log-instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/log-instances/${id}`),

  query: (id: string, data: { query: string; start?: string; end?: string; limit?: number }) =>
    api.post<ApiResponse<LogQueryResult>>(`/log-instances/${id}/query`, data),
};
