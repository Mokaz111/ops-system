import { useLocation, useNavigate } from 'react-router-dom';
import {
  Box,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Typography,
} from '@mui/material';
import HomeOutlinedIcon from '@mui/icons-material/HomeOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import BarChartOutlinedIcon from '@mui/icons-material/BarChartOutlined';
import PeopleOutlinedIcon from '@mui/icons-material/PeopleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import ExtensionOutlinedIcon from '@mui/icons-material/ExtensionOutlined';
import DataUsageOutlinedIcon from '@mui/icons-material/DataUsageOutlined';
import DescriptionOutlinedIcon from '@mui/icons-material/DescriptionOutlined';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import LanguageOutlinedIcon from '@mui/icons-material/LanguageOutlined';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import { appRouteMeta, sectionLabels, type AppRouteKey, type SidebarSection } from '../../config/appRoutes';
import { useAuthStore } from '../../stores/useAuthStore';

const DRAWER_WIDTH = 256;

const iconMap: Record<AppRouteKey, React.ReactNode> = {
  dashboard: <HomeOutlinedIcon />,
  integrations: <ExtensionOutlinedIcon />,
  metrics: <DataUsageOutlinedIcon />,
  instances: <StorageOutlinedIcon />,
  'instance-create': <StorageOutlinedIcon />,
  'instance-detail': <StorageOutlinedIcon />,
  'log-instances': <DescriptionOutlinedIcon />,
  'log-query': <DescriptionOutlinedIcon />,
  'business-clusters': <DnsOutlinedIcon />,
  umodel: <HubOutlinedIcon />,
  'grafana-instances': <VisibilityOutlinedIcon />,
  'grafana-instance-detail': <VisibilityOutlinedIcon />,
  stats: <BarChartOutlinedIcon />,
  alerts: <BarChartOutlinedIcon />,
  workspaces: <GroupsOutlinedIcon />,
  users: <PeopleOutlinedIcon />,
  zones: <LanguageOutlinedIcon />,
  clusters: <HubOutlinedIcon />,
  audit: <DescriptionOutlinedIcon />,
  settings: <SettingsOutlinedIcon />,
};

const sectionOrder: SidebarSection[] = ['overview', 'observability', 'admin'];

const allSidebarRoutes = appRouteMeta.filter((route) => route.showInSidebar && route.label);

interface SidebarProps {
  open: boolean;
  onClose?: () => void;
}

export default function Sidebar({ open, onClose }: SidebarProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const role = useAuthStore((s) => s.user?.role);
  const sidebarRoutes = allSidebarRoutes.filter((r) => !r.requireAdmin || role === 'admin');

  const routeBySection = new Map<SidebarSection, typeof sidebarRoutes>();
  for (const section of sectionOrder) {
    routeBySection.set(
      section,
      sidebarRoutes.filter((r) => (r.sidebarSection || 'overview') === section),
    );
  }

  const drawerContent = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Brand */}
      <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1.5, minHeight: 64 }}>
        <Box
          sx={{
            width: 32,
            height: 32,
            borderRadius: '8px',
            background: 'linear-gradient(135deg, #4285f4 0%, #34a853 50%, #fbbc04 75%, #ea4335 100%)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Typography sx={{ color: '#fff', fontWeight: 700, fontSize: 14 }}>O</Typography>
        </Box>
        <Box>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
            Ops Platform
          </Typography>
          <Typography variant="caption" sx={{ color: 'text.secondary', fontSize: '0.7rem' }}>
            可观测性监控平台
          </Typography>
        </Box>
      </Box>

      {/* Navigation */}
      <List sx={{ flex: 1, px: 1, pt: 0.5 }} dense>
        {sectionOrder.map((section) => {
          const items = routeBySection.get(section) || [];
          if (items.length === 0) return null;

          const label = sectionLabels[section];

          return (
            <Box key={section}>
              {label ? (
                <ListSubheader
                  disableSticky
                  sx={{
                    fontSize: '0.6875rem',
                    fontWeight: 600,
                    lineHeight: '32px',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'text.disabled',
                    bgcolor: 'transparent',
                  }}
                >
                  {label}
                </ListSubheader>
              ) : null}

              {items.map((item) => {
                const path = `/${item.path}`;
                const isSelected = location.pathname === path || location.pathname.startsWith(`${path}/`);
                return (
                  <ListItemButton
                    key={item.key}
                    selected={isSelected}
                    onClick={() => {
                      navigate(path);
                      onClose?.();
                    }}
                    sx={{ mb: 0.25, py: 1, borderRadius: 1 }}
                  >
                    <ListItemIcon sx={{ minWidth: 36 }}>{iconMap[item.key]}</ListItemIcon>
                    <ListItemText
                      primary={item.label}
                      primaryTypographyProps={{ fontSize: '0.875rem', fontWeight: isSelected ? 600 : 400 }}
                    />
                  </ListItemButton>
                );
              })}
            </Box>
          );
        })}
      </List>
    </Box>
  );

  return (
    <Drawer
      variant="persistent"
      open={open}
      sx={{
        width: open ? DRAWER_WIDTH : 0,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
          borderRight: '1px solid',
          borderColor: 'divider',
          backgroundColor: 'background.default',
        },
      }}
    >
      {drawerContent}
    </Drawer>
  );
}

export { DRAWER_WIDTH };
