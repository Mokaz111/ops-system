import api from './index';
import type {
  AlertEvent,
  AlertLevelStat,
  AlertRule,
  AlertRuleStat,
  AlertSummary,
  AlertTrendPoint,
  ApiResponse,
  CreateAlertRuleRequest,
  CreateNotificationChannelRequest,
  ImportRulesResult,
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

  // 批量导入 Prometheus 风格规则文件（.yaml/.yml/.zip/.tar.gz）。
  importRules: (tenantId: string, file: File) => {
    const form = new FormData();
    form.append('tenant_id', tenantId);
    form.append('file', file);
    return api.post<ApiResponse<ImportRulesResult>>('/alerts/rules/import', form);
  },

  listEvents: (params?: PaginationParams & {
    workspace_id?: string;
    rule_id?: string;
    level?: string;
    status?: string;
    start_time?: string;
    end_time?: string;
  }) =>
    api.get<ApiResponse<PaginatedResponse<AlertEvent>>>('/alerts/events', { params }),

  getEvent: (id: string) =>
    api.get<ApiResponse<AlertEvent>>(`/alerts/events/${id}`),

  ackEvent: (id: string) =>
    api.put<ApiResponse<AlertEvent>>(`/alerts/events/${id}/ack`),

  listChannels: (params?: PaginationParams & { workspace_id?: string; channel_type?: string }) =>
    api.get<ApiResponse<PaginatedResponse<NotificationChannel>>>('/alerts/channels', { params: normalizeListParams(params) }),

  createChannel: (data: CreateNotificationChannelRequest) =>
    api.post<ApiResponse<NotificationChannel>>('/alerts/channels', data),

  updateChannel: (id: string, data: Partial<Omit<CreateNotificationChannelRequest, 'tenant_id'>>) =>
    api.put<ApiResponse<NotificationChannel>>(`/alerts/channels/${id}`, data),

  deleteChannel: (id: string) =>
    api.delete<ApiResponse<null>>(`/alerts/channels/${id}`),

  // ── 统计（admin 必须传 workspace_id；普通用户单空间时可省略）──

  statsSummary: (params: { workspace_id?: string }) =>
    api.get<ApiResponse<AlertSummary>>('/alerts/stats/summary', { params }),

  statsTrend: (params: { workspace_id?: string; start: string; end: string; interval?: 'hour' | 'day' }) =>
    api.get<ApiResponse<AlertTrendPoint[]>>('/alerts/stats/trend', { params }),

  statsByLevel: (params: { workspace_id?: string; start: string; end: string }) =>
    api.get<ApiResponse<AlertLevelStat[]>>('/alerts/stats/by-level', { params }),

  statsByRule: (params: { workspace_id?: string; start: string; end: string; limit?: number }) =>
    api.get<ApiResponse<AlertRuleStat[]>>('/alerts/stats/by-rule', { params }),
};

function normalizeListParams<T extends PaginationParams>(params?: T) {
  if (!params?.search) return params;
  const { search, ...rest } = params;
  return { ...rest, keyword: search };
}
