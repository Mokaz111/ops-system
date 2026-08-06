import api from './index';
import type {
  APIToken,
  ApiResponse,
  CreateAPITokenRequest,
  CreateAPITokenResponse,
  PaginatedResponse,
  PaginationParams,
} from '../types/api';

export const apiTokenAPI = {
  list: (params?: PaginationParams) =>
    api.get<ApiResponse<PaginatedResponse<APIToken>>>('/api-tokens', { params }),

  create: (data: CreateAPITokenRequest) =>
    api.post<ApiResponse<CreateAPITokenResponse>>('/api-tokens', data),

  revoke: (id: string) =>
    api.delete<ApiResponse<null>>(`/api-tokens/${id}`),
};
