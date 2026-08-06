import api from './index';
import type { AxiosRequestConfig } from 'axios';
import type {
  ApiResponse,
  CreateInstanceRequest,
  InstanceMetrics,
  Instance,
  PaginatedResponse,
  PaginationParams,
} from '../types/api';

export const instanceAPI = {
  list: (
    params?: PaginationParams & { workspace_id?: string; instance_type?: string; status?: string },
    config?: AxiosRequestConfig,
  ) =>
    api.get<ApiResponse<PaginatedResponse<Instance>>>('/instances', { ...config, params: normalizeListParams(params) }),

  get: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<Instance>>(`/instances/${id}`, config),

  create: (data: CreateInstanceRequest) =>
    api.post<ApiResponse<Instance>>('/instances', data),

  update: (id: string, data: Partial<CreateInstanceRequest>) =>
    api.put<ApiResponse<Instance>>(`/instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/instances/${id}`),

  metrics: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<InstanceMetrics>>(`/instances/${id}/metrics`, config),

  login: (id: string, redirect?: string) =>
    api.post<ApiResponse<{ proxyUrl: string }>>(`/instances/${id}/login`, null, {
      params: redirect ? { redirect } : undefined,
    }),
};

function normalizeListParams<T extends PaginationParams>(params?: T) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
