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
  instance_name: string;
  instance_type: string;
  template_type: string;
  spec: string;
}

export interface ScaleInstanceRequest {
  cpu?: number;
  memory?: number;
  storage?: number;
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

export interface MetricsData {
  cpu_usage: number;
  memory_usage: number;
  storage_usage: number;
  ingestion_rate: number;
  series_count: number;
}
