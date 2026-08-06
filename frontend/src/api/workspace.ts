import api from './index';
import type {
  ApiResponse,
  CreateWorkspaceRequest,
  WorkspaceMetrics,
  PaginatedResponse,
  PaginationParams,
  Workspace,
  WorkspaceMember,
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

  listMembers: (workspaceId: string, params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<WorkspaceMember>>>(`/workspaces/${workspaceId}/members`, { params: normalizeListParams(params) }),

  addMember: (workspaceId: string, data: { user_id: string; role: string }) =>
    api.post<ApiResponse<WorkspaceMember>>(`/workspaces/${workspaceId}/members`, data),

  // 注意：路径参数是成员的 user_id（后端路由为 /members/:userId），不是 membership 记录主键。
  updateMember: (workspaceId: string, userId: string, data: { role: string }) =>
    api.put<ApiResponse<WorkspaceMember>>(`/workspaces/${workspaceId}/members/${userId}`, data),

  removeMember: (workspaceId: string, userId: string) =>
    api.delete<ApiResponse<null>>(`/workspaces/${workspaceId}/members/${userId}`),
};

function normalizeListParams(params?: PaginationParams) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
