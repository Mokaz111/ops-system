import type { AxiosRequestConfig } from 'axios';
import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams } from '../types/api';

export interface BusinessCluster {
  id: string;
  tenant_id: string;
  instance_id: string;
  name: string;
  display_name: string;
  kubeconfig_path: string;
  agent_status: string;
  log_agent_status: string;
  log_instance_id?: string;
  labels: string;
  metrics_collect_config?: string;
  logs_collect_config?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateBusinessClusterRequest {
  instance_id: string;
  name: string;
  display_name?: string;
  kubeconfig?: string;
  kubeconfig_path?: string;
  labels?: Record<string, string>;
  metrics_collect_config?: MetricsCollectConfig;
  logs_collect_config?: LogsCollectConfig;
}

export interface MetricsCollectConfig {
  select_all_by_default?: boolean;
  scrape_interval?: string;
  scrape_timeout?: string;
  namespace_include?: string[];
  namespace_exclude?: string[];
}

export interface LogsCollectConfig {
  namespace_include?: string[];
  namespace_exclude?: string[];
  exclude_paths?: string[];
}

export interface CollectConfigView {
  metrics: MetricsCollectConfig;
  logs: LogsCollectConfig;
}

export const businessClusterAPI = {
  list: (
    params?: PaginationParams & { workspace_id?: string; instance_id?: string },
    config?: AxiosRequestConfig,
  ) =>
    api.get<ApiResponse<PaginatedResponse<BusinessCluster>>>('/business-clusters', { ...config, params }),

  get: (id: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<BusinessCluster>>(`/business-clusters/${id}`, config),

  create: (data: CreateBusinessClusterRequest) =>
    api.post<ApiResponse<BusinessCluster>>('/business-clusters', data),

  delete: (id: string, force?: boolean) =>
    api.delete<ApiResponse<null>>(`/business-clusters/${id}`, { params: { force: force ? 'true' : undefined } }),

  enableLogs: (id: string, data: { log_instance_id: string }) =>
    api.post<ApiResponse<BusinessCluster>>(`/business-clusters/${id}/enable-logs`, data),

  disableLogs: (id: string, force?: boolean) =>
    api.post<ApiResponse<BusinessCluster>>(`/business-clusters/${id}/disable-logs`, undefined, {
      params: { force: force ? 'true' : undefined },
    }),

  getCollectConfig: (id: string) =>
    api.get<ApiResponse<CollectConfigView>>(`/business-clusters/${id}/collect-config`),

  updateCollectConfig: (id: string, data: { metrics?: MetricsCollectConfig; logs?: LogsCollectConfig }) =>
    api.put<ApiResponse<CollectConfigView>>(`/business-clusters/${id}/collect-config`, data),
};
