export type AppRouteKey =
  | 'dashboard'
  | 'departments'
  | 'tenants'
  | 'instances'
  | 'instance-detail'
  | 'grafana-instance-detail'
  | 'integrations'
  | 'metrics'
  | 'log-instances'
  | 'log-query'
  | 'grafana'
  | 'grafana-instances'
  | 'grafana-hosts'
  | 'dashboard-mgmt'
  | 'alerts'
  | 'users'
  | 'zones'
  | 'clusters'
  | 'business-clusters'
  | 'stats'
  | 'settings';

export type SidebarSection = 'overview' | 'observability' | 'admin';

export interface AppRouteMeta {
  key: AppRouteKey;
  path: string;
  label?: string;
  showInSidebar?: boolean;
  sidebarSection?: SidebarSection;
  requireAdmin?: boolean;
}

export const sectionLabels: Record<SidebarSection, string> = {
  overview: '',
  observability: '可观测性',
  admin: '管理',
};

export const appRouteMeta: AppRouteMeta[] = [
  // ── 概览 ──
  { key: 'dashboard', path: 'dashboard', label: '概览', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'dashboard-mgmt', path: 'dashboards', label: 'Dashboard', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'integrations', path: 'integrations', label: '接入中心', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'metrics', path: 'metrics', label: '指标库', showInSidebar: true, sidebarSection: 'overview' },

  // ── 可观测性 ──
  { key: 'instances', path: 'instances', label: '实例', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'log-instances', path: 'log-instances', label: '日志实例', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'business-clusters', path: 'business-clusters', label: '业务集群', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'grafana-instances', path: 'grafana-instances', label: 'Grafana', showInSidebar: true, sidebarSection: 'observability' },
  { key: 'stats', path: 'stats', label: '用量统计', showInSidebar: true, sidebarSection: 'observability' },

  // ── 管理 ──
  { key: 'departments', path: 'departments', label: '部门管理', showInSidebar: true, sidebarSection: 'admin' },
  { key: 'tenants', path: 'tenants', label: '租户管理', showInSidebar: true, sidebarSection: 'admin' },
  { key: 'users', path: 'users', label: '用户管理', showInSidebar: true, sidebarSection: 'admin', requireAdmin: true },
  { key: 'zones', path: 'zones', label: '可用区', showInSidebar: true, sidebarSection: 'admin' },
  { key: 'clusters', path: 'clusters', label: '可观测集群', showInSidebar: true, sidebarSection: 'admin' },
  { key: 'settings', path: 'settings', label: '系统设置', showInSidebar: true, sidebarSection: 'admin' },

  // ── 隐藏路由（侧边栏不显示，供内链跳转）──
  { key: 'grafana', path: 'grafana', label: 'Grafana 管理' },
  { key: 'grafana-hosts', path: 'grafana-hosts', label: 'Grafana 纳管实例' },
  { key: 'instance-detail', path: 'instances/:instanceId' },
  { key: 'grafana-instance-detail', path: 'grafana-instances/:instanceId' },
  { key: 'alerts', path: 'alerts' },
  { key: 'log-query', path: 'logs/query' },
];
