import api from './index';
import type {
  ApiResponse,
  CreateInstanceRequest,
  InstanceMetrics,
  Instance,
  PaginatedResponse,
  PaginationParams,
  ScaleInstanceRequest,
} from '../types/api';

export const instanceAPI = {
  list: (params?: PaginationParams & { tenant_id?: string; instance_type?: string; status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<Instance>>>('/instances', { params }),

  get: (id: string) =>
    api.get<ApiResponse<Instance>>(`/instances/${id}`),

  create: (data: CreateInstanceRequest) =>
    api.post<ApiResponse<Instance>>('/instances', data),

  update: (id: string, data: Partial<CreateInstanceRequest>) =>
    api.put<ApiResponse<Instance>>(`/instances/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/instances/${id}`),

  scale: (id: string, data: ScaleInstanceRequest) =>
    api.post<ApiResponse<Instance>>(`/instances/${id}/scale`, data),

  metrics: (id: string) =>
    api.get<ApiResponse<InstanceMetrics>>(`/instances/${id}/metrics`),
};
