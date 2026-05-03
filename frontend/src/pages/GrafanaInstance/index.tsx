import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import { instanceAPI } from '../../api/instance';
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { tenantAPI } from '../../api/tenant';
import { extractApiError } from '../../api';
import { ssoLoginToGrafana } from '../../api/grafanaSso';
import type { Instance, Tenant } from '../../types/api';
import { parseSpec } from '../../utils/instance';
import { useAuthStore } from '../../stores/useAuthStore';

const statusFilterItems = [
  { key: '', label: '全部状态' },
  { key: 'running', label: '运行中' },
  { key: 'creating', label: '创建中' },
  { key: 'stopped', label: '已停止' },
  { key: 'error', label: '异常' },
];

interface HostFormState {
  name: string;
  scope: 'platform' | 'tenant';
  tenant_id: string;
  url: string;
  admin_user: string;
  admin_password: string;
  admin_token: string;
}

const defaultHostForm: HostFormState = {
  name: '',
  scope: 'platform',
  tenant_id: '',
  url: '',
  admin_user: 'admin',
  admin_password: '',
  admin_token: '',
};

export default function GrafanaInstancePage() {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [tabIndex, setTabIndex] = useState(0);

  // ---- Instance state ----
  const [instances, setInstances] = useState<Instance[]>([]);
  const [instLoading, setInstLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaHost[]>([]);
  const [metricsInstances, setMetricsInstances] = useState<Instance[]>([]);

  const [createOpen, setCreateOpen] = useState(false);
  const [editDialog, setEditDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [editForm, setEditForm] = useState({ instance_name: '', grafana_host_id: '' });
  const [rebuildDialog, setRebuildDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [createForm, setCreateForm] = useState({ tenant_id: '', instance_name: '', metrics_instance_id: '', grafana_host_id: '' });
  const [saving, setSaving] = useState(false);

  // ---- Host state ----
  const [hosts, setHosts] = useState<GrafanaHost[]>([]);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [hostLoading, setHostLoading] = useState(true);
  const [hostDialogOpen, setHostDialogOpen] = useState(false);
  const [hostEditingId, setHostEditingId] = useState<string | null>(null);
  const [hostForm, setHostForm] = useState<HostFormState>(defaultHostForm);
  const [hostDeleteDialog, setHostDeleteDialog] = useState<{ open: boolean; host?: GrafanaHost }>({ open: false });

  // ---- Instance fetching ----
  const fetchInstances = useCallback(async () => {
    setInstLoading(true);
    try {
      const { data: res } = await instanceAPI.list({
        page: page + 1,
        page_size: pageSize,
        search,
        instance_type: 'visual',
        status: statusFilter || undefined,
      });
      setInstances(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取 Grafana 实例列表失败'), { variant: 'error' });
    } finally {
      setInstLoading(false);
    }
  }, [page, pageSize, search, statusFilter, enqueueSnackbar]);

  useEffect(() => { fetchInstances(); }, [fetchInstances]);

  const fetchGrafanaHosts = useCallback(async () => {
    try {
      const { data: res } = await grafanaHostAPI.list({ page: 1, page_size: 100 });
      setGrafanaHosts(res.data?.items || []);
    } catch { /* optional */ }
  }, []);

  useEffect(() => { fetchGrafanaHosts(); }, [fetchGrafanaHosts]);

  const fetchMetricsInstances = useCallback(async () => {
    try {
      const { data: res } = await instanceAPI.list({ page: 1, page_size: 200, instance_type: 'metrics', status: 'running' });
      setMetricsInstances(res.data?.items || []);
    } catch { /* optional */ }
  }, []);

  useEffect(() => { fetchMetricsInstances(); }, [fetchMetricsInstances]);

  const statusStats = useMemo(() => {
    return instances.reduce(
      (acc, item) => {
        acc.total += 1;
        if (item.status === 'running') acc.running += 1;
        if (item.status === 'creating') acc.creating += 1;
        if (item.status === 'error') acc.error += 1;
        return acc;
      },
      { total: 0, running: 0, creating: 0, error: 0 },
    );
  }, [instances]);

  // ---- Instance handlers ----
  const handleCreate = async () => {
    setSaving(true);
    try {
      const spec = JSON.stringify({
        cpu: 1, memory: 2, storage: 10, retention: 30,
        metrics_instance_id: createForm.metrics_instance_id || undefined,
      });
      await instanceAPI.create({
        tenant_id: createForm.tenant_id,
        instance_name: createForm.instance_name,
        instance_type: 'visual',
        template_type: 'dedicated_single',
        spec,
        grafana_host_id: createForm.grafana_host_id || undefined,
      });
      enqueueSnackbar('Grafana 实例创建成功', { variant: 'success' });
      setCreateOpen(false);
      fetchInstances();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = async () => {
    if (!editDialog.instance) return;
    setSaving(true);
    try {
      await instanceAPI.update(editDialog.instance.id, {
        instance_name: editForm.instance_name,
        grafana_host_id: editForm.grafana_host_id || undefined,
        spec: editDialog.instance.spec,
      });
      enqueueSnackbar('Grafana 实例更新成功', { variant: 'success' });
      setEditDialog({ open: false });
      fetchInstances();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.instance) return;
    try {
      await instanceAPI.delete(deleteDialog.instance.id);
      enqueueSnackbar('Grafana 实例删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchInstances();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const handleRebuild = async () => {
    if (!rebuildDialog.instance) return;
    setSaving(true);
    try {
      await instanceAPI.rebuild(rebuildDialog.instance.id);
      enqueueSnackbar('重建请求已提交', { variant: 'success' });
      setRebuildDialog({ open: false });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '重建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleUpgrade = async (inst: Instance) => {
    try {
      await instanceAPI.upgrade(inst.id);
      enqueueSnackbar('升级请求已提交', { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '升级失败'), { variant: 'error' });
    }
  };

  // ---- Host fetching ----
  const fetchHosts = useCallback(async () => {
    setHostLoading(true);
    try {
      const [hostsRes, tenantsRes] = await Promise.all([
        grafanaHostAPI.list({ page: 1, page_size: 100 }),
        tenantAPI.list({ page: 1, page_size: 100 }).catch(() => ({ data: { data: { items: [] } } })),
      ]);
      setHosts(hostsRes.data.data?.items || []);
      setTenants(tenantsRes.data.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取纳管实例列表失败'), { variant: 'error' });
    } finally {
      setHostLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => { fetchHosts(); }, [fetchHosts]);

  const tenantNameById = tenants.reduce<Record<string, string>>((acc, t) => { acc[t.id] = t.tenant_name; return acc; }, {});

  // ---- Host handlers ----
  const openHostCreate = () => {
    setHostEditingId(null);
    setHostForm(defaultHostForm);
    setHostDialogOpen(true);
  };

  const openHostEdit = (h: GrafanaHost) => {
    setHostEditingId(h.id);
    setHostForm({ name: h.name, scope: h.scope, tenant_id: h.tenant_id || '', url: h.url, admin_user: h.admin_user || 'admin', admin_password: '', admin_token: '' });
    setHostDialogOpen(true);
  };

  const handleHostSave = async () => {
    if (!hostForm.name || !hostForm.url) { enqueueSnackbar('名称和 URL 必填', { variant: 'warning' }); return; }
    if (hostForm.scope === 'tenant' && !hostForm.tenant_id) { enqueueSnackbar('租户级实例必须选择所属租户', { variant: 'warning' }); return; }
    setSaving(true);
    try {
      if (hostEditingId) {
        await grafanaHostAPI.update(hostEditingId, {
          name: hostForm.name,
          url: hostForm.url,
          admin_user: hostForm.admin_user,
          admin_password: hostForm.admin_password || undefined,
          admin_token: hostForm.admin_token || undefined,
        });
        enqueueSnackbar('纳管实例更新成功', { variant: 'success' });
      } else {
        await grafanaHostAPI.create({
          name: hostForm.name,
          scope: hostForm.scope,
          tenant_id: hostForm.scope === 'tenant' ? hostForm.tenant_id : undefined,
          url: hostForm.url,
          admin_user: hostForm.admin_user,
          admin_password: hostForm.admin_password || undefined,
          admin_token: hostForm.admin_token || undefined,
        });
        enqueueSnackbar('纳管实例登记成功', { variant: 'success' });
      }
      setHostDialogOpen(false);
      fetchHosts();
      fetchGrafanaHosts(); // refresh instance tab's host list too
    } catch (err) {
      enqueueSnackbar(extractApiError(err, hostEditingId ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleHostDelete = async () => {
    if (!hostDeleteDialog.host) return;
    try {
      await grafanaHostAPI.delete(hostDeleteDialog.host.id);
      enqueueSnackbar('纳管实例删除成功', { variant: 'success' });
      setHostDeleteDialog({ open: false });
      fetchHosts();
      fetchGrafanaHosts();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const loading = (tabIndex === 0 && instLoading && instances.length === 0) || (tabIndex === 1 && hostLoading && hosts.length === 0);
  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="Grafana 实例"
        subtitle="管理平台实例与纳管外部 Grafana 实例"
        actionLabel={tabIndex === 0 ? '创建实例' : isAdmin ? '登记实例' : undefined}
        onAction={tabIndex === 0 ? () => setCreateOpen(true) : isAdmin ? openHostCreate : undefined}
      />
      <Card sx={{ mb: 2 }}>
        <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} sx={{ px: 2, pt: 1 }}>
          <Tab label="平台实例" />
          <Tab label="纳管实例" />
        </Tabs>
        <Divider />

        {/* ===== Tab 0: 平台实例 ===== */}
        {tabIndex === 0 && (
          <Box>
            <Box sx={{ p: 2 }}>
              <Typography variant="subtitle2" sx={{ mb: 1.5 }}>平台实例概览（当前页）</Typography>
              <Grid container spacing={1.5}>
                <Grid size={{ xs: 6, md: 3 }}>
                  <Card variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary">实例总数</Typography>
                    <Typography variant="h6">{statusStats.total}</Typography>
                  </Card>
                </Grid>
                <Grid size={{ xs: 6, md: 3 }}>
                  <Card variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary">运行中</Typography>
                    <Typography variant="h6" color="success.main">{statusStats.running}</Typography>
                  </Card>
                </Grid>
                <Grid size={{ xs: 6, md: 3 }}>
                  <Card variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary">创建中</Typography>
                    <Typography variant="h6" color="warning.main">{statusStats.creating}</Typography>
                  </Card>
                </Grid>
                <Grid size={{ xs: 6, md: 3 }}>
                  <Card variant="outlined" sx={{ p: 1.5 }}>
                    <Typography variant="caption" color="text.secondary">异常</Typography>
                    <Typography variant="h6" color="error.main">{statusStats.error}</Typography>
                  </Card>
                </Grid>
              </Grid>
            </Box>

            <FilterToolbar>
              <TextField
                placeholder="搜索实例名称..."
                size="small"
                value={search}
                onChange={(e) => { setSearch(e.target.value); setPage(0); }}
                InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
                sx={{ width: 280 }}
              />
              <FormControl size="small" sx={{ minWidth: 140 }}>
                <InputLabel>状态</InputLabel>
                <Select value={statusFilter} label="状态" onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}>
                  {statusFilterItems.map((item) => <MenuItem key={item.key || 'all'} value={item.key}>{item.label}</MenuItem>)}
                </Select>
              </FormControl>
            </FilterToolbar>

            <DataTableCard
              pagination={total > 0 ? (
                <TablePagination component="div" count={total} page={page} onPageChange={(_, np) => setPage(np)}
                  rowsPerPage={pageSize} onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }}
                  rowsPerPageOptions={[10, 20, 50]} labelRowsPerPage="每页行数" />
              ) : null}
            >
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>实例名称</TableCell>
                      <TableCell>类型</TableCell>
                      <TableCell>规格</TableCell>
                      <TableCell>命名空间</TableCell>
                      <TableCell>状态</TableCell>
                      <TableCell>创建时间</TableCell>
                      <TableCell align="right">操作</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {instances.length === 0 ? (
                      <TableRow><TableCell colSpan={7}><EmptyState title="暂无 Grafana 实例" description="点击右上角按钮创建第一个 Grafana 实例" /></TableCell></TableRow>
                    ) : instances.map((inst) => {
                      const spec = parseSpec(inst.spec);
                      return (
                        <TableRow key={inst.id}>
                          <TableCell sx={{ fontWeight: 500 }}>{inst.instance_name}</TableCell>
                          <TableCell><Chip label="Grafana" size="small" color="success" variant="outlined" /></TableCell>
                          <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>{spec.cpu}C / {spec.memory}G / {spec.storage}Gi</TableCell>
                          <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{inst.namespace || '-'}</TableCell>
                          <TableCell><StatusChip status={inst.status} /></TableCell>
                          <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(inst.created_at).toLocaleDateString()}</TableCell>
                          <TableCell align="right">
                            <Tooltip title="详情"><IconButton size="small" onClick={() => navigate(`/instances/${inst.id}`)}><InfoOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                            <Tooltip title="登录"><IconButton size="small" color="primary" onClick={() => {
                              ssoLoginToGrafana(instanceAPI.login(inst.id)).catch((err) =>
                                enqueueSnackbar(extractApiError(err, '获取登录信息失败'), { variant: 'error' })
                              );
                            }}><LoginOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                            <Tooltip title="编辑"><IconButton size="small" onClick={() => {
                              setEditForm({ instance_name: inst.instance_name, grafana_host_id: inst.grafana_host_id || '' });
                              setEditDialog({ open: true, instance: inst });
                            }}><EditOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                            <Tooltip title="重建">
                              <span><IconButton size="small" disabled={inst.status !== 'running' && inst.status !== 'failed'} onClick={() => setRebuildDialog({ open: true, instance: inst })}><RefreshOutlinedIcon fontSize="small" /></IconButton></span>
                            </Tooltip>
                            <Tooltip title="升级"><IconButton size="small" color="primary" disabled={inst.status !== 'running'} onClick={() => handleUpgrade(inst)}><RefreshOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                            <Tooltip title="删除"><IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, instance: inst })}><DeleteOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </TableContainer>
            </DataTableCard>
          </Box>
        )}

        {/* ===== Tab 1: 纳管实例 ===== */}
        {tabIndex === 1 && (
          <Box>
            {!isAdmin && (
              <Alert severity="info" sx={{ mx: 2, mt: 2 }}>仅管理员可登记/编辑/删除纳管实例。当前仅提供只读视图。</Alert>
            )}
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>名称</TableCell>
                    <TableCell>范围</TableCell>
                    <TableCell>所属租户</TableCell>
                    <TableCell>地址</TableCell>
                    <TableCell>管理员</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell align="right">操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {hosts.length === 0 ? (
                    <TableRow><TableCell colSpan={7}><EmptyState title="暂无纳管实例" description="点击右上角按钮登记外部 Grafana 实例" /></TableCell></TableRow>
                  ) : hosts.map((h) => (
                    <TableRow key={h.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{h.name}</TableCell>
                      <TableCell><Chip size="small" label={h.scope === 'platform' ? '平台' : '租户'} color={h.scope === 'platform' ? 'primary' : 'secondary'} variant="outlined" /></TableCell>
                      <TableCell sx={{ color: 'text.secondary' }}>{h.tenant_id ? (tenantNameById[h.tenant_id] || h.tenant_id.slice(0, 8)) : '-'}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                        <Typography component="a" href={h.url} target="_blank" rel="noopener noreferrer" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, color: 'primary.main', textDecoration: 'none', fontSize: '0.8125rem' }}>
                          {h.url}<OpenInNewIcon sx={{ fontSize: 14 }} />
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem' }}>{h.admin_user || '-'}</TableCell>
                      <TableCell><StatusChip status={h.status || 'active'} /></TableCell>
                      <TableCell align="right">
                        <Tooltip title="登录"><IconButton size="small" color="primary" onClick={() => {
                          ssoLoginToGrafana(grafanaHostAPI.login(h.id)).catch((err) =>
                            enqueueSnackbar(extractApiError(err, '获取登录信息失败'), { variant: 'error' })
                          );
                        }}><LoginOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                        {isAdmin && (
                          <>
                            <Tooltip title="编辑"><IconButton size="small" onClick={() => openHostEdit(h)}><EditOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                            <Tooltip title="删除"><IconButton size="small" color="error" onClick={() => setHostDeleteDialog({ open: true, host: h })}><DeleteOutlinedIcon fontSize="small" /></IconButton></Tooltip>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}
      </Card>

      {/* ===== Instance Create Dialog ===== */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>创建 Grafana 实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="租户 ID" value={createForm.tenant_id} onChange={(e) => setCreateForm({ ...createForm, tenant_id: e.target.value })} sx={{ mb: 2.5 }} required helperText="关联租户的 UUID" />
          <TextField fullWidth label="实例名称" value={createForm.instance_name} onChange={(e) => setCreateForm({ ...createForm, instance_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>关联 VictoriaMetrics 实例</InputLabel>
            <Select value={createForm.metrics_instance_id} label="关联 VictoriaMetrics 实例" onChange={(e) => setCreateForm({ ...createForm, metrics_instance_id: e.target.value })}>
              <MenuItem value="">暂不关联</MenuItem>
              {metricsInstances.map((m) => (
                <MenuItem key={m.id} value={m.id}>{m.instance_name}<Chip size="small" label={m.namespace || m.id.slice(0, 8)} variant="outlined" sx={{ ml: 1, height: 18, fontSize: '0.65rem' }} /></MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>部署 Grafana 主机（可选）</InputLabel>
            <Select value={createForm.grafana_host_id} label="部署 Grafana 主机（可选）" onChange={(e) => setCreateForm({ ...createForm, grafana_host_id: e.target.value })}>
              <MenuItem value="">平台默认</MenuItem>
              {grafanaHosts.map((h) => (
                <MenuItem key={h.id} value={h.id}>{h.name}<Chip size="small" label={h.scope === 'platform' ? '平台' : '租户'} variant="outlined" sx={{ ml: 1, height: 18, fontSize: '0.65rem' }} /></MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !createForm.tenant_id || !createForm.instance_name}>{saving ? '创建中...' : '创建'}</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Instance Edit Dialog ===== */}
      <Dialog open={editDialog.open} onClose={() => setEditDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>编辑 Grafana 实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="实例名称" value={editForm.instance_name} onChange={(e) => setEditForm({ ...editForm, instance_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>部署 Grafana 主机（可选）</InputLabel>
            <Select value={editForm.grafana_host_id} label="部署 Grafana 主机（可选）" onChange={(e) => setEditForm({ ...editForm, grafana_host_id: e.target.value })}>
              <MenuItem value="">平台默认</MenuItem>
              {grafanaHosts.map((h) => <MenuItem key={h.id} value={h.id}>{h.name}<Chip size="small" label={h.scope === 'platform' ? '平台' : '租户'} variant="outlined" sx={{ ml: 1, height: 18, fontSize: '0.65rem' }} /></MenuItem>)}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setEditDialog({ open: false })}>取消</Button>
          <Button variant="contained" onClick={handleEdit} disabled={saving || !editForm.instance_name}>{saving ? '保存中...' : '保存'}</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Instance Delete ===== */}
      <ConfirmDialog open={deleteDialog.open} title="删除 Grafana 实例" message={`确定要删除 Grafana 实例「${deleteDialog.instance?.instance_name}」吗？关联的 Helm Release 也将被卸载。`} severity="error" confirmLabel="删除" onConfirm={handleDelete} onCancel={() => setDeleteDialog({ open: false })} />

      {/* ===== Rebuild ===== */}
      <ConfirmDialog open={rebuildDialog.open} title="重建 Grafana 实例" message={`确定要重建 Grafana 实例「${rebuildDialog.instance?.instance_name}」吗？将重新部署 Helm Release。`} severity="warning" confirmLabel="重建" onConfirm={handleRebuild} onCancel={() => setRebuildDialog({ open: false })} />

      {/* ===== Host Dialog ===== */}
      <Dialog open={hostDialogOpen} onClose={() => setHostDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{hostEditingId ? '编辑纳管实例' : '登记纳管实例'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="名称" value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }} disabled={!!hostEditingId}>
            <InputLabel>范围</InputLabel>
            <Select label="范围" value={hostForm.scope} onChange={(e) => setHostForm({ ...hostForm, scope: e.target.value as 'platform' | 'tenant' })}>
              <MenuItem value="platform">平台共享</MenuItem>
              <MenuItem value="tenant">租户专属</MenuItem>
            </Select>
          </FormControl>
          {hostForm.scope === 'tenant' && (
            <FormControl fullWidth size="small" sx={{ mb: 2 }} disabled={!!hostEditingId}>
              <InputLabel>所属租户</InputLabel>
              <Select label="所属租户" value={hostForm.tenant_id} onChange={(e) => setHostForm({ ...hostForm, tenant_id: e.target.value })}>
                <MenuItem value="">请选择</MenuItem>
                {tenants.map((t) => <MenuItem key={t.id} value={t.id}>{t.tenant_name}</MenuItem>)}
              </Select>
            </FormControl>
          )}
          <TextField fullWidth size="small" label="URL" value={hostForm.url} onChange={(e) => setHostForm({ ...hostForm, url: e.target.value })} sx={{ mb: 2 }} required placeholder="http://grafana.monitoring.svc.cluster.local:3000" />
          <TextField fullWidth size="small" label="管理员账号" value={hostForm.admin_user} onChange={(e) => setHostForm({ ...hostForm, admin_user: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth size="small" label={hostEditingId ? '管理员密码（留空保持原值）' : '管理员密码'} value={hostForm.admin_password} onChange={(e) => setHostForm({ ...hostForm, admin_password: e.target.value })} sx={{ mb: 2 }} type="password" helperText="Grafana Admin 登陆密码，用于一键 SSO 登录" />
          <TextField fullWidth size="small" label={hostEditingId ? 'API Token（留空保持原值）' : 'API Token'} value={hostForm.admin_token} onChange={(e) => setHostForm({ ...hostForm, admin_token: e.target.value })} sx={{ mb: 1 }} type="password" helperText="建议使用 Grafana Service Account Token；数据库中将加密存储" />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setHostDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleHostSave} disabled={saving || !hostForm.name || !hostForm.url}>{saving ? '保存中...' : hostEditingId ? '更新' : '登记'}</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Host Delete ===== */}
      <ConfirmDialog open={hostDeleteDialog.open} title="删除纳管实例" message={`确定要删除纳管实例「${hostDeleteDialog.host?.name}」吗？关联的模板安装将回退到平台默认 Grafana。`} severity="error" confirmLabel="删除" onConfirm={handleHostDelete} onCancel={() => setHostDeleteDialog({ open: false })} />
    </Box>
  );
}
