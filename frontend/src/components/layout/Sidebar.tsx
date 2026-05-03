import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  Box,
  Collapse,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Typography,
} from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import HomeOutlinedIcon from '@mui/icons-material/HomeOutlined';
import BusinessOutlinedIcon from '@mui/icons-material/BusinessOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import BarChartOutlinedIcon from '@mui/icons-material/BarChartOutlined';
import PeopleOutlinedIcon from '@mui/icons-material/PeopleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import SettingsEthernetOutlinedIcon from '@mui/icons-material/SettingsEthernetOutlined';
import ExtensionOutlinedIcon from '@mui/icons-material/ExtensionOutlined';
import DataUsageOutlinedIcon from '@mui/icons-material/DataUsageOutlined';
import DescriptionOutlinedIcon from '@mui/icons-material/DescriptionOutlined';
import GridViewOutlinedIcon from '@mui/icons-material/GridViewOutlined';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import { appRouteMeta, sectionLabels, type AppRouteKey, type SidebarSection } from '../../config/appRoutes';
import { useAuthStore } from '../../stores/useAuthStore';

const DRAWER_WIDTH = 256;

const iconMap: Record<AppRouteKey, React.ReactNode> = {
  dashboard: <HomeOutlinedIcon />,
  departments: <BusinessOutlinedIcon />,
  tenants: <GroupsOutlinedIcon />,
  instances: <StorageOutlinedIcon />,
  'instance-detail': <StorageOutlinedIcon />,
  integrations: <ExtensionOutlinedIcon />,
  metrics: <DataUsageOutlinedIcon />,
  'log-instances': <DescriptionOutlinedIcon />,
  'log-query': <DescriptionOutlinedIcon />,
  grafana: <BarChartOutlinedIcon />,
  'grafana-instance-detail': <VisibilityOutlinedIcon />,
  'grafana-instances': <VisibilityOutlinedIcon />,
  'grafana-hosts': <VisibilityOutlinedIcon />,
  'dashboard-mgmt': <GridViewOutlinedIcon />,
  alerts: <BarChartOutlinedIcon />,
  users: <PeopleOutlinedIcon />,
  clusters: <HubOutlinedIcon />,
  'platform-scaling': <SettingsEthernetOutlinedIcon />,
  settings: <SettingsOutlinedIcon />,
  'vm-stats': <BarChartOutlinedIcon />,
  'log-stats': <BarChartOutlinedIcon />,
};

const sectionOrder: SidebarSection[] = ['overview', 'observability', 'system'];

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

  // Track expanded state for sub-groups. Keyed by "section//subGroup".
  const [expandedSubGroups, setExpandedSubGroups] = useState<Record<string, boolean>>({});

  const toggleSubGroup = (key: string) => {
    setExpandedSubGroups((prev) => ({ ...prev, [key]: !prev[key] }));
  };

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

          // Group items: those with subGroup vs. flat items
          const subGroupItems = items.filter((r) => r.sidebarSubGroup);
          const flatItems = items.filter((r) => !r.sidebarSubGroup);

          // Collect unique sub-groups preserving order
          const subGroups: string[] = [];
          for (const r of subGroupItems) {
            if (!subGroups.includes(r.sidebarSubGroup!)) {
              subGroups.push(r.sidebarSubGroup!);
            }
          }

          // Render a single nav item
          const renderItem = (item: (typeof items)[number]) => {
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
          };

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

              {/* Flat items (no sub-group) */}
              {flatItems.map(renderItem)}

              {/* Nested sub-groups */}
              {subGroups.map((subGroup) => {
                const sgKey = `${section}//${subGroup}`;
                const isExpanded = expandedSubGroups[sgKey] !== false; // default expanded
                const sgItems = subGroupItems.filter((r) => r.sidebarSubGroup === subGroup);
                const anySelected = sgItems.some((r) => {
                  const path = `/${r.path}`;
                  return location.pathname === path || location.pathname.startsWith(`${path}/`);
                });

                return (
                  <Box key={sgKey}>
                    <ListItemButton
                      onClick={() => toggleSubGroup(sgKey)}
                      sx={{
                        mb: 0.25,
                        py: 0.75,
                        borderRadius: 1,
                      }}
                    >
                      <ListItemText
                        primary={subGroup}
                        primaryTypographyProps={{
                          fontSize: '0.8125rem',
                          fontWeight: anySelected ? 600 : 500,
                          color: anySelected ? 'text.primary' : 'text.secondary',
                        }}
                      />
                      {isExpanded ? (
                        <ExpandLess sx={{ fontSize: 18, color: 'text.disabled' }} />
                      ) : (
                        <ExpandMore sx={{ fontSize: 18, color: 'text.disabled' }} />
                      )}
                    </ListItemButton>
                    <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                      <Box sx={{ pl: 1 }}>
                        {sgItems.map(renderItem)}
                      </Box>
                    </Collapse>
                  </Box>
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
