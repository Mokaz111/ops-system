import api from './index';
import type { ApiResponse, PaginatedResponse, PaginationParams, PreflightCheck, ZoneComponent } from '../types/api';

export interface Zone {
  id: string;
  slug: string;
  display_name: string;
  description: string;
  cluster_id: string;
  endpoint: string;
  labels: string;
  capacity: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ZoneStats {
  zone_id: string;
  total_instances: number;
}

export interface CreateZoneRequest {
  slug: string;
  display_name: string;
  description?: string;
  cluster_id: string;
  endpoint?: string;
  labels?: Record<string, string>;
  capacity?: { max_instances: number; max_storage?: string };
}

export interface UpdateZoneRequest {
  display_name?: string;
  description?: string;
  endpoint?: string;
  labels?: Record<string, string>;
  capacity?: { max_instances: number; max_storage?: string };
  status?: string;
}

export const zoneAPI = {
  list: (params?: PaginationParams & { status?: string }) =>
    api.get<ApiResponse<PaginatedResponse<Zone>>>('/zones', { params }),

  get: (id: string) =>
    api.get<ApiResponse<{ zone: Zone; stats: ZoneStats }>>(`/zones/${id}`),

  create: (data: CreateZoneRequest) =>
    api.post<ApiResponse<Zone>>('/zones', data),

  update: (id: string, data: UpdateZoneRequest) =>
    api.put<ApiResponse<Zone>>(`/zones/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/zones/${id}`),

  initShared: (id: string, body?: { dry_run?: boolean; namespace?: string; release_name?: string; values?: Record<string, unknown> }) =>
    api.post<ApiResponse<Record<string, unknown>>>(`/zones/${id}/init-shared`, body || {}),

  initLogs: (id: string, body?: { dry_run?: boolean; namespace?: string; release_name?: string; values?: Record<string, unknown> }) =>
    api.post<ApiResponse<Record<string, unknown>>>(`/zones/${id}/init-logs`, body || {}),

  initGrafana: (id: string) =>
    api.post<ApiResponse<Record<string, unknown>>>(`/zones/${id}/init-grafana`),

  preflight: (id: string) =>
    api.get<ApiResponse<{ checks: PreflightCheck[] }>>(`/zones/${id}/preflight`),

  getComponents: (id: string) =>
    api.get<ApiResponse<{ components: ZoneComponent[] }>>(`/zones/${id}/components`),
};
