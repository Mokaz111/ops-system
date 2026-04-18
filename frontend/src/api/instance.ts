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

  scaleEvents: (id: string, params?: PaginationParams & { scale_type?: string; status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<ScaleEvent>>>(`/instances/${id}/scale-events`, { params }),
};

export interface ScaleEvent {
  id: string;
  instance_id: string;
  instance_name: string;
  tenant_id: string;
  scale_type: 'horizontal' | 'vertical' | 'storage' | string;
  method: 'cr_patch' | 'helm_upgrade' | 'k8s_native' | 'rejected' | string;
  replicas?: number | null;
  cpu?: string;
  memory?: string;
  storage?: string;
  status: 'success' | 'failed' | string;
  error_message?: string;
  operator?: string;
  created_at: string;
}
