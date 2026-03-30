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
import OpenInNewIcon from '@mui/icons-material/OpenInNew';

const DRAWER_WIDTH = 256;

const menuItems = [
  { key: 'overview', label: '概览', icon: <DashboardOutlinedIcon />, path: '/dashboard' },
  { type: 'divider' as const },
  { key: 'departments', label: '部门管理', icon: <BusinessOutlinedIcon />, path: '/departments' },
  { key: 'tenants', label: '租户管理', icon: <GroupsOutlinedIcon />, path: '/tenants' },
  { key: 'instances', label: '实例管理', icon: <StorageOutlinedIcon />, path: '/instances' },
  { type: 'divider' as const },
  { key: 'grafana', label: 'Grafana 管理', icon: <BarChartOutlinedIcon />, path: '/grafana' },
  { key: 'alerts', label: '告警引擎', icon: <NotificationsOutlinedIcon />, path: '/alerts', external: true },
  { type: 'divider' as const },
  { key: 'users', label: '用户管理', icon: <PeopleOutlinedIcon />, path: '/users' },
  { key: 'settings', label: '系统设置', icon: <SettingsOutlinedIcon />, path: '/settings' },
];

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
        {menuItems.map((item, index) => {
          if ('type' in item && item.type === 'divider') {
            return <Divider key={`d-${index}`} sx={{ my: 1, mx: 2 }} />;
          }
          const menuItem = item as { key: string; label: string; icon: React.ReactNode; path: string; external?: boolean };
          const isSelected = location.pathname === menuItem.path || location.pathname.startsWith(menuItem.path + '/');
          return (
            <ListItemButton
              key={menuItem.key}
              selected={isSelected}
              onClick={() => {
                if (menuItem.external) {
                  navigate(menuItem.path);
                } else {
                  navigate(menuItem.path);
                }
                onClose?.();
              }}
              sx={{ mb: 0.25, py: 1 }}
            >
              <ListItemIcon>{menuItem.icon}</ListItemIcon>
              <ListItemText
                primary={menuItem.label}
                primaryTypographyProps={{ fontSize: '0.875rem', fontWeight: isSelected ? 600 : 400 }}
              />
              {menuItem.external && <OpenInNewIcon sx={{ fontSize: 14, color: 'text.disabled' }} />}
            </ListItemButton>
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
