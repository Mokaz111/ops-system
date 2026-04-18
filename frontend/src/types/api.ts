export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface PaginationParams {
  page?: number;
  page_size?: number;
  search?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface User {
  id: string;
  username: string;
  display_name: string;
  email: string;
  phone: string;
  role: 'admin' | 'operator' | 'viewer';
  dept_id: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Department {
  id: string;
  dept_name: string;
  parent_id: string | null;
  leader_id: string | null;
  leader_name?: string;
  sort_order: number;
  status: string;
  children?: Department[];
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  tenant_name: string;
  dept_id: string;
  dept_name?: string;
  vmuser_id: string;
  vmuser_key: string;
  template_type: 'shared' | 'dedicated_single' | 'dedicated_cluster';
  quota_config: string;
  status: string;
  n9e_team_id: number;
  grafana_org_id: number;
  created_at: string;
  updated_at: string;
}

export interface Instance {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  cluster_id?: string | null;
  instance_name: string;
  instance_type: 'metrics' | 'logs' | 'visual' | 'alert';
  template_type: 'shared' | 'dedicated_single' | 'dedicated_cluster';
  release_name: string;
  namespace: string;
  spec: string;
  status: 'creating' | 'running' | 'stopped' | 'error' | 'scaling' | 'deleting';
  url: string;
  created_at: string;
  updated_at: string;
}

export interface InstanceSpec {
  cpu: number;
  memory: number;
  storage: number;
  retention: number;
  replicas?: number;
}

export interface CreateTenantRequest {
  tenant_name: string;
  dept_id: string;
  template_type: string;
  quota_config?: string;
}

export interface CreateInstanceRequest {
  tenant_id: string;
  cluster_id?: string;
  instance_name: string;
  instance_type: string;
  template_type: string;
  spec: string;
}

export interface ScaleInstanceRequest {
  scale_type: 'horizontal' | 'vertical' | 'storage';
  cpu?: string;
  memory?: string;
  storage?: string;
  replicas?: number;
}

export interface GrafanaOrg {
  id: number;
  name: string;
}

export interface GrafanaOrgUser {
  orgId: number;
  userId: number;
  login: string;
  role: string;
  email: string;
}

export interface GrafanaDatasource {
  id: number;
  orgId: number;
  name: string;
  type: string;
  url: string;
  access: string;
  isDefault: boolean;
}

export interface TenantMetrics {
  cpu_usage_percent: number;
  memory_usage_percent: number;
  series_count: number;
  ingest_qps: number;
  note?: string;
}

export interface InstanceMetrics {
  cpu_usage_percent: number;
  memory_usage_percent: number;
  disk_usage_percent: number;
  note?: string;
}

export type PlatformScaleScope = 'shared_metrics' | 'dedicated_metrics';

export interface PlatformScaleVMClusterRequest {
  target_id: string;
  dry_run?: boolean;
  vmselect_replicas?: number;
  vminsert_replicas?: number;
  vmstorage_replicas?: number;
  storage_size?: string;
}

export interface PlatformScaleVMClusterPlan {
  target_id: string;
  scope: PlatformScaleScope;
  namespace: string;
  name: string;
  dry_run: boolean;
  resource: string;
  spec_patch: Record<string, unknown>;
}

export interface PlatformScaleTarget {
  id: string;
  scope: PlatformScaleScope;
  namespace: string;
  name: string;
  display_name: string;
}

export interface PlatformScaleAuditItem {
  id: string;
  user_id: string;
  username: string;
  role: string;
  client_ip: string;
  target_id: string;
  dry_run: boolean;
  status: 'success' | 'failed' | 'replayed';
  spec_patch: string;
  error_message: string;
  created_at: string;
}

export interface PlatformInitSharedClusterRequest {
  dry_run?: boolean;
  namespace?: string;
  release_name?: string;
}

export interface PlatformInitSharedClusterPlan {
  dry_run: boolean;
  namespace: string;
  release_name: string;
  chart: string;
  action: string;
  values: Record<string, unknown>;
}
