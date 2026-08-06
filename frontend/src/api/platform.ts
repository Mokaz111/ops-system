import api from './index';
import type {
  ApiResponse,
  AuditLog,
  PaginatedResponse,
  PlatformInitSharedClusterPlan,
  PlatformInitSharedClusterRequest,
  PlatformScaleTarget,
  PlatformScaleVMClusterPlan,
  PlatformScaleVMClusterRequest,
} from '../types/api';

export const platformAPI = {
  listVMClusterTargets: () =>
    api.get<ApiResponse<PlatformScaleTarget[]>>('/platform/scaling/vmcluster/targets'),

  // 后端已并入统一审计（action=platform.scale），返回 AuditLog。
  listAudits: (params?: { page?: number; page_size?: number }) =>
    api.get<ApiResponse<PaginatedResponse<AuditLog>>>('/platform/scaling/audits', { params }),

  initSharedCluster: (data: PlatformInitSharedClusterRequest) =>
    api.post<ApiResponse<PlatformInitSharedClusterPlan>>('/platform/scaling/bootstrap/shared/init', data),

  scaleVMCluster: (data: PlatformScaleVMClusterRequest, opts?: { idempotencyKey?: string }) =>
    api.post<ApiResponse<PlatformScaleVMClusterPlan>>('/platform/scaling/vmcluster', data, {
      headers: opts?.idempotencyKey ? { 'Idempotency-Key': opts.idempotencyKey } : undefined,
    }),
};
