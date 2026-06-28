import type { AxiosRequestConfig } from 'axios';
import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface BusinessCluster {
  id: string;
  workspace_id: string;
  instance_id: string;
  name: string;
  display_name: string;
  kubeconfig_path: string;
  agent_status: string;
  labels: string;
  created_at: string;
  updated_at: string;
}

export interface CreateBusinessClusterRequest {
  instance_id: string;
  name: string;
  display_name?: string;
  kubeconfig?: string;
  kubeconfig_path?: string;
  labels?: Record<string, string>;
}

export const businessClusterAPI = {
  list: (
    params?: PaginationParams & { workspace_id?: string; instance_id?: string },
    config?: AxiosRequestConfig,
  ) =>
    api.get<ApiResponse<PaginatedResponse<BusinessCluster>>>('/business-clusters', { ...config, params }),

  get: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<BusinessCluster>>(`/business-clusters/${id}`, config),

  create: (data: CreateBusinessClusterRequest) =>
    api.post<ApiResponse<BusinessCluster>>('/business-clusters', data),

  delete: (id: string, force?: boolean) =>
    api.delete<ApiResponse<null>>(`/business-clusters/${id}`, { params: { force: force ? 'true' : undefined } }),
};
