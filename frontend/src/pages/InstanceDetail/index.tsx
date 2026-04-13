import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  Chip,
  FormControl,
  Grid,
  InputLabel,
  LinearProgress,
  MenuItem,
  Select,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import RefreshIcon from '@mui/icons-material/Refresh';
import InsightsOutlinedIcon from '@mui/icons-material/InsightsOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import LoadingScreen from '../../components/common/LoadingScreen';
import DetailTabs from '../../components/common/DetailTabs';
import { instanceAPI } from '../../api/instance';
import type { Instance, InstanceMetrics, InstanceSpec } from '../../types/api';

const typeLabels: Record<string, { label: string; color: 'primary' | 'secondary' | 'success' | 'warning' }> = {
  metrics: { label: 'Metrics', color: 'primary' },
  logs: { label: 'Logs', color: 'secondary' },
  visual: { label: 'Grafana', color: 'success' },
  alert: { label: 'Alert', color: 'warning' },
};

function parseSpec(spec: string): InstanceSpec {
  try {
    return JSON.parse(spec);
  } catch {
    return { cpu: 0, memory: 0, storage: 0, retention: 0 };
  }
}

function ResourceBar({ label, used, total, unit }: { label: string; used: number; total: number; unit: string }) {
  const pct = total > 0 ? Math.min((used / total) * 100, 100) : 0;
  const color = pct > 85 ? 'error' : pct > 60 ? 'warning' : 'primary';
  return (
    <Box sx={{ mb: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
        <Typography variant="body2" color="text.secondary">{label}</Typography>
        <Typography variant="body2">{used} / {total} {unit} ({pct.toFixed(0)}%)</Typography>
      </Box>
      <LinearProgress variant="determinate" value={pct} color={color} sx={{ height: 7, borderRadius: 4 }} />
    </Box>
  );
}

function EndpointRow({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    if (!value || value === '-') return;
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };

  return (
    <Grid container spacing={1} sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
      <Grid size={{ xs: 4, md: 3 }}>
        <Typography variant="body2" color="text.secondary">{label}</Typography>
      </Grid>
      <Grid size={{ xs: 8, md: 9 }} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>{value || '-'}</Typography>
        <Button size="small" onClick={copy} disabled={!value || value === '-'}>{copied ? '已复制' : '复制'}</Button>
      </Grid>
    </Grid>
  );
}

export default function InstanceDetailPage() {
  const navigate = useNavigate();
  const { instanceId = '' } = useParams();
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(true);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [instance, setInstance] = useState<Instance | null>(null);
  const [metrics, setMetrics] = useState<InstanceMetrics | null>(null);
  const [activeTab, setActiveTab] = useState('base');
  const [timeRange, setTimeRange] = useState('1h');

  const fetchInstance = useCallback(async () => {
    if (!instanceId) return;
    setLoading(true);
    try {
      const { data: res } = await instanceAPI.get(instanceId);
      setInstance(res.data || null);
    } catch {
      enqueueSnackbar('获取实例详情失败', { variant: 'error' });
      navigate('/instances');
    } finally {
      setLoading(false);
    }
  }, [instanceId, enqueueSnackbar, navigate]);

  const fetchMetrics = useCallback(async () => {
    if (!instanceId) return;
    setMetricsLoading(true);
    try {
      const { data: res } = await instanceAPI.metrics(instanceId);
      setMetrics(res.data || null);
    } catch {
      setMetrics(null);
      enqueueSnackbar('获取实例监控数据失败', { variant: 'warning' });
    } finally {
      setMetricsLoading(false);
    }
  }, [instanceId, enqueueSnackbar]);

  useEffect(() => {
    fetchInstance();
    fetchMetrics();
  }, [fetchInstance, fetchMetrics]);

  const spec = useMemo(() => parseSpec(instance?.spec || '{}'), [instance?.spec]);

  if (loading) return <LoadingScreen />;
  if (!instance) {
    return (
      <Box>
        <Alert severity="error" sx={{ mb: 2 }}>实例不存在或已删除</Alert>
        <Button variant="outlined" onClick={() => navigate('/instances')}>返回实例列表</Button>
      </Box>
    );
  }

  const endpointBase = (instance.url || '').replace(/\/$/, '');
  const remoteWrite = endpointBase ? `${endpointBase}/api/v1/write` : '-';
  const remoteRead = endpointBase ? `${endpointBase}/api/v1/read` : '-';
  const httpApi = endpointBase ? `${endpointBase}/api/v1` : '-';

  return (
    <Box>
      <PageHeader
        title={instance.instance_name}
        subtitle="实例详情、监控与告警联动"
        extra={(
          <Button startIcon={<ArrowBackIcon />} variant="outlined" onClick={() => navigate('/instances')}>
            返回列表
          </Button>
        )}
      />

      <DetailTabs
        value={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'base',
            label: '基本信息',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Grid container spacing={2}>
                  <Grid size={{ xs: 12, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">实例名称</Typography>
                    <Typography variant="body1" sx={{ fontWeight: 500 }}>{instance.instance_name}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">类型</Typography>
                    <Chip
                      label={typeLabels[instance.instance_type]?.label || instance.instance_type}
                      color={typeLabels[instance.instance_type]?.color || 'default'}
                      size="small"
                    />
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">状态</Typography>
                    <StatusChip status={instance.status} />
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">模板</Typography>
                    <Typography variant="body2">{instance.template_type}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">命名空间</Typography>
                    <Typography variant="body2">{instance.namespace || '-'}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">CPU</Typography>
                    <Typography variant="body2">{spec.cpu} Core</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">内存</Typography>
                    <Typography variant="body2">{spec.memory} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">存储</Typography>
                    <Typography variant="body2">{spec.storage} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">创建时间</Typography>
                    <Typography variant="body2">{new Date(instance.created_at).toLocaleString()}</Typography>
                  </Grid>
                </Grid>
              </Card>
            ),
          },
          {
            key: 'service',
            label: '服务地址',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1.5 }}>服务连接信息</Typography>
                <EndpointRow label="访问地址" value={instance.url || '-'} />
                <EndpointRow label="Remote Write" value={remoteWrite} />
                <EndpointRow label="Remote Read" value={remoteRead} />
                <EndpointRow label="HTTP API" value={httpApi} />
                <Box sx={{ mt: 2 }}>
                  <Button
                    variant="outlined"
                    startIcon={<OpenInNewIcon />}
                    onClick={() => instance.url && window.open(instance.url, '_blank')}
                    disabled={!instance.url}
                  >
                    打开实例监控页面
                  </Button>
                </Box>
              </Card>
            ),
          },
          {
            key: 'monitor',
            label: '实例监控',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <InsightsOutlinedIcon color="primary" />
                    <Typography variant="subtitle2">资源监控</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <FormControl size="small" sx={{ minWidth: 140 }}>
                      <InputLabel>时间范围</InputLabel>
                      <Select value={timeRange} label="时间范围" onChange={(e) => setTimeRange(e.target.value)}>
                        <MenuItem value="15m">最近 15 分钟</MenuItem>
                        <MenuItem value="1h">最近 1 小时</MenuItem>
                        <MenuItem value="6h">最近 6 小时</MenuItem>
                        <MenuItem value="24h">最近 24 小时</MenuItem>
                      </Select>
                    </FormControl>
                    <Button startIcon={<RefreshIcon />} onClick={fetchMetrics} disabled={metricsLoading}>刷新</Button>
                  </Box>
                </Box>

                {metricsLoading ? (
                  <LinearProgress />
                ) : metrics ? (
                  <>
                    <ResourceBar
                      label="CPU"
                      used={Math.round(spec.cpu * (metrics.cpu_usage_percent || 0) / 100)}
                      total={spec.cpu}
                      unit="Core"
                    />
                    <ResourceBar
                      label="内存"
                      used={Math.round(spec.memory * (metrics.memory_usage_percent || 0) / 100)}
                      total={spec.memory}
                      unit="GiB"
                    />
                    <ResourceBar
                      label="存储"
                      used={Math.round(spec.storage * (metrics.disk_usage_percent || 0) / 100)}
                      total={spec.storage}
                      unit="GiB"
                    />
                    {metrics.note && (
                      <Alert severity="info">{metrics.note}</Alert>
                    )}
                  </>
                ) : (
                  <Alert severity="warning">暂无可用监控数据，请稍后重试。</Alert>
                )}
              </Card>
            ),
          },
          {
            key: 'alert',
            label: '告警管理',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>告警联动</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  可跳转到告警引擎查看当前实例对应的告警规则、告警事件与通知渠道。
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  <Button
                    variant="contained"
                    startIcon={<NotificationsActiveOutlinedIcon />}
                    onClick={() => navigate(`/alerts?instance_id=${instance.id}&instance_name=${encodeURIComponent(instance.instance_name)}`)}
                  >
                    打开告警引擎
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<LinkOutlinedIcon />}
                    onClick={() => navigate('/platform-scaling')}
                  >
                    前往平台扩容
                  </Button>
                </Box>
                <Alert severity="info" sx={{ mt: 2 }}>
                  独享集群与共享版的扩容入口不同：实例级操作仅适用于单节点独享实例，集群扩容请使用平台扩容页面。
                </Alert>
              </Card>
            ),
          },
        ]}
      />
    </Box>
  );
}
