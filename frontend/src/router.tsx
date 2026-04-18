import { lazy, Suspense, useEffect } from 'react';
import { createBrowserRouter, Navigate, useNavigate } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import LoadingScreen from './components/common/LoadingScreen';
import { appRouteMeta, type AppRouteKey } from './config/appRoutes';
import { UNAUTHORIZED_EVENT } from './api';
import { useAuthStore } from './stores/useAuthStore';

const LoginPage = lazy(() => import('./pages/Login'));
const DashboardPage = lazy(() => import('./pages/Dashboard'));
const DepartmentPage = lazy(() => import('./pages/Department'));
const TenantPage = lazy(() => import('./pages/Tenant'));
const InstancePage = lazy(() => import('./pages/Instance'));
const GrafanaPage = lazy(() => import('./pages/Grafana'));
const AlertPage = lazy(() => import('./pages/Alert'));
const UserPage = lazy(() => import('./pages/User'));
const SettingsPage = lazy(() => import('./pages/Settings'));
const PlatformScalingPage = lazy(() => import('./pages/PlatformScaling'));
const InstanceDetailPage = lazy(() => import('./pages/InstanceDetail'));
const IntegrationPage = lazy(() => import('./pages/Integration'));
const MetricPage = lazy(() => import('./pages/Metric'));
const LogInstancePage = lazy(() => import('./pages/LogInstance'));
const LogQueryPage = lazy(() => import('./pages/LogQuery'));
const DashboardMgmtPage = lazy(() => import('./pages/DashboardMgmt'));
const ClusterPage = lazy(() => import('./pages/Cluster'));
const GrafanaHostPage = lazy(() => import('./pages/GrafanaHost'));

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LoadingScreen />}>{children}</Suspense>;
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  // 订阅 store 的 token，401 拦截器把 token 清掉后会自动重渲染并跳到 /login，
  // 不再依赖外层手动监听 UNAUTHORIZED_EVENT 才能感知。
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

const routeComponentMap: Record<AppRouteKey, React.ReactNode | null> = {
  dashboard: <Lazy><DashboardPage /></Lazy>,
  departments: <Lazy><DepartmentPage /></Lazy>,
  tenants: <Lazy><TenantPage /></Lazy>,
  instances: <Lazy><InstancePage /></Lazy>,
  'instance-detail': <Lazy><InstanceDetailPage /></Lazy>,
  integrations: <Lazy><IntegrationPage /></Lazy>,
  metrics: <Lazy><MetricPage /></Lazy>,
  'log-instances': <Lazy><LogInstancePage /></Lazy>,
  'log-query': <Lazy><LogQueryPage /></Lazy>,
  grafana: <Lazy><GrafanaPage /></Lazy>,
  'grafana-hosts': <Lazy><GrafanaHostPage /></Lazy>,
  'dashboard-mgmt': <Lazy><DashboardMgmtPage /></Lazy>,
  alerts: <Lazy><AlertPage /></Lazy>,
  users: <Lazy><UserPage /></Lazy>,
  clusters: <Lazy><ClusterPage /></Lazy>,
  settings: <Lazy><SettingsPage /></Lazy>,
  'platform-scaling': <Lazy><PlatformScalingPage /></Lazy>,
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
