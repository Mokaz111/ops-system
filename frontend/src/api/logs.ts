import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface LogInstance {
  id: string;
  tenant_id: string;
  zone_id?: string;
  backend_type: string;
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

export interface LogEntry {
  time: string;
  message: string;
  fields?: Record<string, string>;
}

export interface LogQueryStats {
  returned: number;
  limit: number;
}

export interface LogQueryResult {
  entries: LogEntry[];
  stats: LogQueryStats;
}

export const logAPI = {
  list: (params?: PaginationParams & { workspace_id?: string; keyword?: string }) =>
    api.get<ApiResponse<PaginatedResponse<LogInstance>>>('/log-instances', { params }),

  get: (id: string) =>
    api.get<ApiResponse<LogInstance>>(`/log-instances/${id}`),

  create: (data: {
    tenant_id: string;
    zone_id: string;
    instance_name: string;
    backend_type?: string;
    namespace?: string;
    release_name?: string;
    retention_days?: number;
    spec?: string;
  }) => api.post<ApiResponse<LogInstance>>('/log-instances', data),

  update: (id: string, data: Partial<LogInstance>) =>
    api.put<ApiResponse<LogInstance>>(`/log-instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/log-instances/${id}`),

  query: (id: string, data: { query?: string; start?: string; end?: string; limit?: number }) =>
    api.post<ApiResponse<LogQueryResult>>(`/log-instances/${id}/query`, data),
};
