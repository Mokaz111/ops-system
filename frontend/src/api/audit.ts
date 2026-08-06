import api from './index';
import type { ApiResponse, AuditLog, PaginatedResponse, PaginationParams } from '../types/api';

export const auditAPI = {
  list: (params?: PaginationParams & {
    action?: string;
    resource?: string;
    status?: string;
    tenant_id?: string;
    start_time?: string;
    end_time?: string;
  }) =>
    api.get<ApiResponse<PaginatedResponse<AuditLog>>>('/audits', { params: normalizeListParams(params) }),
};

function normalizeListParams<T extends PaginationParams>(params?: T) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
