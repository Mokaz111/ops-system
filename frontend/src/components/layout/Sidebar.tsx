import { useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  Box,
  Collapse,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
} from '@mui/material';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
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
import ManageSearchOutlinedIcon from '@mui/icons-material/ManageSearchOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import RuleOutlinedIcon from '@mui/icons-material/RuleOutlined';
import SendOutlinedIcon from '@mui/icons-material/SendOutlined';
import AccountTreeOutlinedIcon from '@mui/icons-material/AccountTreeOutlined';
import TimelineOutlinedIcon from '@mui/icons-material/TimelineOutlined';
import CategoryOutlinedIcon from '@mui/icons-material/CategoryOutlined';
import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined';
import { sidebarNav, type AppRouteKey, type SidebarNavItem } from '../../config/appRoutes';
import { useAuthStore } from '../../stores/useAuthStore';

const DRAWER_WIDTH = 256;
const OVERRIDE_KEY = 'ops.sidebar.expandedOverrides';

const iconMap: Record<AppRouteKey, React.ReactNode> = {
  dashboard: <HomeOutlinedIcon />,
  integrations: <ExtensionOutlinedIcon />,
  metrics: <DataUsageOutlinedIcon />,
  instances: <StorageOutlinedIcon />,
  'instance-create': <StorageOutlinedIcon />,
  'instance-detail': <StorageOutlinedIcon />,
  'log-instances': <DescriptionOutlinedIcon />,
  'log-query': <ManageSearchOutlinedIcon />,
  traces: <AccountTreeOutlinedIcon />,
  'business-clusters': <DnsOutlinedIcon />,
  umodel: <HubOutlinedIcon />,
  'grafana-instances': <VisibilityOutlinedIcon />,
  'grafana-instance-detail': <VisibilityOutlinedIcon />,
  stats: <BarChartOutlinedIcon />,
  alerts: <NotificationsActiveOutlinedIcon />,
  'alert-rules': <RuleOutlinedIcon />,
  'alert-events': <NotificationsActiveOutlinedIcon />,
  'alert-silences': <NotificationsPausedOutlinedIcon />,
  'alert-channels': <SendOutlinedIcon />,
  workspaces: <GroupsOutlinedIcon />,
  users: <PeopleOutlinedIcon />,
  zones: <LanguageOutlinedIcon />,
  clusters: <HubOutlinedIcon />,
  audit: <DescriptionOutlinedIcon />,
  settings: <SettingsOutlinedIcon />,
};

/** 分组图标（无 routeKey 或与叶子区分时使用） */
const groupIconMap: Record<string, React.ReactNode> = {
  monitoring: <TimelineOutlinedIcon />,
  logging: <DescriptionOutlinedIcon />,
  alerting: <NotificationsActiveOutlinedIcon />,
  resources: <CategoryOutlinedIcon />,
  admin: <AdminPanelSettingsOutlinedIcon />,
};

function pathMatches(pathname: string, path?: string): boolean {
  if (!path) return false;
  return pathname === path || pathname.startsWith(`${path}/`);
}

function itemActive(pathname: string, item: SidebarNavItem): boolean {
  if (item.children?.length) {
    if (item.children.some((c) => pathMatches(pathname, c.path))) return true;
    return (item.activePrefixes || []).some((p) => pathMatches(pathname, p));
  }
  return pathMatches(pathname, item.path);
}

function filterNav(items: SidebarNavItem[], isAdmin: boolean): SidebarNavItem[] {
  return items
    .filter((item) => !item.requireAdmin || isAdmin)
    .map((item) => {
      if (!item.children) return item;
      const children = filterNav(item.children, isAdmin);
      if (children.length === 0) return null;
      return { ...item, children };
    })
    .filter(Boolean) as SidebarNavItem[];
}

function loadOverrides(): Record<string, boolean> {
  try {
    const raw = sessionStorage.getItem(OVERRIDE_KEY);
    return raw ? (JSON.parse(raw) as Record<string, boolean>) : {};
  } catch {
    return {};
  }
}

interface SidebarProps {
  open: boolean;
  onClose?: () => void;
}

export default function Sidebar({ open, onClose }: SidebarProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const role = useAuthStore((s) => s.user?.role);
  const isAdmin = role === 'admin';
  const nav = useMemo(() => filterNav(sidebarNav, isAdmin), [isAdmin]);

  // 用户手动展开/收起的覆盖态；未覆盖时，路径命中的分组自动展开。
  const [overrides, setOverrides] = useState<Record<string, boolean>>(() => loadOverrides());

  const pathExpanded = useMemo(() => {
    const map: Record<string, boolean> = {};
    for (const item of nav) {
      if (item.children?.length && itemActive(location.pathname, item)) {
        map[item.id] = true;
      }
    }
    return map;
  }, [location.pathname, nav]);

  const isGroupOpen = (id: string) => (id in overrides ? overrides[id] : !!pathExpanded[id]);

  const toggleGroup = (id: string) => {
    setOverrides((prev) => {
      const currentlyOpen = id in prev ? prev[id] : !!pathExpanded[id];
      const next = { ...prev, [id]: !currentlyOpen };
      try {
        sessionStorage.setItem(OVERRIDE_KEY, JSON.stringify(next));
      } catch {
        /* ignore */
      }
      return next;
    });
  };

  const go = (path: string) => {
    navigate(path);
    onClose?.();
  };

  const renderLeaf = (item: SidebarNavItem, nested: boolean) => {
    const selected = pathMatches(location.pathname, item.path);
    const icon = (item.routeKey && iconMap[item.routeKey]) || groupIconMap[item.id];
    return (
      <ListItemButton
        key={item.id}
        selected={selected}
        onClick={() => item.path && go(item.path)}
        sx={{
          mb: 0.25,
          py: nested ? 0.75 : 1,
          pl: nested ? 4.5 : 1.5,
          borderRadius: 1,
        }}
      >
        <ListItemIcon sx={{ minWidth: nested ? 32 : 36, '& .MuiSvgIcon-root': { fontSize: nested ? 20 : 22 } }}>
          {icon}
        </ListItemIcon>
        <ListItemText
          primary={item.label}
          primaryTypographyProps={{ fontSize: nested ? '0.8125rem' : '0.875rem', fontWeight: selected ? 600 : 400 }}
        />
      </ListItemButton>
    );
  };

  const renderGroup = (item: SidebarNavItem) => {
    const openGroup = isGroupOpen(item.id);
    const active = itemActive(location.pathname, item);
    const icon = groupIconMap[item.id] || (item.routeKey && iconMap[item.routeKey]);
    return (
      <Box key={item.id}>
        <ListItemButton
          onClick={() => {
            // 点击分组：展开/收起；若当前未在该组下，同时跳到默认落地页。
            const willOpen = !openGroup;
            toggleGroup(item.id);
            if (willOpen && item.path && !active) {
              go(item.path);
            }
          }}
          sx={{
            mb: 0.25,
            py: 1,
            pl: 1.5,
            borderRadius: 1,
            bgcolor: active ? 'action.selected' : undefined,
          }}
        >
          <ListItemIcon sx={{ minWidth: 36 }}>{icon}</ListItemIcon>
          <ListItemText
            primary={item.label}
            primaryTypographyProps={{ fontSize: '0.875rem', fontWeight: active ? 600 : 500 }}
          />
          {openGroup ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </ListItemButton>
        <Collapse in={openGroup} timeout="auto" unmountOnExit>
          <List disablePadding dense>
            {item.children!.map((child) => renderLeaf(child, true))}
          </List>
        </Collapse>
      </Box>
    );
  };

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

      <List sx={{ flex: 1, px: 1, pt: 0.5, overflowY: 'auto' }} dense>
        {nav.map((item) => (item.children?.length ? renderGroup(item) : renderLeaf(item, false)))}
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
