export type AppRouteKey =
  | 'dashboard'
  | 'departments'
  | 'tenants'
  | 'instances'
  | 'instance-detail'
  | 'integrations'
  | 'metrics'
  | 'log-instances'
  | 'log-query'
  | 'grafana'
  | 'grafana-hosts'
  | 'dashboard-mgmt'
  | 'alerts'
  | 'users'
  | 'clusters'
  | 'platform-scaling'
  | 'settings';

type SidebarSection = 'overview' | 'resource' | 'monitor' | 'logs' | 'visualization' | 'system';

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

  { key: 'instances', path: 'instances', label: '监控实例', showInSidebar: true, sidebarSection: 'monitor' },
  { key: 'instance-detail', path: 'instances/:instanceId' },
  { key: 'integrations', path: 'integrations', label: '接入中心', showInSidebar: true, sidebarSection: 'monitor' },
  { key: 'metrics', path: 'metrics', label: '指标库', showInSidebar: true, sidebarSection: 'monitor' },

  { key: 'log-instances', path: 'log-instances', label: '日志实例', showInSidebar: true, sidebarSection: 'logs' },
  { key: 'log-query', path: 'logs/query', label: '日志查询', showInSidebar: true, sidebarSection: 'logs' },

  { key: 'grafana', path: 'grafana', label: 'Grafana 管理', showInSidebar: true, sidebarSection: 'visualization' },
  { key: 'grafana-hosts', path: 'grafana-hosts', label: 'Grafana 主机', showInSidebar: true, sidebarSection: 'visualization' },
  { key: 'dashboard-mgmt', path: 'dashboards', label: 'Dashboard 管理', showInSidebar: true, sidebarSection: 'visualization' },

  // 告警引擎保留路由（供 InstanceDetail 内链跳转）但不在侧边栏显示。
  { key: 'alerts', path: 'alerts' },

  { key: 'users', path: 'users', label: '用户管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'clusters', path: 'clusters', label: '集群管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'platform-scaling', path: 'platform-scaling', label: '平台扩容', showInSidebar: true, sidebarSection: 'system' },
  { key: 'settings', path: 'settings', label: '系统设置', showInSidebar: true, sidebarSection: 'system' },
];
