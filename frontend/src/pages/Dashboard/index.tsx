import { useEffect, useState } from 'react';
import { Box, Card, CardContent, Grid, Typography, Skeleton, Alert } from '@mui/material';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import SpeedOutlinedIcon from '@mui/icons-material/SpeedOutlined';
import PageHeader from '../../components/common/PageHeader';
import { tenantAPI } from '../../api/tenant';
import { instanceAPI } from '../../api/instance';

interface StatCard {
  label: string;
  value: string | number;
  change?: string;
  icon: React.ReactNode;
  color: string;
  bgColor: string;
}

export default function DashboardPage() {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({ tenants: 0, instances: 0 });

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [tenantRes, instanceRes] = await Promise.allSettled([
          tenantAPI.list({ page: 1, page_size: 1 }),
          instanceAPI.list({ page: 1, page_size: 1 }),
        ]);
        setStats({
          tenants: tenantRes.status === 'fulfilled' ? tenantRes.value.data.data?.total || 0 : 0,
          instances: instanceRes.status === 'fulfilled' ? instanceRes.value.data.data?.total || 0 : 0,
        });
      } catch {
        // keep defaults
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  const statCards: StatCard[] = [
    { label: '租户总数', value: stats.tenants, icon: <GroupsOutlinedIcon />, color: '#1a73e8', bgColor: '#e8f0fe' },
    { label: '实例总数', value: stats.instances, icon: <StorageOutlinedIcon />, color: '#1e8e3e', bgColor: '#e6f4ea' },
    { label: '活跃告警', value: '--', change: '由夜莺 N9E 提供', icon: <NotificationsActiveOutlinedIcon />, color: '#d93025', bgColor: '#fce8e6' },
    { label: '指标写入速率', value: '--', change: '监控接口接入后展示', icon: <SpeedOutlinedIcon />, color: '#e37400', bgColor: '#fef7e0' },
  ];

  return (
    <Box>
      <PageHeader title="概览" subtitle="平台运行状态总览" />

      <Grid container spacing={2.5} sx={{ mb: 3 }}>
        {statCards.map((card) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={card.label}>
            <Card>
              <CardContent sx={{ p: 2.5, '&:last-child': { pb: 2.5 } }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                      {card.label}
                    </Typography>
                    {loading ? (
                      <Skeleton width={60} height={36} />
                    ) : (
                      <Typography variant="h4" sx={{ fontWeight: 600, color: card.color }}>
                        {card.value}
                      </Typography>
                    )}
                    {card.change && (
                      <Typography variant="caption" color="text.secondary">
                        {card.change}
                      </Typography>
                    )}
                  </Box>
                  <Box sx={{ p: 1, borderRadius: 2, backgroundColor: card.bgColor, color: card.color }}>
                    {card.icon}
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Grid container spacing={2.5}>
        <Grid size={{ xs: 12, md: 8 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>
                指标趋势
              </Typography>
              <Alert severity="info">
                当前阶段未在平台侧聚合时序写入趋势，建议从 Grafana 或 VictoriaMetrics 面板查看实时趋势。
              </Alert>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>
                告警分布
              </Typography>
              <Alert severity="info">
                告警统计由夜莺 N9E 独立提供，请前往告警页进入夜莺控制台查看实时分布。
              </Alert>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
