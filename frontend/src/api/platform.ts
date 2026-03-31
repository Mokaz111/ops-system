import api from './index';
import type {
  ApiResponse,
  PaginatedResponse,
  PlatformScaleAuditItem,
  PlatformScaleTarget,
  PlatformScaleVMClusterPlan,
  PlatformScaleVMClusterRequest,
} from '../types/api';

export const platformAPI = {
  listVMClusterTargets: () =>
    api.get<ApiResponse<PlatformScaleTarget[]>>('/platform/scaling/vmcluster/targets'),

  listAudits: (params?: {
    page?: number;
    page_size?: number;
    target_id?: string;
    status?: 'success' | 'failed' | 'replayed' | '';
    operator?: string;
    start_time?: string;
    end_time?: string;
  }) =>
    api.get<ApiResponse<PaginatedResponse<PlatformScaleAuditItem>>>('/platform/scaling/audits', { params }),

  scaleVMCluster: (data: PlatformScaleVMClusterRequest, opts?: { idempotencyKey?: string }) =>
    api.post<ApiResponse<PlatformScaleVMClusterPlan>>('/platform/scaling/vmcluster', data, {
      headers: opts?.idempotencyKey ? { 'Idempotency-Key': opts.idempotencyKey } : undefined,
    }),
};
