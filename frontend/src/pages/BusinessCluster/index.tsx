import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import {
  businessClusterAPI,
  type BusinessCluster,
  type CollectConfigView,
  type CreateBusinessClusterRequest,
  type LogsCollectConfig,
  type MetricsCollectConfig,
} from '../../api/businessCluster';
import { logAPI, type LogInstance } from '../../api/logs';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { isPlatformAdmin } from '../../utils/membership';

function listToLines(items?: string[]): string {
  return (items || []).join('\n');
}

function linesToList(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

const agentStatusLabels: Record<string, { label: string; color: 'success' | 'warning' | 'error' | 'default' }> = {
  pending: { label: '待部署', color: 'default' },
  deploying: { label: '部署中', color: 'warning' },
  active: { label: '运行中', color: 'success' },
  failed: { label: '失败', color: 'error' },
  off: { label: '已停止', color: 'default' },
};

const logAgentStatusLabels: Record<string, { label: string; color: 'success' | 'warning' | 'error' | 'default' }> = {
  pending: { label: '未启用', color: 'default' },
  deploying: { label: '部署中', color: 'warning' },
  active: { label: '采集中', color: 'success' },
  failed: { label: '失败', color: 'error' },
  off: { label: '已停止', color: 'default' },
};

const defaultForm: CreateBusinessClusterRequest = {
  instance_id: '',
  name: '',
  display_name: '',
  kubeconfig: '',
  kubeconfig_path: '',
  labels: {},
};

export default function BusinessClusterPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  // 业务集群的写接口（接入/启停日志/删除）在后端为平台管理员专属。
  const isAdmin = isPlatformAdmin(user);
  const [searchParams] = useSearchParams();

  const tenantFilter = searchParams.get('workspace_id') || '';
  const instanceFilter = searchParams.get('instance_id') || '';

  const [clusters, setClusters] = useState<BusinessCluster[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; cluster?: BusinessCluster }>({ open: false });
  const [logsDialog, setLogsDialog] = useState<{ open: boolean; cluster?: BusinessCluster }>({ open: false });
  const [logInstances, setLogInstances] = useState<LogInstance[]>([]);
  const [selectedLogInstance, setSelectedLogInstance] = useState('');
  const [form, setForm] = useState<CreateBusinessClusterRequest>(defaultForm);
  const [saving, setSaving] = useState(false);

  const [collectDialog, setCollectDialog] = useState<{ open: boolean; cluster?: BusinessCluster }>({ open: false });
  const [collectTab, setCollectTab] = useState(0);
  const [collectLoading, setCollectLoading] = useState(false);
  const [metricsCfg, setMetricsCfg] = useState<MetricsCollectConfig>({});
  const [logsCfg, setLogsCfg] = useState<LogsCollectConfig>({});
  const [metricsNSInclude, setMetricsNSInclude] = useState('');
  const [metricsNSExclude, setMetricsNSExclude] = useState('');
  const [logsNSInclude, setLogsNSInclude] = useState('');
  const [logsNSExclude, setLogsNSExclude] = useState('');
  const [logsExcludePaths, setLogsExcludePaths] = useState('');

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await businessClusterAPI.list({
        page: page + 1,
        page_size: pageSize,
        search,
        workspace_id: tenantFilter || undefined,
        instance_id: instanceFilter || undefined,
      });
      setClusters(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取业务集群列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar, tenantFilter, instanceFilter, page, pageSize, search, statusFilter]);

  useEffect(() => {
    fetch();
  }, [fetch]);

  const handleSave = async () => {
    if (!form.name || !form.instance_id) {
      enqueueSnackbar('名称和实例 ID 为必填项', { variant: 'warning' });
      return;
    }
    if (!form.kubeconfig && !form.kubeconfig_path) {
      enqueueSnackbar('请至少提供 Kubeconfig 内容或文件路径', { variant: 'warning' });
      return;
    }
    setSaving(true);
    try {
      await businessClusterAPI.create(form);
      enqueueSnackbar('业务集群接入成功', { variant: 'success' });
      setDialogOpen(false);
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '接入失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleEnableLogs = async () => {
    if (!logsDialog.cluster || !selectedLogInstance) return;
    setSaving(true);
    try {
      await businessClusterAPI.enableLogs(logsDialog.cluster.id, { log_instance_id: selectedLogInstance });
      enqueueSnackbar('日志采集已启用（Vector Agent → Kafka）', { variant: 'success' });
      setLogsDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '启用日志采集失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const openLogsDialog = async (cluster: BusinessCluster) => {
    setLogsDialog({ open: true, cluster });
    setSelectedLogInstance('');
    try {
      const { data: res } = await logAPI.list({ page: 1, page_size: 100 });
      const items = res.data?.items || [];
      setLogInstances(items);
      if (items.length > 0) setSelectedLogInstance(items[0].id);
    } catch {
      setLogInstances([]);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.cluster) return;
    try {
      await businessClusterAPI.delete(deleteDialog.cluster.id);
      enqueueSnackbar('业务集群移除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '移除失败'), { variant: 'error' });
    }
  };

  const openCollectDialog = async (cluster: BusinessCluster) => {
    setCollectDialog({ open: true, cluster });
    setCollectTab(0);
    setCollectLoading(true);
    try {
      const { data: res } = await businessClusterAPI.getCollectConfig(cluster.id);
      const view: CollectConfigView = res.data;
      setMetricsCfg(view.metrics || {});
      setLogsCfg(view.logs || {});
      setMetricsNSInclude(listToLines(view.metrics?.namespace_include));
      setMetricsNSExclude(listToLines(view.metrics?.namespace_exclude));
      setLogsNSInclude(listToLines(view.logs?.namespace_include));
      setLogsNSExclude(listToLines(view.logs?.namespace_exclude));
      setLogsExcludePaths(listToLines(view.logs?.exclude_paths));
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取采集配置失败'), { variant: 'error' });
      setCollectDialog({ open: false });
    } finally {
      setCollectLoading(false);
    }
  };

  const handleSaveCollectConfig = async () => {
    if (!collectDialog.cluster) return;
    setSaving(true);
    try {
      await businessClusterAPI.updateCollectConfig(collectDialog.cluster.id, {
        metrics: {
          ...metricsCfg,
          namespace_include: linesToList(metricsNSInclude),
          namespace_exclude: linesToList(metricsNSExclude),
        },
        logs: {
          ...logsCfg,
          namespace_include: linesToList(logsNSInclude),
          namespace_exclude: linesToList(logsNSExclude),
          exclude_paths: linesToList(logsExcludePaths),
        },
      });
      enqueueSnackbar('采集配置已保存；已运行的 Agent 将按新配置重新下发', { variant: 'success' });
      setCollectDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '保存采集配置失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const parseLabels = (labelsJson: string): Record<string, string> => {
    try { return JSON.parse(labelsJson || '{}'); } catch { return {}; }
  };

  if (loading && clusters.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="业务集群管理"
        subtitle="管理工作空间接入的业务 Kubernetes 集群，VMAgent 采集指标，Vector Agent 采集日志（→ Kafka）"
        actionLabel={isAdmin ? '接入业务集群' : undefined}
        onAction={isAdmin ? () => {
          setForm({ ...defaultForm, instance_id: instanceFilter });
          setDialogOpen(true);
        } : undefined}
      />

      {!isAdmin && (
        <Alert severity="info" sx={{ mb: 2 }}>
          仅平台管理员可接入业务集群、修改采集配置。当前仅提供只读视图。
        </Alert>
      )}

      <FilterToolbar>
        <TextField
          placeholder="搜索集群名称..."
          size="small"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(0);
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'text.disabled' }} />
              </InputAdornment>
            ),
          }}
          sx={{ width: 280 }}
        />
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <InputLabel>Agent 状态</InputLabel>
          <Select
            value={statusFilter}
            label="Agent 状态"
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(0);
            }}
          >
            <MenuItem value="">全部</MenuItem>
            <MenuItem value="pending">待部署</MenuItem>
            <MenuItem value="deploying">部署中</MenuItem>
            <MenuItem value="active">运行中</MenuItem>
            <MenuItem value="failed">失败</MenuItem>
            <MenuItem value="off">已停止</MenuItem>
          </Select>
        </FormControl>
      </FilterToolbar>

      <DataTableCard
        pagination={total > 0 ? (
          <TablePagination
            component="div"
            count={total}
            page={page}
            onPageChange={(_, nextPage) => setPage(nextPage)}
            rowsPerPage={pageSize}
            onRowsPerPageChange={(e) => {
              setPageSize(parseInt(e.target.value, 10));
              setPage(0);
            }}
            rowsPerPageOptions={[10, 20, 50]}
            labelRowsPerPage="每页行数"
          />
        ) : null}
      >
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>显示名</TableCell>
                <TableCell>实例 ID</TableCell>
                <TableCell>指标 Agent</TableCell>
                <TableCell>日志 Agent</TableCell>
                <TableCell>标签</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {clusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8}>
                    <EmptyState
                      title="暂无业务集群"
                      description={isAdmin ? '点击右上角按钮接入第一个业务集群' : '当前工作空间下没有已接入的业务集群'}
                    />
                  </TableCell>
                </TableRow>
              ) : (
                clusters.map((c) => {
                  const labels = parseLabels(c.labels);
                  return (
                    <TableRow key={c.id}>
                      <TableCell sx={{ fontWeight: 500, fontFamily: 'monospace' }}>
                        {c.name}
                      </TableCell>
                      <TableCell sx={{ color: 'text.secondary' }}>{c.display_name || '-'}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}>
                        {c.instance_id.substring(0, 8)}...
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={agentStatusLabels[c.agent_status]?.label || c.agent_status}
                          color={agentStatusLabels[c.agent_status]?.color || 'default'}
                        />
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={logAgentStatusLabels[c.log_agent_status]?.label || c.log_agent_status || 'pending'}
                          color={logAgentStatusLabels[c.log_agent_status]?.color || 'default'}
                        />
                      </TableCell>
                      <TableCell>
                        {Object.keys(labels).length > 0
                          ? Object.entries(labels).map(([k, v]) => (
                              <Chip
                                key={k}
                                size="small"
                                label={`${k}: ${v}`}
                                variant="outlined"
                                sx={{ mr: 0.5, mb: 0.5 }}
                              />
                            ))
                          : '-'}
                      </TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                        {new Date(c.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="采集配置">
                          <IconButton size="small" onClick={() => openCollectDialog(c)} sx={{ mr: 0.5 }}>
                            <TuneOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        {isAdmin && (
                          <>
                            {c.log_agent_status !== 'active' && (
                              <Tooltip title="启用日志采集">
                                <Button size="small" onClick={() => openLogsDialog(c)} sx={{ mr: 1 }}>
                                  日志
                                </Button>
                              </Tooltip>
                            )}
                            {c.log_agent_status === 'active' && (
                              <Tooltip title="关闭日志采集">
                                <Button
                                  size="small"
                                  onClick={async () => {
                                    try {
                                      await businessClusterAPI.disableLogs(c.id);
                                      enqueueSnackbar('日志采集已关闭', { variant: 'success' });
                                      fetch();
                                    } catch (err) {
                                      enqueueSnackbar(extractApiError(err, '关闭日志采集失败'), { variant: 'error' });
                                    }
                                  }}
                                  sx={{ mr: 1 }}
                                >
                                  关日志
                                </Button>
                              </Tooltip>
                            )}
                            <Tooltip title="移除">
                              <IconButton
                                size="small"
                                color="error"
                                onClick={() => setDeleteDialog({ open: true, cluster: c })}
                              >
                                <DeleteOutlinedIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </DataTableCard>

      {/* 接入业务集群对话框 */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>接入业务集群</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            业务集群需要预先安装 VM Operator，系统将通过 VMAgent CR 自动下发采集配置。
          </Typography>
          <TextField
            fullWidth
            size="small"
            label="实例 ID (instance_id)"
            value={form.instance_id}
            onChange={(e) => setForm({ ...form, instance_id: e.target.value })}
            sx={{ mb: 2 }}
            required
            helperText="该业务集群关联的监控实例 UUID"
          />
          <TextField
            fullWidth
            size="small"
            label="集群名称 (name)"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            sx={{ mb: 2 }}
            required
            helperText="唯一标识，小写字母与数字"
          />
          <TextField
            fullWidth
            size="small"
            label="显示名 (display_name)"
            value={form.display_name}
            onChange={(e) => setForm({ ...form, display_name: e.target.value })}
            sx={{ mb: 2 }}
          />
          <TextField
            fullWidth
            size="small"
            label="Kubeconfig 文件路径（推荐）"
            value={form.kubeconfig_path}
            onChange={(e) => setForm({ ...form, kubeconfig_path: e.target.value })}
            sx={{ mb: 2 }}
            helperText="例如 /etc/opsconfig/business-kubeconfig.yaml"
          />
          <TextField
            fullWidth
            size="small"
            label="Kubeconfig 内容（二选一）"
            value={form.kubeconfig}
            onChange={(e) => setForm({ ...form, kubeconfig: e.target.value })}
            sx={{ mb: 2 }}
            multiline
            minRows={4}
            helperText="目前仅存档，优先使用文件路径"
          />
          <TextField
            fullWidth
            size="small"
            label="Labels (JSON)"
            value={form.labels ? JSON.stringify(form.labels, null, 2) : ''}
            onChange={(e) => {
              try { setForm({ ...form, labels: JSON.parse(e.target.value) }); } catch { /* 允许输入过程中暂时非 JSON */ }
            }}
            sx={{ mb: 2 }}
            multiline
            minRows={2}
            helperText='键值对，例如 {"env": "prod", "region": "cn-east"}'
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.name || !form.instance_id}>
            {saving ? '接入中...' : '接入'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* 启用日志采集 */}
      <Dialog open={logsDialog.open} onClose={() => setLogsDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>启用日志采集</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            将在业务集群部署 Vector Agent，日志写入 Zone Kafka（不直连存储）。
          </Typography>
          <FormControl fullWidth size="small">
            <InputLabel>日志实例</InputLabel>
            <Select
              value={selectedLogInstance}
              label="日志实例"
              onChange={(e) => setSelectedLogInstance(e.target.value)}
            >
              {logInstances.map((li) => (
                <MenuItem key={li.id} value={li.id}>
                  {li.instance_name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          {logInstances.length === 0 && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              请先创建日志实例并完成 Zone init-logs。
            </Alert>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setLogsDialog({ open: false })}>取消</Button>
          <Button
            variant="contained"
            onClick={handleEnableLogs}
            disabled={saving || !selectedLogInstance}
          >
            {saving ? '部署中...' : '启用'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* 采集配置 */}
      <Dialog open={collectDialog.open} onClose={() => !saving && setCollectDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>
          采集配置 — {collectDialog.cluster?.display_name || collectDialog.cluster?.name}
        </DialogTitle>
        <DialogContent sx={{ pt: '8px !important' }}>
          {collectLoading ? (
            <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>加载中...</Typography>
          ) : (
            <>
              <Alert severity="info" sx={{ mb: 2 }}>
                指标走 VMAgent，日志走 Vector。保存后会对已启用的 Agent 重新下发配置。
              </Alert>
              <Tabs value={collectTab} onChange={(_, v) => setCollectTab(v)} sx={{ mb: 2 }}>
                <Tab label="指标采集" />
                <Tab label="日志采集" />
              </Tabs>

              {collectTab === 0 && (
                <Box>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={metricsCfg.select_all_by_default !== false}
                        onChange={(e) => setMetricsCfg({ ...metricsCfg, select_all_by_default: e.target.checked })}
                        disabled={!isAdmin}
                      />
                    }
                    label="自动发现全部 scrape 对象（selectAllByDefault）"
                    sx={{ mb: 1.5, display: 'flex' }}
                  />
                  <TextField
                    fullWidth size="small" label="抓取间隔" value={metricsCfg.scrape_interval || ''}
                    onChange={(e) => setMetricsCfg({ ...metricsCfg, scrape_interval: e.target.value })}
                    helperText="例如 30s / 1m" sx={{ mb: 2 }} disabled={!isAdmin}
                  />
                  <TextField
                    fullWidth size="small" label="抓取超时" value={metricsCfg.scrape_timeout || ''}
                    onChange={(e) => setMetricsCfg({ ...metricsCfg, scrape_timeout: e.target.value })}
                    helperText="例如 10s" sx={{ mb: 2 }} disabled={!isAdmin}
                  />
                  <TextField
                    fullWidth size="small" multiline minRows={2}
                    label="命名空间白名单（每行一个，空=不限）"
                    value={metricsNSInclude}
                    onChange={(e) => setMetricsNSInclude(e.target.value)}
                    sx={{ mb: 2 }} disabled={!isAdmin}
                  />
                  <TextField
                    fullWidth size="small" multiline minRows={2}
                    label="命名空间黑名单（丢弃这些 namespace 标签）"
                    value={metricsNSExclude}
                    onChange={(e) => setMetricsNSExclude(e.target.value)}
                    disabled={!isAdmin}
                  />
                </Box>
              )}

              {collectTab === 1 && (
                <Box>
                  <TextField
                    fullWidth size="small" multiline minRows={2}
                    label="命名空间白名单（每行一个，空=不限）"
                    value={logsNSInclude}
                    onChange={(e) => setLogsNSInclude(e.target.value)}
                    sx={{ mb: 2 }} disabled={!isAdmin}
                  />
                  <TextField
                    fullWidth size="small" multiline minRows={2}
                    label="命名空间黑名单"
                    value={logsNSExclude}
                    onChange={(e) => setLogsNSExclude(e.target.value)}
                    helperText="默认排除 kube-system"
                    sx={{ mb: 2 }} disabled={!isAdmin}
                  />
                  <TextField
                    fullWidth size="small" multiline minRows={2}
                    label="排除路径 glob（每行一个）"
                    value={logsExcludePaths}
                    onChange={(e) => setLogsExcludePaths(e.target.value)}
                    helperText='例如 **/tmp/**'
                    disabled={!isAdmin}
                  />
                </Box>
              )}
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCollectDialog({ open: false })} disabled={saving}>关闭</Button>
          {isAdmin && (
            <Button variant="contained" onClick={handleSaveCollectConfig} disabled={saving || collectLoading}>
              {saving ? '保存中...' : '保存并下发'}
            </Button>
          )}
        </DialogActions>
      </Dialog>

      {/* 删除确认 */}
      <ConfirmDialog
        open={deleteDialog.open}
        title="移除业务集群"
        message={`确定要移除业务集群「${deleteDialog.cluster?.name}」吗？该操作将删除对应的 VMAgent CR 并停止采集。`}
        severity="error"
        confirmLabel="移除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
