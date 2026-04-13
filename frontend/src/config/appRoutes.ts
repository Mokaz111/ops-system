export type AppRouteKey =
  | 'dashboard'
  | 'departments'
  | 'tenants'
  | 'instances'
  | 'instance-detail'
  | 'grafana'
  | 'alerts'
  | 'users'
  | 'platform-scaling'
  | 'settings';

type SidebarSection = 'overview' | 'resource' | 'observability' | 'system';

export interface AppRouteMeta {
  key: AppRouteKey;
  path: string;
  label?: string;
  showInSidebar?: boolean;
  sidebarSection?: SidebarSection;
}

export const appRouteMeta: AppRouteMeta[] = [
  { key: 'dashboard', path: 'dashboard', label: '概览', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'departments', path: 'departments', label: '部门管理', showInSidebar: true, sidebarSection: 'resource' },
  { key: 'tenants', path: 'tenants', label: '租户管理', showInSidebar: true, sidebarSection: 'resource' },
  { key: 'instances', path: 'instances', label: '实例管理', showInSidebar: true, sidebarSection: 'resource' },
  { key: 'instance-detail', path: 'instances/:instanceId' },
  { key: 'grafana', path: 'grafana', label: 'Grafana 管理', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'alerts', path: 'alerts', label: '告警引擎', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'users', path: 'users', label: '用户管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'platform-scaling', path: 'platform-scaling', label: '平台扩容', showInSidebar: true, sidebarSection: 'system' },
  { key: 'settings', path: 'settings', label: '系统设置', showInSidebar: true, sidebarSection: 'system' },
];
