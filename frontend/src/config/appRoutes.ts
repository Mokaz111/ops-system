export type AppRouteKey =
  | 'dashboard'
  | 'workspaces'
  | 'instances'
  | 'instance-create'
  | 'instance-detail'
  | 'grafana-instance-detail'
  | 'integrations'
  | 'metrics'
  | 'log-instances'
  | 'log-query'
  | 'traces'
  | 'grafana-instances'
  | 'alerts'
  | 'alert-rules'
  | 'alert-events'
  | 'alert-silences'
  | 'alert-channels'
  | 'users'
  | 'zones'
  | 'clusters'
  | 'business-clusters'
  | 'umodel'
  | 'stats'
  | 'settings'
  | 'audit';

export interface AppRouteMeta {
  key: AppRouteKey;
  path: string;
}

/** 路由表：只负责 path ↔ key 映射，侧栏结构见 sidebarNav。 */
export const appRouteMeta: AppRouteMeta[] = [
  { key: 'dashboard', path: 'dashboard' },
  { key: 'integrations', path: 'integrations' },
  { key: 'instances', path: 'instances' },
  { key: 'instance-create', path: 'instances/create' },
  { key: 'instance-detail', path: 'instances/:instanceId' },
  { key: 'metrics', path: 'metrics' },
  { key: 'grafana-instances', path: 'grafana-instances' },
  { key: 'grafana-instance-detail', path: 'grafana-instances/:instanceId' },
  { key: 'log-query', path: 'logs/query' },
  { key: 'log-instances', path: 'log-instances' },
  { key: 'traces', path: 'traces' },
  { key: 'alerts', path: 'alerts' },
  { key: 'alert-rules', path: 'alerts/rules' },
  { key: 'alert-events', path: 'alerts/events' },
  { key: 'alert-silences', path: 'alerts/silences' },
  { key: 'alert-channels', path: 'alerts/channels' },
  { key: 'business-clusters', path: 'business-clusters' },
  { key: 'umodel', path: 'umodel' },
  { key: 'stats', path: 'stats' },
  { key: 'workspaces', path: 'workspaces' },
  { key: 'users', path: 'users' },
  { key: 'zones', path: 'zones' },
  { key: 'clusters', path: 'clusters' },
  { key: 'audit', path: 'audit' },
  { key: 'settings', path: 'settings' },
];

/**
 * 侧栏导航树。
 * - 叶子节点必须有 path
 * - 分组节点可有 defaultPath（点击标题时跳转），children 为子入口
 * - 未出现在树中的路由视为子页面，由父页内链进入（如通知渠道、集群详情）
 */
export interface SidebarNavItem {
  id: string;
  label: string;
  /** 叶子或分组默认落地页 */
  path?: string;
  /** 用于图标映射；分组可省略 */
  routeKey?: AppRouteKey;
  /**
   * 分组激活前缀：覆盖子页面（如 /alerts/channels）时仍展开并高亮该组。
   * 未设置时仅按 children 的 path 匹配。
   */
  activePrefixes?: string[];
  requireAdmin?: boolean;
  children?: SidebarNavItem[];
}

export const sidebarNav: SidebarNavItem[] = [
  { id: 'dashboard', label: '概览', path: '/dashboard', routeKey: 'dashboard' },
  { id: 'integrations', label: '接入中心', path: '/integrations', routeKey: 'integrations' },

  {
    id: 'monitoring',
    label: '监控',
    path: '/instances',
    routeKey: 'instances',
    activePrefixes: ['/instances', '/metrics', '/grafana-instances'],
    children: [
      { id: 'instances', label: '监控实例', path: '/instances', routeKey: 'instances' },
      { id: 'metrics', label: '指标库', path: '/metrics', routeKey: 'metrics' },
      { id: 'grafana', label: 'Grafana', path: '/grafana-instances', routeKey: 'grafana-instances' },
    ],
  },

  {
    id: 'logging',
    label: '日志',
    path: '/logs/query',
    routeKey: 'log-query',
    activePrefixes: ['/logs', '/log-instances'],
    children: [
      { id: 'log-query', label: '日志查询', path: '/logs/query', routeKey: 'log-query' },
      { id: 'log-instances', label: '日志实例', path: '/log-instances', routeKey: 'log-instances' },
    ],
  },

  { id: 'traces', label: '链路追踪', path: '/traces', routeKey: 'traces' },

  {
    id: 'alerting',
    label: '告警',
    path: '/alerts/events',
    routeKey: 'alert-events',
    activePrefixes: ['/alerts'],
    children: [
      { id: 'alert-events', label: '事件', path: '/alerts/events', routeKey: 'alert-events' },
      { id: 'alert-rules', label: '规则', path: '/alerts/rules', routeKey: 'alert-rules' },
      { id: 'alert-silences', label: '静默', path: '/alerts/silences', routeKey: 'alert-silences' },
    ],
  },

  {
    id: 'resources',
    label: '资源',
    path: '/business-clusters',
    routeKey: 'business-clusters',
    children: [
      { id: 'business-clusters', label: '业务集群', path: '/business-clusters', routeKey: 'business-clusters' },
      { id: 'umodel', label: 'UModel', path: '/umodel', routeKey: 'umodel' },
    ],
  },

  {
    id: 'admin',
    label: '管理',
    path: '/workspaces',
    routeKey: 'workspaces',
    children: [
      { id: 'workspaces', label: '工作空间', path: '/workspaces', routeKey: 'workspaces' },
      { id: 'users', label: '用户', path: '/users', routeKey: 'users', requireAdmin: true },
      { id: 'zones', label: '可用区', path: '/zones', routeKey: 'zones' },
      { id: 'audit', label: '审计', path: '/audit', routeKey: 'audit', requireAdmin: true },
      { id: 'settings', label: '系统设置', path: '/settings', routeKey: 'settings' },
    ],
  },
];
