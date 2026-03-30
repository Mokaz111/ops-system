import api from './index';
import type { ApiResponse, Department, PaginatedResponse, PaginationParams, User } from '../types/api';

export const departmentAPI = {
  list: (params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<Department>>>('/departments', { params }),

  tree: () =>
    api.get<ApiResponse<Department[]>>('/departments/tree'),

  get: (id: string) =>
    api.get<ApiResponse<Department>>(`/departments/${id}`),

  create: (data: Partial<Department>) =>
    api.post<ApiResponse<Department>>('/departments', data),

  update: (id: string, data: Partial<Department>) =>
    api.put<ApiResponse<Department>>(`/departments/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/departments/${id}`),

  listUsers: (id: string) =>
    api.get<ApiResponse<User[]>>(`/departments/${id}/users`),
};
