import api from './index';
import type {
  ApiResponse,
  CreateTenantRequest,
  TenantMetrics,
  PaginatedResponse,
  PaginationParams,
  Tenant,
} from '../types/api';

export const tenantAPI = {
  list: (params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<Tenant>>>('/tenants', { params }),

  get: (id: string) =>
    api.get<ApiResponse<Tenant>>(`/tenants/${id}`),

  create: (data: CreateTenantRequest) =>
    api.post<ApiResponse<Tenant>>('/tenants', data),

  update: (id: string, data: Partial<CreateTenantRequest>) =>
    api.put<ApiResponse<Tenant>>(`/tenants/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/tenants/${id}`),

  metrics: (id: string) =>
    api.get<ApiResponse<TenantMetrics>>(`/tenants/${id}/metrics`),
};
