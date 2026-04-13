import { lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import LoadingScreen from './components/common/LoadingScreen';
import { appRouteMeta, type AppRouteKey } from './config/appRoutes';

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

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LoadingScreen />}>{children}</Suspense>;
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function GuestGuard({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token');
  if (token) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

const routeComponentMap: Record<AppRouteKey, React.ReactNode | null> = {
  dashboard: <Lazy><DashboardPage /></Lazy>,
  departments: <Lazy><DepartmentPage /></Lazy>,
  tenants: <Lazy><TenantPage /></Lazy>,
  instances: <Lazy><InstancePage /></Lazy>,
  'instance-detail': <Lazy><InstanceDetailPage /></Lazy>,
  grafana: <Lazy><GrafanaPage /></Lazy>,
  alerts: <Lazy><AlertPage /></Lazy>,
  users: <Lazy><UserPage /></Lazy>,
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
