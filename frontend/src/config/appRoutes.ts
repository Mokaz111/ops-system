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
  | 'clusters'
  | 'platform-scaling'
  | 'settings'
  | 'vm-stats'
  | 'log-stats';

export type SidebarSection = 'overview' | 'observability' | 'system';

export interface AppRouteMeta {
  key: AppRouteKey;
  path: string;
  label?: string;
  showInSidebar?: boolean;
  sidebarSection?: SidebarSection;
  sidebarSubGroup?: string;
  requireAdmin?: boolean;
}

export const sectionLabels: Record<SidebarSection, string> = {
  overview: '',
  observability: '可观测性',
  system: '系统管理',
};

export const appRouteMeta: AppRouteMeta[] = [
  // 概览
  { key: 'dashboard', path: 'dashboard', label: '概览', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'dashboard-mgmt', path: 'dashboards', label: 'Dashboard 管理', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'integrations', path: 'integrations', label: '接入中心', showInSidebar: true, sidebarSection: 'overview' },
  { key: 'metrics', path: 'metrics', label: '指标库', showInSidebar: true, sidebarSection: 'overview' },

  // 可观测性
  { key: 'instances', path: 'instances', label: '实例管理', showInSidebar: true, sidebarSection: 'observability', sidebarSubGroup: 'VictoriaMetrics中心' },
  { key: 'vm-stats', path: 'vm-stats', label: '用量统计', showInSidebar: true, sidebarSection: 'observability', sidebarSubGroup: 'VictoriaMetrics中心' },
  { key: 'grafana-instances', path: 'grafana-instances', label: 'Grafana 实例', showInSidebar: true, sidebarSection: 'observability', sidebarSubGroup: 'Grafana服务' },
  { key: 'log-instances', path: 'log-instances', label: '实例管理', showInSidebar: true, sidebarSection: 'observability', sidebarSubGroup: '日志中心' },
  { key: 'log-stats', path: 'log-stats', label: '用量统计', showInSidebar: true, sidebarSection: 'observability', sidebarSubGroup: '日志中心' },

  // 系统管理
  { key: 'departments', path: 'departments', label: '部门管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'tenants', path: 'tenants', label: '租户管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'users', path: 'users', label: '用户管理', showInSidebar: true, sidebarSection: 'system', requireAdmin: true },
  { key: 'clusters', path: 'clusters', label: '集群管理', showInSidebar: true, sidebarSection: 'system' },
  { key: 'platform-scaling', path: 'platform-scaling', label: '平台扩容', showInSidebar: true, sidebarSection: 'system', requireAdmin: true },
  { key: 'settings', path: 'settings', label: '系统设置', showInSidebar: true, sidebarSection: 'system' },

  // 隐藏路由（不在侧边栏显示，供内链跳转）
  { key: 'grafana', path: 'grafana', label: 'Grafana 管理', sidebarSection: 'observability' },
  { key: 'grafana-hosts', path: 'grafana-hosts', label: 'Grafana 纳管实例', sidebarSection: 'observability' },
  { key: 'instance-detail', path: 'instances/:instanceId' },
  { key: 'grafana-instance-detail', path: 'grafana-instances/:instanceId' },
  { key: 'alerts', path: 'alerts' },
  { key: 'log-query', path: 'logs/query' },
];
