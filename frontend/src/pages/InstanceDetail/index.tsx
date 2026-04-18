import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Card,
  Chip,
  CircularProgress,
  Grid,
  LinearProgress,
  Link as MuiLink,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExtensionOutlinedIcon from '@mui/icons-material/ExtensionOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import LoadingScreen from '../../components/common/LoadingScreen';
import EmptyState from '../../components/common/EmptyState';
import DetailTabs from '../../components/common/DetailTabs';
import { instanceAPI, type ScaleEvent } from '../../api/instance';
import {
  integrationAPI,
  latestAppliedRefs,
  type AppliedRef,
  type IntegrationInstallation,
  type IntegrationInstallationRevision,
} from '../../api/integration';
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { extractApiError } from '../../api';
import { isAbortError, makeAbortController } from '../../api/client';
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

function ScaleEventList({ events, loading }: { events: ScaleEvent[]; loading: boolean }) {
  if (loading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <CircularProgress size={16} />
        <Typography variant="caption">加载伸缩事件...</Typography>
      </Box>
    );
  }
  if (events.length === 0) {
    return <EmptyState title="暂无伸缩事件" description="通过 /api/v1/instances/:id/scale 触发后会在此处展示。" />;
  }
  const methodChip = (m: string) => {
    const colorMap: Record<string, 'primary' | 'secondary' | 'default' | 'error'> = {
      cr_patch: 'primary',
      helm_upgrade: 'secondary',
      k8s_native: 'default',
      rejected: 'error',
    };
    return <Chip size="small" label={m} color={colorMap[m] || 'default'} variant="outlined" />;
  };
  return (
    <Table size="small">
      <TableHead>
        <TableRow>
          <TableCell>时间</TableCell>
          <TableCell>类型</TableCell>
          <TableCell>生效路径</TableCell>
          <TableCell>参数</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>操作人</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {events.map((e) => {
          const params: string[] = [];
          if (e.replicas != null) params.push(`replicas=${e.replicas}`);
          if (e.cpu) params.push(`cpu=${e.cpu}`);
          if (e.memory) params.push(`memory=${e.memory}`);
          if (e.storage) params.push(`storage=${e.storage}`);
          return (
            <TableRow key={e.id}>
              <TableCell>
                <Typography variant="caption">{new Date(e.created_at).toLocaleString()}</Typography>
              </TableCell>
              <TableCell>
                <Chip size="small" label={e.scale_type} />
              </TableCell>
              <TableCell>{methodChip(e.method)}</TableCell>
              <TableCell>
                <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                  {params.join(' · ') || '-'}
                </Typography>
              </TableCell>
              <TableCell>
                <Chip
                  size="small"
                  label={e.status}
                  color={e.status === 'success' ? 'success' : 'error'}
                />
                {e.error_message && (
                  <Typography variant="caption" color="error" sx={{ display: 'block', maxWidth: 260, wordBreak: 'break-all' }}>
                    {e.error_message}
                  </Typography>
                )}
              </TableCell>
              <TableCell>
                <Typography variant="caption">{e.operator || '-'}</Typography>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function statusColor(s?: string): 'success' | 'warning' | 'error' | 'default' {
  switch (s) {
    case 'success':
    case 'uninstalled':
      return 'success';
    case 'partial':
    case 'rendered':
      return 'warning';
    case 'failed':
    case 'preflight_failed':
    case 'uninstall_failed':
      return 'error';
    default:
      return 'default';
  }
}

// installation/revision 状态在 Chip 上的中文标签；后端值原样保留为 fallback。
function statusLabel(s?: string): string {
  switch (s) {
    case 'success': return '成功';
    case 'partial': return '部分成功';
    case 'rendered': return '仅渲染';
    case 'failed': return '失败';
    case 'preflight_failed': return '预检失败';
    case 'uninstalled': return '已卸载';
    case 'uninstall_failed': return '卸载失败';
    default: return s || '-';
  }
}

// revision.action 中文标签：reinstall 是 stage-5 INS-1 引入的"卸载后再装回来"语义。
function actionLabel(a?: string): string {
  switch (a) {
    case 'install': return '安装';
    case 'upgrade': return '升级';
    case 'reinstall': return '重新安装';
    case 'uninstall': return '卸载';
    default: return a || '-';
  }
}

function AppliedRefsTable({ refs, grafanaHosts }: { refs: AppliedRef[]; grafanaHosts: GrafanaHost[] }) {
  if (refs.length === 0) {
    return (
      <Alert severity="info" sx={{ mt: 1 }}>
        暂无已应用资源明细。可能为仅渲染未下发（rendered）状态，或该实例的 Apply 过程未产生资源。
      </Alert>
    );
  }
  const hostById: Record<string, GrafanaHost> = {};
  for (const h of grafanaHosts) hostById[h.id] = h;
  return (
    <Table size="small" sx={{ mt: 1 }}>
      <TableHead>
        <TableRow>
          <TableCell>Target</TableCell>
          <TableCell>资源</TableCell>
          <TableCell>命名空间 / 位置</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>动作</TableCell>
          <TableCell />
        </TableRow>
      </TableHead>
      <TableBody>
        {refs.map((r, idx) => {
          let locLabel = r.namespace || '-';
          let openLink: string | null = null;
          if (r.target === 'grafana') {
            const host = r.grafana_host_id ? hostById[r.grafana_host_id] : null;
            locLabel = host ? `${host.name}${r.grafana_org ? ` · org=${r.grafana_org}` : ''}` : `org=${r.grafana_org ?? '-'}`;
            if (host && r.uid) {
              openLink = `${host.url.replace(/\/$/, '')}/d/${r.uid}`;
            }
          }
          return (
            <TableRow key={idx}>
              <TableCell>
                <Chip size="small" label={r.target || '-'} />
              </TableCell>
              <TableCell>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  {r.apiVersion ? `${r.apiVersion} / ` : ''}{r.kind || '-'} · {r.name || '-'}
                </Typography>
                {r.uid && (
                  <Typography variant="caption" color="text.secondary">uid: {r.uid}</Typography>
                )}
              </TableCell>
              <TableCell>
                <Typography variant="body2">{locLabel}</Typography>
              </TableCell>
              <TableCell>
                <Chip size="small" color={statusColor(r.status)} label={statusLabel(r.status)} />
                {r.error && (
                  <Typography variant="caption" color="error" sx={{ display: 'block', maxWidth: 260, wordBreak: 'break-all' }}>
                    {r.error}
                  </Typography>
                )}
              </TableCell>
              <TableCell>
                <Typography variant="caption" color="text.secondary">{actionLabel(r.action)}</Typography>
              </TableCell>
              <TableCell>
                {openLink && (
                  <MuiLink href={openLink} target="_blank" rel="noreferrer" underline="hover">
                    打开
                  </MuiLink>
                )}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function InstallationCard({
  installation,
  grafanaHosts,
}: {
  installation: IntegrationInstallation;
  grafanaHosts: GrafanaHost[];
}) {
  const [expanded, setExpanded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [revisions, setRevisions] = useState<IntegrationInstallationRevision[] | null>(null);

  const handleChange = async (_: unknown, isOpen: boolean) => {
    setExpanded(isOpen);
    if (isOpen && !revisions) {
      setLoading(true);
      try {
        const { data: res } = await integrationAPI.listInstallationRevisions(installation.id);
        setRevisions(res.data || []);
      } catch {
        // 该面板是次级信息，列表请求失败不弹 toast，让上层保持安静；
        // 设空数组让 UI 走"未找到变更历史"分支。
        setRevisions([]);
      } finally {
        setLoading(false);
      }
    }
  };

  const refs = revisions ? latestAppliedRefs(revisions) : [];

  return (
    <Accordion
      expanded={expanded}
      onChange={handleChange}
      sx={{ '&:before': { display: 'none' }, border: '1px solid', borderColor: 'divider' }}
      elevation={0}
    >
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Stack direction="row" spacing={1.5} alignItems="center" sx={{ flex: 1, flexWrap: 'wrap' }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            模版 {installation.template_id.slice(0, 8)} · {installation.template_version}
          </Typography>
          <Chip size="small" color={statusColor(installation.status)} label={statusLabel(installation.status)} />
          <Typography variant="caption" color="text.secondary">
            部件：{installation.installed_parts || '-'} · 安装人：{installation.installed_by || '-'}
          </Typography>
        </Stack>
      </AccordionSummary>
      <AccordionDetails>
        {loading ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CircularProgress size={16} />
            <Typography variant="caption">正在加载变更历史...</Typography>
          </Box>
        ) : revisions && revisions.length > 0 ? (
          <>
            <Typography variant="caption" color="text.secondary">
              最新一次 {actionLabel(revisions[0].action)} · {new Date(revisions[0].created_at).toLocaleString()} · {statusLabel(revisions[0].status)}
            </Typography>
            {revisions[0].error_message && (
              // uninstall_failed / failed / partial 的 ErrorMessage 透传后端真实错误（INS-3 修复后保留），
              // 给运维直接定位失败原因。
              <Alert severity={revisions[0].status === 'uninstall_failed' || revisions[0].status === 'failed' ? 'error' : 'warning'} sx={{ mt: 1 }}>
                {revisions[0].error_message}
              </Alert>
            )}
            <AppliedRefsTable refs={refs} grafanaHosts={grafanaHosts} />
          </>
        ) : (
          <Alert severity="warning">未找到变更历史记录。</Alert>
        )}
      </AccordionDetails>
    </Accordion>
  );
}

function InstallationList({
  installations,
  grafanaHosts,
  filterPart,
}: {
  installations: IntegrationInstallation[];
  grafanaHosts: GrafanaHost[];
  filterPart?: string;
}) {
  const filtered = filterPart
    ? installations.filter((i) => (i.installed_parts || '').includes(filterPart))
    : installations;
  if (filtered.length === 0) {
    return <EmptyState title="暂无记录" description="通过接入中心安装模版后会展示在此。" />;
  }
  return (
    <Stack spacing={1.5}>
      {filtered.map((i) => (
        <InstallationCard key={i.id} installation={i} grafanaHosts={grafanaHosts} />
      ))}
    </Stack>
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
  const [installations, setInstallations] = useState<IntegrationInstallation[]>([]);
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaHost[]>([]);
  const [scaleEvents, setScaleEvents] = useState<ScaleEvent[]>([]);
  const [scaleEventsLoading, setScaleEventsLoading] = useState(false);

  // 详情页一次性拉 4 个独立资源，统一通过同一个 AbortController 兜起：
  //   - 切换实例 / 卸载组件时立即取消在飞请求；
  //   - 单个失败用 extractApiError 给可读文案，不再吞错只显示通用提示。
  useEffect(() => {
    if (!instanceId) return;
    const ctl = makeAbortController();
    let alive = true;
    setLoading(true);
    setMetricsLoading(true);
    setScaleEventsLoading(true);

    (async () => {
      try {
        const { data: res } = await instanceAPI.get(instanceId, { signal: ctl.signal });
        if (alive) setInstance(res.data || null);
      } catch (err) {
        if (isAbortError(err) || !alive) return;
        enqueueSnackbar(extractApiError(err, '获取实例详情失败'), { variant: 'error' });
        navigate('/instances');
      } finally {
        if (alive) setLoading(false);
      }
    })();

    (async () => {
      try {
        const { data: res } = await instanceAPI.metrics(instanceId, { signal: ctl.signal });
        if (alive) setMetrics(res.data || null);
      } catch (err) {
        if (isAbortError(err) || !alive) return;
        setMetrics(null);
        enqueueSnackbar(extractApiError(err, '获取实例监控数据失败'), { variant: 'warning' });
      } finally {
        if (alive) setMetricsLoading(false);
      }
    })();

    (async () => {
      try {
        const { data: res } = await instanceAPI.scaleEvents(instanceId, { page: 1, page_size: 50 }, { signal: ctl.signal });
        if (alive) setScaleEvents(res.data?.items || []);
      } catch (err) {
        if (isAbortError(err) || !alive) return;
        // 伸缩历史为辅助信息，失败时静默置空（StatusChip 会显示"暂无伸缩事件"）。
        setScaleEvents([]);
      } finally {
        if (alive) setScaleEventsLoading(false);
      }
    })();

    (async () => {
      try {
        const { data: res } = await integrationAPI.listInstallations(
          { page: 1, page_size: 50, instance_id: instanceId },
          { signal: ctl.signal },
        );
        if (alive) setInstallations(res.data?.items || []);
      } catch (err) {
        if (isAbortError(err) || !alive) return;
        setInstallations([]);
      }
    })();

    return () => {
      alive = false;
      ctl.abort();
    };
  }, [instanceId, enqueueSnackbar, navigate]);

  useEffect(() => {
    const ctl = makeAbortController();
    let alive = true;
    (async () => {
      try {
        const { data: res } = await grafanaHostAPI.list({ page: 1, page_size: 100 }, { signal: ctl.signal });
        if (alive) setGrafanaHosts(res.data?.items || []);
      } catch (err) {
        if (isAbortError(err) || !alive) return;
        // grafana 主机列表用于 dashboard 链接渲染，缺失不阻塞主流程，静默降级。
        setGrafanaHosts([]);
      }
    })();
    return () => {
      alive = false;
      ctl.abort();
    };
  }, []);

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
              <>
                <Card sx={{ p: 2.5, mb: 2 }}>
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

                <Card sx={{ p: 2.5, mb: 2 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1.5 }}>服务地址</Typography>
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

                <Card sx={{ p: 2.5 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1.5 }}>资源使用</Typography>
                  {metricsLoading ? (
                    <LinearProgress />
                  ) : metrics ? (
                    <>
                      <ResourceBar label="CPU" used={Math.round(spec.cpu * (metrics.cpu_usage_percent || 0) / 100)} total={spec.cpu} unit="Core" />
                      <ResourceBar label="内存" used={Math.round(spec.memory * (metrics.memory_usage_percent || 0) / 100)} total={spec.memory} unit="GiB" />
                      <ResourceBar label="存储" used={Math.round(spec.storage * (metrics.disk_usage_percent || 0) / 100)} total={spec.storage} unit="GiB" />
                      {metrics.note && <Alert severity="info">{metrics.note}</Alert>}
                    </>
                  ) : (
                    <Alert severity="warning">暂无可用监控数据，请稍后重试。</Alert>
                  )}
                </Card>
              </>
            ),
          },
          {
            key: 'collect',
            label: '数据采集',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>接入中心</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  选择模版将 VMPodScrape / VMServiceScrape / VMAgent 等采集配置下发到本实例。
                </Typography>
                <Button
                  variant="contained"
                  startIcon={<ExtensionOutlinedIcon />}
                  onClick={() => navigate(`/integrations?instance_id=${instance.id}&instance_name=${encodeURIComponent(instance.instance_name)}`)}
                >
                  打开接入中心
                </Button>
                <Box sx={{ mt: 3 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>本实例已安装的采集模版</Typography>
                  <InstallationList installations={installations} grafanaHosts={grafanaHosts} filterPart="collector" />
                </Box>
              </Card>
            ),
          },
          {
            key: 'dashboard',
            label: 'Dashboard',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>本实例已安装的 Dashboard</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  通过接入中心安装模版时勾选 Dashboard 部件，会在此处展示。
                </Typography>
                <InstallationList installations={installations} grafanaHosts={grafanaHosts} filterPart="dashboard" />
              </Card>
            ),
          },
          {
            key: 'alert',
            label: '告警',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>本实例已下发的告警模版</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  M2 起，VMRule 会以 K8s CR 形式下发；N9E 规则为占位，未执行。
                </Typography>
                <InstallationList installations={installations} grafanaHosts={grafanaHosts} filterPart="vmrule" />
                <Box sx={{ mt: 2, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  <Button
                    variant="outlined"
                    startIcon={<NotificationsActiveOutlinedIcon />}
                    onClick={() => navigate(`/alerts?instance_id=${instance.id}&instance_name=${encodeURIComponent(instance.instance_name)}`)}
                  >
                    打开 N9E 告警引擎（占位）
                  </Button>
                </Box>
              </Card>
            ),
          },
          {
            key: 'scale',
            label: '伸缩历史',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>实例伸缩事件（最近 50 条）</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  每次水平 / 垂直 / 存储伸缩都会记录生效路径（CR 直 patch / helm upgrade / k8s 原生），
                  方便审计与诊断。
                </Typography>
                <ScaleEventList events={scaleEvents} loading={scaleEventsLoading} />
              </Card>
            ),
          },
        ]}
      />
    </Box>
  );
}
