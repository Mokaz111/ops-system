import type { AxiosRequestConfig } from 'axios';
import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface Entity {
  id: string;
  tenant_id: string;
  entity_type: string;
  name: string;
  display_name: string;
  labels: string;
  status: string;
  created_at: string;
}

export interface MetricSet {
  id: string;
  tenant_id: string;
  name: string;
  display_name: string;
  component: string;
  description: string;
  status: string;
  created_at: string;
}

export interface DataLink {
  id: string;
  entity_id: string;
  target_type: string;
  target_id: string;
  relation_type: string;
}

export const umodelAPI = {
  listEntities: (params?: PaginationParams & { entity_type?: string; keyword?: string; workspace_id?: string }, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<PaginatedResponse<Entity>>>('/umodel/entities', { ...config, params }),

  createEntity: (data: { entity_type: string; name: string; display_name?: string; labels?: Record<string, string> }, workspaceId?: string) =>
    api.post<ApiResponse<Entity>>('/umodel/entities', data, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),

  deleteEntity: (id: string, workspaceId?: string) =>
    api.delete<ApiResponse<{ deleted: boolean }>>(`/umodel/entities/${id}`, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),

  listMetricSets: (params?: PaginationParams & { component?: string; keyword?: string; workspace_id?: string }, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<PaginatedResponse<MetricSet>>>('/umodel/metric-sets', { ...config, params }),

  createMetricSet: (data: { name: string; display_name?: string; component?: string; description?: string }, workspaceId?: string) =>
    api.post<ApiResponse<MetricSet>>('/umodel/metric-sets', data, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),

  deleteMetricSet: (id: string, workspaceId?: string) =>
    api.delete<ApiResponse<{ deleted: boolean }>>(`/umodel/metric-sets/${id}`, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),

  listDataLinks: (entityId: string, workspaceId?: string) =>
    api.get<ApiResponse<{ items: DataLink[] }>>(`/umodel/entities/${entityId}/data-links`, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),

  createDataLink: (data: { entity_id: string; target_type: string; target_id: string; relation_type?: string }, workspaceId?: string) =>
    api.post<ApiResponse<DataLink>>('/umodel/data-links', data, { params: workspaceId ? { workspace_id: workspaceId } : undefined }),
};
