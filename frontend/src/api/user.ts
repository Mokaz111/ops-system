import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams, User } from '../types/api';

export const userAPI = {
  list: (params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<User>>>('/users', { params: normalizeListParams(params) }),

  get: (id: string) =>
    api.get<ApiResponse<User>>(`/users/${id}`),

  create: (data: Partial<User> & { password?: string }) =>
    api.post<ApiResponse<User>>('/users', data),

  update: (id: string, data: Partial<User>) =>
    api.put<ApiResponse<User>>(`/users/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/users/${id}`),
};

function normalizeListParams(params?: PaginationParams) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
