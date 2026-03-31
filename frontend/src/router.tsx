import { lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import LoadingScreen from './components/common/LoadingScreen';

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
      { path: 'dashboard', element: <Lazy><DashboardPage /></Lazy> },
      { path: 'departments', element: <Lazy><DepartmentPage /></Lazy> },
      { path: 'tenants', element: <Lazy><TenantPage /></Lazy> },
      { path: 'instances', element: <Lazy><InstancePage /></Lazy> },
      { path: 'grafana', element: <Lazy><GrafanaPage /></Lazy> },
      { path: 'alerts', element: <Lazy><AlertPage /></Lazy> },
      { path: 'users', element: <Lazy><UserPage /></Lazy> },
      { path: 'settings', element: <Lazy><SettingsPage /></Lazy> },
      { path: 'platform-scaling', element: <Lazy><PlatformScalingPage /></Lazy> },
    ],
  },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
]);
