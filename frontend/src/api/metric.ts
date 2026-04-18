import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface Metric {
  id: string;
  name: string;
  metric_type: string;
  unit: string;
  component: string;
  description_cn: string;
  description_en: string;
  labels: string;
  examples: string;
  source_template_id: string | null;
  source_template_version: string;
  manual_override: boolean;
  tags: string;
  created_at: string;
  updated_at: string;
}

export interface MetricTemplateMapping {
  id: string;
  metric_id: string;
  template_id: string;
  template_version: string;
  appears_in_collector: boolean;
  appears_in_dashboard: boolean;
  appears_in_alert: boolean;
  dashboard_panels: string;
  created_at: string;
}

export const metricAPI = {
  list: (params?: PaginationParams & { component?: string; template_id?: string; keyword?: string }) =>
    api.get<ApiResponse<PaginatedResponse<Metric>>>('/metrics', { params }),

  get: (id: string) =>
    api.get<ApiResponse<Metric>>(`/metrics/${id}`),

  create: (data: Partial<Metric> & { name: string }) =>
    api.post<ApiResponse<Metric>>('/metrics', data),

  update: (id: string, data: Partial<Metric>) =>
    api.put<ApiResponse<Metric>>(`/metrics/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/metrics/${id}`),

  related: (id: string) =>
    api.get<ApiResponse<MetricTemplateMapping[]>>(`/metrics/${id}/related`),

  reparse: (templateId: string) =>
    api.post<ApiResponse<unknown>>(`/metrics/reparse/${templateId}`),
};
