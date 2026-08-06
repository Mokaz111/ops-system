import { lazy, Suspense, useEffect } from 'react';
import { createBrowserRouter, Navigate, useNavigate, useSearchParams } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import LoadingScreen from './components/common/LoadingScreen';
import { appRouteMeta, type AppRouteKey } from './config/appRoutes';
import { UNAUTHORIZED_EVENT } from './api';
import { useAuthStore } from './stores/useAuthStore';

const LoginPage = lazy(() => import('./pages/Login'));
const DashboardPage = lazy(() => import('./pages/Dashboard'));
const WorkspacePage = lazy(() => import('./pages/Workspace'));
const InstancePage = lazy(() => import('./pages/Instance'));
const AlertRulesPage = lazy(() => import('./pages/Alert/Rules'));
const AlertEventsPage = lazy(() => import('./pages/Alert/Events'));
const AlertChannelsPage = lazy(() => import('./pages/Alert/Channels'));
const AlertSilencesPage = lazy(() => import('./pages/Alert/Silences'));
const UserPage = lazy(() => import('./pages/User'));
const SettingsPage = lazy(() => import('./pages/Settings'));
const InstanceCreatePage = lazy(() => import('./pages/Instance/Create'));
const InstanceDetailPage = lazy(() => import('./pages/InstanceDetail'));
const IntegrationPage = lazy(() => import('./pages/Integration'));
const MetricPage = lazy(() => import('./pages/Metric'));
const LogInstancePage = lazy(() => import('./pages/LogInstance'));
const LogQueryPage = lazy(() => import('./pages/LogQuery'));
const TracePage = lazy(() => import('./pages/Trace'));
const ZonePage = lazy(() => import('./pages/Zone'));
const ClusterPage = lazy(() => import('./pages/Cluster'));
const BusinessClusterPage = lazy(() => import('./pages/BusinessCluster'));
const UModelPage = lazy(() => import('./pages/UModel'));
const GrafanaInstancePage = lazy(() => import('./pages/GrafanaInstance'));
const GrafanaInstanceDetailPage = lazy(() => import('./pages/GrafanaInstanceDetail'));
const StatsPage = lazy(() => import('./pages/Stats'));
const AuditPage = lazy(() => import('./pages/Audit'));

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LoadingScreen />}>{children}</Suspense>;
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const token = useAuthStore((s) => s.token);

  useEffect(() => {
    const onUnauthorized = () => navigate('/login', { replace: true });
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [navigate]);

  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function GuestGuard({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  if (token) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

// 兼容旧 /alerts 链接：保留查询参数重定向到规则页。
function AlertsRedirect() {
  const [searchParams] = useSearchParams();
  const qs = searchParams.toString();
  return <Navigate to={`/alerts/rules${qs ? `?${qs}` : ''}`} replace />;
}

const routeComponentMap: Record<AppRouteKey, React.ReactNode | null> = {
  dashboard: <Lazy><DashboardPage /></Lazy>,
  integrations: <Lazy><IntegrationPage /></Lazy>,
  metrics: <Lazy><MetricPage /></Lazy>,
  instances: <Lazy><InstancePage /></Lazy>,
  'instance-create': <Lazy><InstanceCreatePage /></Lazy>,
  'instance-detail': <Lazy><InstanceDetailPage /></Lazy>,
  'log-instances': <Lazy><LogInstancePage /></Lazy>,
  'log-query': <Lazy><LogQueryPage /></Lazy>,
  traces: <Lazy><TracePage /></Lazy>,
  'business-clusters': <Lazy><BusinessClusterPage /></Lazy>,
  umodel: <Lazy><UModelPage /></Lazy>,
  'grafana-instances': <Lazy><GrafanaInstancePage /></Lazy>,
  'grafana-instance-detail': <Lazy><GrafanaInstanceDetailPage /></Lazy>,
  stats: <Lazy><StatsPage /></Lazy>,
  alerts: <AlertsRedirect />,
  'alert-rules': <Lazy><AlertRulesPage /></Lazy>,
  'alert-events': <Lazy><AlertEventsPage /></Lazy>,
  'alert-silences': <Lazy><AlertSilencesPage /></Lazy>,
  'alert-channels': <Lazy><AlertChannelsPage /></Lazy>,
  workspaces: <Lazy><WorkspacePage /></Lazy>,
  users: <Lazy><UserPage /></Lazy>,
  zones: <Lazy><ZonePage /></Lazy>,
  clusters: <Lazy><ClusterPage /></Lazy>,
  audit: <Lazy><AuditPage /></Lazy>,
  settings: <Lazy><SettingsPage /></Lazy>,
};

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <GuestGuard><Lazy><LoginPage /></Lazy></GuestGuard>,
  },
  {
    path: '/',
    element: <AuthGuard><AppLayout /></AuthGuard>,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      ...appRouteMeta.map((route) => ({ path: route.path, element: routeComponentMap[route.key] })),
    ],
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
]);
