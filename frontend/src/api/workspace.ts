import api from './index';
import type {
  ApiResponse,
  CreateWorkspaceRequest,
  WorkspaceMetrics,
  PaginatedResponse,
  PaginationParams,
  Workspace,
} from '../types/api';

export const workspaceAPI = {
  list: (params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<Workspace>>>('/workspaces', { params: normalizeListParams(params) }),

  get: (id: string) =>
    api.get<ApiResponse<Workspace>>(`/workspaces/${id}`),

  create: (data: CreateWorkspaceRequest) =>
    api.post<ApiResponse<Workspace>>('/workspaces', data),

  update: (id: string, data: Partial<CreateWorkspaceRequest>) =>
    api.put<ApiResponse<Workspace>>(`/workspaces/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/workspaces/${id}`),

  metrics: (id: string) =>
    api.get<ApiResponse<WorkspaceMetrics>>(`/workspaces/${id}/metrics`),
};

function normalizeListParams(params?: PaginationParams) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
