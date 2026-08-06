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

export interface UserMembership {
  workspace_id: string;
  workspace_name: string;
  role: 'admin' | 'member' | 'viewer' | string;
}

export interface User {
  id: string;
  username: string;
  display_name?: string;
  email?: string;
  phone?: string;
  role: 'admin' | 'user' | string;
  memberships: UserMembership[];
  status: string;
  created_at: string;
  updated_at: string;
}

export interface APIToken {
  id: string;
  user_id: string;
  name: string;
  token_prefix: string;
  scope: 'read' | 'read_write' | string;
  expires_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateAPITokenRequest {
  name: string;
  scope?: 'read' | 'read_write';
  expires_at?: string | null;
}

export interface CreateAPITokenResponse extends APIToken {
  token: string;
}

export interface WorkspaceMember {
  id: string;
  workspace_id: string;
  user_id: string;
  username?: string;
  display_name?: string;
  role: 'admin' | 'member' | 'viewer' | string;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: string;
  tenant_id?: string | null;
  actor_id?: string | null;
  actor_type: string;
  action: string;
  resource: string;
  resource_id: string;
  details: string;
  ip: string;
  user_agent: string;
  status: string;
  created_at: string;
}

export interface PreflightCheck {
  ok: boolean;
  issues: Array<{
    component: string;
    reason: string;
    message?: string;
  }>;
}

export interface ZoneComponent {
  name: string;
  component: string;
  status: string;
  version?: string;
  message?: string;
  ready?: boolean;
}

export interface Workspace {
  id: string;
  workspace_name: string;
  slug?: string;
  vmuser_id: string;
  vmuser_key: string;
  template_type: 'shared';
  quota_config: string;
  isolation_level?: 'shared' | string;
  vm_namespace?: string;
  vm_select_url?: string;
  vm_insert_url?: string;
  insert_url?: string;
  status: 'creating' | 'active' | 'degraded' | 'suspended' | 'deleting' | 'failed' | string;
  grafana_org_id: number;
  grafana_instance_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Instance {
  id: string;
  workspace_id: string;
  workspace_name?: string;
  cluster_id?: string | null;
  zone_id?: string | null;
  instance_name: string;
  instance_type: 'metrics' | 'logs' | 'alert';
  template_type: 'shared';
  release_name: string;
  namespace: string;
  spec: string;
  status: 'creating' | 'running' | 'stopped' | 'error' | 'failed' | 'deleting';
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

export interface CreateWorkspaceRequest {
  workspace_name: string;
  template_type: string;
  quota_config?: string;
  grafana_instance_id?: string;
}

export interface CreateInstanceRequest {
  workspace_id?: string;
  cluster_id?: string;
  zone_id?: string;
  instance_name: string;
  instance_type: string;
  template_type: string;
  spec: string;
  grafana_instance_id?: string;
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

export interface WorkspaceMetrics {
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
  workspace_id: string;
  rule_name: string;
  rule_type: 'metrics' | 'logs' | string;
  query: string;
  condition: string;
  level: 'critical' | 'warning' | 'info' | string;
  channels: string;
  annotations: string;
  enabled: boolean;
  vm_rule_name?: string;
  vm_namespace?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertRuleRequest {
  workspace_id: string;
  rule_name: string;
  rule_type: string;
  query: string;
  condition?: string;
  level: string;
  channels?: string;
  annotations?: string;
  enabled: boolean;
}

export interface NotificationChannel {
  id: string;
  workspace_id?: string;
  tenant_id?: string;
  channel_name: string;
  channel_type: 'dingtalk' | 'email' | 'slack' | 'sms' | 'webhook' | string;
  config: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateNotificationChannelRequest {
  workspace_id: string;
  channel_name: string;
  channel_type: string;
  config?: string;
  enabled?: boolean;
}

export interface AlertEvent {
  id: string;
  workspace_id: string;
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

export type PlatformScaleScope = 'shared_metrics';

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
