import api from './index';
import type {
  AlertEvent,
  AlertRule,
  ApiResponse,
  CreateAlertRuleRequest,
  CreateNotificationChannelRequest,
  NotificationChannel,
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

  listChannels: (params?: PaginationParams & { workspace_id?: string; channel_type?: string }) =>
    api.get<ApiResponse<PaginatedResponse<NotificationChannel>>>('/alerts/channels', { params: normalizeListParams(params) }),

  createChannel: (data: CreateNotificationChannelRequest) =>
    api.post<ApiResponse<NotificationChannel>>('/alerts/channels', {
      tenant_id: data.workspace_id,
      channel_name: data.channel_name,
      channel_type: data.channel_type,
      config: data.config,
      enabled: data.enabled,
    }),

  updateChannel: (id: string, data: Partial<CreateNotificationChannelRequest>) =>
    api.put<ApiResponse<NotificationChannel>>(`/alerts/channels/${id}`, data),

  deleteChannel: (id: string) =>
    api.delete<ApiResponse<null>>(`/alerts/channels/${id}`),
};

function normalizeListParams<T extends PaginationParams>(params?: T) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
