import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Box, Button, Card, CardContent, Chip, Grid, Skeleton, Stack, Typography } from '@mui/material';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import CheckCircleOutlineOutlinedIcon from '@mui/icons-material/CheckCircleOutlineOutlined';
import { useTheme } from '@mui/material/styles';
import {
  Area, AreaChart, Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip as ChartTooltip, XAxis, YAxis,
} from 'recharts';
import PageHeader from '../../components/common/PageHeader';
import { workspaceAPI } from '../../api/workspace';
import { instanceAPI } from '../../api/instance';
import { alertAPI } from '../../api/alert';
import { useAuthStore } from '../../stores/useAuthStore';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import { isPlatformAdmin } from '../../utils/membership';
import type { AlertLevelStat, AlertRuleStat, AlertSummary, AlertTrendPoint } from '../../types/api';

const levelLabels: Record<string, string> = { critical: '严重', warning: '警告', info: '信息' };

export default function DashboardPage() {
  const theme = useTheme();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const admin = isPlatformAdmin(user);
  const workspaceId = useWorkspaceStore((s) => s.currentId);
  // 告警统计接口对平台管理员强制要求 workspace_id；普通用户单空间时可省略。
  const alertScopeReady = !admin || !!workspaceId;

  const [loading, setLoading] = useState(true);
  const [totals, setTotals] = useState({ workspaces: 0, instances: 0 });
  const [summary, setSummary] = useState<AlertSummary | null>(null);
  const [trend, setTrend] = useState<AlertTrendPoint[]>([]);
  const [byLevel, setByLevel] = useState<AlertLevelStat[]>([]);
  const [topRules, setTopRules] = useState<AlertRuleStat[]>([]);

  useEffect(() => {
    (async () => {
      const [wsRes, instRes] = await Promise.allSettled([
        workspaceAPI.list({ page: 1, page_size: 1 }),
        instanceAPI.list({ page: 1, page_size: 1, instance_type: 'metrics' }),
      ]);
      setTotals({
        workspaces: wsRes.status === 'fulfilled' ? wsRes.value.data.data?.total || 0 : 0,
        instances: instRes.status === 'fulfilled' ? instRes.value.data.data?.total || 0 : 0,
      });
      setLoading(false);
    })();
  }, []);

  useEffect(() => {
    // 未选工作空间时 UI 各处按 alertScopeReady 渲染占位，无需清空 state。
    if (!alertScopeReady) return;
    const end = new Date();
    const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
    const range = { start: start.toISOString(), end: end.toISOString() };
    const ws = workspaceId || undefined;
    (async () => {
      const [sRes, tRes, lRes, rRes] = await Promise.allSettled([
        alertAPI.statsSummary({ workspace_id: ws }),
        alertAPI.statsTrend({ workspace_id: ws, ...range, interval: 'hour' }),
        alertAPI.statsByLevel({ workspace_id: ws, ...range }),
        alertAPI.statsByRule({ workspace_id: ws, ...range, limit: 5 }),
      ]);
      setSummary(sRes.status === 'fulfilled' ? sRes.value.data.data : null);
      setTrend(tRes.status === 'fulfilled' ? tRes.value.data.data || [] : []);
      setByLevel(lRes.status === 'fulfilled' ? lRes.value.data.data || [] : []);
      setTopRules(rRes.status === 'fulfilled' ? rRes.value.data.data || [] : []);
    })();
  }, [alertScopeReady, workspaceId]);

  const trendData = useMemo(
    () => trend.map((p) => ({
      time: new Date(p.bucket).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      count: p.count,
    })),
    [trend],
  );

  const levelData = useMemo(
    () => byLevel.map((l) => ({ level: levelLabels[l.level] || l.level, count: l.count })),
    [byLevel],
  );

  const statCards = [
    {
      label: '工作空间',
      value: loading ? null : totals.workspaces,
      icon: <GroupsOutlinedIcon />,
      color: theme.palette.primary.main,
      onClick: () => navigate('/workspaces'),
    },
    {
      label: '监控实例',
      value: loading ? null : totals.instances,
      icon: <StorageOutlinedIcon />,
      color: theme.palette.success.main,
      onClick: () => navigate('/instances'),
    },
    {
      label: '告警中',
      value: alertScopeReady ? (summary ? summary.firing : null) : '--',
      icon: <NotificationsActiveOutlinedIcon />,
      color: theme.palette.error.main,
      onClick: () => navigate('/alerts/events'),
    },
    {
      label: '24h 已恢复',
      value: alertScopeReady ? (summary ? summary.resolved : null) : '--',
      icon: <CheckCircleOutlineOutlinedIcon />,
      color: theme.palette.warning.main,
      onClick: () => navigate('/alerts/events'),
    },
  ];

  return (
    <Box>
      <PageHeader title="概览" subtitle="平台运行状态与告警态势" />

      {admin && !workspaceId && (
        <Alert severity="info" sx={{ mb: 2 }}>
          在顶部选择一个工作空间即可查看该空间的告警态势（告警统计按工作空间隔离）。
        </Alert>
      )}

      <Grid container spacing={2.5} sx={{ mb: 3 }}>
        {statCards.map((card) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={card.label}>
            <Card sx={{ cursor: 'pointer' }} onClick={card.onClick}>
              <CardContent sx={{ p: 2.5, '&:last-child': { pb: 2.5 } }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <Box>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                      {card.label}
                    </Typography>
                    {card.value === null ? (
                      <Skeleton width={60} height={36} />
                    ) : (
                      <Typography variant="h4" sx={{ fontWeight: 600, color: card.color }}>
                        {card.value}
                      </Typography>
                    )}
                  </Box>
                  <Box sx={{ p: 1, borderRadius: 2, color: card.color, bgcolor: `${card.color}14` }}>
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
              <Typography variant="subtitle1" sx={{ mb: 2 }}>告警趋势（近 24 小时）</Typography>
              {!alertScopeReady ? (
                <Alert severity="info">选择工作空间后展示。</Alert>
              ) : trendData.length === 0 ? (
                <Alert severity="success">近 24 小时无告警事件。</Alert>
              ) : (
                <ResponsiveContainer width="100%" height={260}>
                  <AreaChart data={trendData} margin={{ top: 8, right: 8, bottom: 0, left: -20 }}>
                    <defs>
                      <linearGradient id="alertTrend" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor={theme.palette.error.main} stopOpacity={0.35} />
                        <stop offset="95%" stopColor={theme.palette.error.main} stopOpacity={0.02} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                    <XAxis dataKey="time" tick={{ fontSize: 12 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                    <ChartTooltip formatter={(v) => [v as number, '事件数']} />
                    <Area type="monotone" dataKey="count" stroke={theme.palette.error.main} fill="url(#alertTrend)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <Stack spacing={2.5}>
            <Card>
              <CardContent>
                <Typography variant="subtitle1" sx={{ mb: 2 }}>级别分布（近 24 小时）</Typography>
                {!alertScopeReady ? (
                  <Alert severity="info">选择工作空间后展示。</Alert>
                ) : levelData.length === 0 ? (
                  <Alert severity="success">近 24 小时无告警事件。</Alert>
                ) : (
                  <ResponsiveContainer width="100%" height={160}>
                    <BarChart data={levelData} margin={{ top: 8, right: 8, bottom: 0, left: -24 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                      <XAxis dataKey="level" tick={{ fontSize: 12 }} />
                      <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                      <ChartTooltip formatter={(v) => [v as number, '事件数']} />
                      <Bar dataKey="count" fill={theme.palette.warning.main} radius={[4, 4, 0, 0]} maxBarSize={48} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                  <Typography variant="subtitle1">Top 告警规则</Typography>
                  <Box sx={{ flex: 1 }} />
                  <Button size="small" onClick={() => navigate('/alerts/rules')}>查看全部</Button>
                </Box>
                {!alertScopeReady || topRules.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">暂无数据</Typography>
                ) : (
                  <Stack spacing={1}>
                    {topRules.map((r) => (
                      <Box key={r.rule_id} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="body2" sx={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {r.rule_name || r.rule_id.slice(0, 8)}
                        </Typography>
                        <Chip size="small" label={r.count} color="error" variant="outlined" />
                      </Box>
                    ))}
                  </Stack>
                )}
              </CardContent>
            </Card>
          </Stack>
        </Grid>
      </Grid>
    </Box>
  );
}
