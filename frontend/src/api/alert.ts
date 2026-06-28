import api from './index';
import type {
  AlertEvent,
  AlertRule,
  ApiResponse,
  CreateAlertRuleRequest,
  PaginatedResponse,
  PaginationParams,
} from '../types/api';

export const alertAPI = {
  listRules: (params?: PaginationParams & { workspace_id?: string; rule_type?: string; level?: string }) =>
    api.get<ApiResponse<PaginatedResponse<AlertRule>>>('/alerts/rules', { params: normalizeListParams(params) }),

  createRule: (data: CreateAlertRuleRequest) =>
    api.post<ApiResponse<AlertRule>>('/alerts/rules', data),

  updateRule: (id: string, data: Partial<CreateAlertRuleRequest>) =>
    api.put<ApiResponse<AlertRule>>(`/alerts/rules/${id}`, data),

  deleteRule: (id: string) =>
    api.delete<ApiResponse<null>>(`/alerts/rules/${id}`),

  listEvents: (params?: PaginationParams & { workspace_id?: string; rule_id?: string; level?: string; status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<AlertEvent>>>('/alerts/events', { params }),

  ackEvent: (id: string) =>
    api.put<ApiResponse<AlertEvent>>(`/alerts/events/${id}/ack`),
};

function normalizeListParams<T extends PaginationParams>(params?: T) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
