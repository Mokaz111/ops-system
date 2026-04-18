import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface Cluster {
  id: string;
  name: string;
  display_name: string;
  description: string;
  in_cluster: boolean;
  kubeconfig_path: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateClusterRequest {
  name: string;
  display_name?: string;
  description?: string;
  in_cluster?: boolean;
  kubeconfig?: string;
  kubeconfig_path?: string;
}

export interface UpdateClusterRequest {
  display_name?: string;
  description?: string;
  in_cluster?: boolean;
  kubeconfig?: string;
  kubeconfig_path?: string;
  status?: string;
}

export const clusterAPI = {
  list: (params?: PaginationParams & { status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<Cluster>>>('/clusters', { params }),

  get: (id: string) =>
    api.get<ApiResponse<Cluster>>(`/clusters/${id}`),

  create: (data: CreateClusterRequest) =>
    api.post<ApiResponse<Cluster>>('/clusters', data),

  update: (id: string, data: UpdateClusterRequest) =>
    api.put<ApiResponse<Cluster>>(`/clusters/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/clusters/${id}`),
};
