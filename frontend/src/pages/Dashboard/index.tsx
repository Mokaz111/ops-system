import { useEffect, useState } from 'react';
import { Box, Card, CardContent, Grid, Typography, Skeleton } from '@mui/material';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import SpeedOutlinedIcon from '@mui/icons-material/SpeedOutlined';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
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

const trendData = [
  { date: '03-24', value: 12000 },
  { date: '03-25', value: 15000 },
  { date: '03-26', value: 13500 },
  { date: '03-27', value: 18000 },
  { date: '03-28', value: 16000 },
  { date: '03-29', value: 21000 },
  { date: '03-30', value: 19500 },
];

const alertDistribution = [
  { name: '严重', value: 2, color: '#d93025' },
  { name: '警告', value: 5, color: '#f9ab00' },
  { name: '信息', value: 8, color: '#1a73e8' },
];

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
    { label: '租户总数', value: stats.tenants, change: '+2 本月', icon: <GroupsOutlinedIcon />, color: '#1a73e8', bgColor: '#e8f0fe' },
    { label: '实例总数', value: stats.instances, change: '+5 本月', icon: <StorageOutlinedIcon />, color: '#1e8e3e', bgColor: '#e6f4ea' },
    { label: '活跃告警', value: 3, change: '-1 较昨日', icon: <NotificationsActiveOutlinedIcon />, color: '#d93025', bgColor: '#fce8e6' },
    { label: '指标写入速率', value: '2.1M/s', icon: <SpeedOutlinedIcon />, color: '#e37400', bgColor: '#fef7e0' },
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
                指标写入趋势 (近 7 天)
              </Typography>
              <ResponsiveContainer width="100%" height={280}>
                <LineChart data={trendData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e8eaed" />
                  <XAxis dataKey="date" tick={{ fontSize: 12 }} stroke="#9aa0a6" />
                  <YAxis tick={{ fontSize: 12 }} stroke="#9aa0a6" />
                  <Tooltip />
                  <Line type="monotone" dataKey="value" stroke="#1a73e8" strokeWidth={2} dot={{ r: 3 }} activeDot={{ r: 5 }} />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>
                告警分布
              </Typography>
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie data={alertDistribution} cx="50%" cy="50%" innerRadius={50} outerRadius={80} dataKey="value" label={({ name, value }) => `${name}: ${value}`}>
                    {alertDistribution.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
              <Box sx={{ display: 'flex', justifyContent: 'center', gap: 2, mt: 1 }}>
                {alertDistribution.map((item) => (
                  <Box key={item.name} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Box sx={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: item.color }} />
                    <Typography variant="caption">
                      {item.name}: {item.value}
                    </Typography>
                  </Box>
                ))}
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
