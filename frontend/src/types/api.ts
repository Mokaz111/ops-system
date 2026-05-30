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
  keyword?: string;
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
  display_name?: string;
  email?: string;
  phone?: string;
  role: 'admin' | 'user' | 'platform_admin' | 'tenant_admin' | 'editor' | 'viewer' | 'alert_admin' | string;
  dept_id?: string | null;
  tenant_id?: string | null;
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
  slug?: string;
  vmuser_id: string;
  vmuser_key: string;
  template_type: 'shared' | 'dedicated_single' | 'dedicated_cluster';
  quota_config: string;
  isolation_level?: 'shared' | 'namespace' | 'dedicated' | string;
  vm_namespace?: string;
  vm_select_url?: string;
  vm_insert_url?: string;
  insert_url?: string;
  status: 'creating' | 'active' | 'degraded' | 'suspended' | 'deleting' | 'failed' | string;
  n9e_team_id: number;
  grafana_org_id: number;
  grafana_instance_id?: string | null;
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
  status: 'creating' | 'running' | 'stopped' | 'error' | 'failed' | 'scaling' | 'deleting';
  grafana_instance_id?: string | null;
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
  grafana_instance_id?: string;
}

export interface CreateInstanceRequest {
  tenant_id?: string;
  cluster_id?: string;
  instance_name: string;
  instance_type: string;
  template_type: string;
  spec: string;
  grafana_instance_id?: string;
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

export interface GrafanaDashboard {
  id: number;
  uid: string;
  title: string;
  url: string;
  type: string;
  tags: string[];
  folder_id: number;
  folder_title: string;
}

export interface GrafanaPlugin {
  id: string;
  name: string;
  type: string;
  version: string;
  enabled: boolean;
  pinned: boolean;
}

export interface GrafanaHealthStatus {
  status: string;
  message?: string;
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

export interface AlertRule {
  id: string;
  tenant_id: string;
  rule_name: string;
  rule_type: 'metrics' | 'logs' | string;
  query: string;
  condition: string;
  level: 'critical' | 'warning' | 'info' | string;
  channels: string;
  annotations: string;
  enabled: boolean;
  n9e_rule_id?: number;
  vm_rule_name?: string;
  vm_namespace?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertRuleRequest {
  tenant_id: string;
  rule_name: string;
  rule_type: string;
  query: string;
  condition?: string;
  level: string;
  channels?: string;
  annotations?: string;
  enabled: boolean;
}

export interface AlertEvent {
  id: string;
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  level: string;
  status: string;
  start_time: string;
  end_time?: string | null;
  details: string;
  notified: boolean;
  acked_by?: string | null;
  acked_at?: string | null;
  created_at: string;
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
