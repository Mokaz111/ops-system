import { useLocation, useNavigate } from 'react-router-dom';
import {
  Box,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
  Divider,
} from '@mui/material';
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined';
import BusinessOutlinedIcon from '@mui/icons-material/BusinessOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import BarChartOutlinedIcon from '@mui/icons-material/BarChartOutlined';
import NotificationsOutlinedIcon from '@mui/icons-material/NotificationsOutlined';
import PeopleOutlinedIcon from '@mui/icons-material/PeopleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import SettingsEthernetOutlinedIcon from '@mui/icons-material/SettingsEthernetOutlined';
import { appRouteMeta, type AppRouteKey } from '../../config/appRoutes';

const DRAWER_WIDTH = 256;

const iconMap: Record<AppRouteKey, React.ReactNode> = {
  dashboard: <DashboardOutlinedIcon />,
  departments: <BusinessOutlinedIcon />,
  tenants: <GroupsOutlinedIcon />,
  instances: <StorageOutlinedIcon />,
  'instance-detail': <StorageOutlinedIcon />,
  grafana: <BarChartOutlinedIcon />,
  alerts: <NotificationsOutlinedIcon />,
  users: <PeopleOutlinedIcon />,
  'platform-scaling': <SettingsEthernetOutlinedIcon />,
  settings: <SettingsOutlinedIcon />,
};

const sidebarRoutes = appRouteMeta.filter((route) => route.showInSidebar && route.label);

interface SidebarProps {
  open: boolean;
  onClose?: () => void;
}

export default function Sidebar({ open, onClose }: SidebarProps) {
  const location = useLocation();
  const navigate = useNavigate();

  const drawerContent = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
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

      <List sx={{ flex: 1, px: 1, pt: 1 }}>
        {sidebarRoutes.map((menuItem, index) => {
          const prev = sidebarRoutes[index - 1];
          const showDivider = index > 0 && prev?.sidebarSection !== menuItem.sidebarSection;
          const path = `/${menuItem.path}`;
          const isSelected = location.pathname === path || location.pathname.startsWith(path + '/');
          return (
            <Box key={menuItem.key}>
              {showDivider && <Divider sx={{ my: 1, mx: 2 }} />}
              <ListItemButton
                selected={isSelected}
                onClick={() => {
                  navigate(path);
                  onClose?.();
                }}
                sx={{ mb: 0.25, py: 1 }}
              >
                <ListItemIcon>{iconMap[menuItem.key]}</ListItemIcon>
                <ListItemText
                  primary={menuItem.label}
                  primaryTypographyProps={{ fontSize: '0.875rem', fontWeight: isSelected ? 600 : 400 }}
                />
              </ListItemButton>
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
