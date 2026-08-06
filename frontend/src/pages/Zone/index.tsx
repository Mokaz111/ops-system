import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
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
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import CloudUploadOutlinedIcon from '@mui/icons-material/CloudUploadOutlined';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import { zoneAPI, type Zone } from '../../api/zone';
import type { PreflightCheck, ZoneComponent } from '../../types/api';
import { clusterAPI, type Cluster } from '../../api/cluster';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';

interface FormState {
  slug: string;
  display_name: string;
  description: string;
  cluster_id: string;
  endpoint: string;
  labels: string;
  max_instances: string;
  max_storage: string;
  status: string;
}

const defaultForm: FormState = {
  slug: '', display_name: '', description: '', cluster_id: '',
  endpoint: '', labels: '', max_instances: '', max_storage: '', status: 'active',
};

const defaultHelmValues = JSON.stringify({
  vmauth: { enabled: true },
  grafana: { enabled: true, persistence: { size: '10Gi' } },
  vmstorage: { persistentVolume: { size: '100Gi' } },
}, null, 2);

export default function ZonePage() {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const [zones, setZones] = useState<Zone[]>([]);
  const [loading, setLoading] = useState(true);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; zone?: Zone }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(defaultForm);
  const [saving, setSaving] = useState(false);

  // InitShared 配置弹窗
  const [initDialog, setInitDialog] = useState<{ open: boolean; zone?: Zone }>({ open: false });
  const [initForm, setInitForm] = useState({ namespace: '', release_name: 'vm-shared-stack', helm_values: defaultHelmValues });
  const [initLoading, setInitLoading] = useState(false);
  const [initPlan, setInitPlan] = useState<object | null>(null);
  const [initConfirmOpen, setInitConfirmOpen] = useState(false);
  const [preflightResult, setPreflightResult] = useState<PreflightCheck | null>(null);

  const [componentsDialog, setComponentsDialog] = useState<{ open: boolean; zone?: Zone }>({ open: false });
  const [components, setComponents] = useState<ZoneComponent[]>([]);
  const [componentsLoading, setComponentsLoading] = useState(false);

  const clusterNameById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const c of clusters) map[c.id] = c.display_name || c.name;
    return map;
  }, [clusters]);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await zoneAPI.list({ page: page + 1, page_size: pageSize, search, status: statusFilter || undefined });
      setZones(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取可用区列表失败'), { variant: 'error' });
    } finally { setLoading(false); }
  }, [enqueueSnackbar, page, pageSize, search, statusFilter]);

  useEffect(() => { fetch(); }, [fetch]);

  useEffect(() => {
    (async () => {
      try { const { data: res } = await clusterAPI.list({ page: 1, page_size: 100 }); setClusters(res.data?.items || []); } catch { /* ok */ }
    })();
  }, []);

  const openCreate = () => { setEditingId(null); setForm(defaultForm); setDialogOpen(true); };

  const openEdit = (z: Zone) => {
    setEditingId(z.id);
    let labelsStr = '';
    let cap: { max_instances?: number; max_storage?: string } = {};
    try { labelsStr = JSON.stringify(JSON.parse(z.labels || '{}'), null, 2); } catch { labelsStr = z.labels || ''; }
    try { cap = JSON.parse(z.capacity || '{}'); } catch { cap = {}; }
    setForm({ slug: z.slug, display_name: z.display_name || '', description: z.description || '', cluster_id: z.cluster_id || '', endpoint: z.endpoint || '', labels: labelsStr, max_instances: cap.max_instances?.toString() || '', max_storage: cap.max_storage || '', status: z.status || 'active' });
    setDialogOpen(true);
  };

  const openInitDialog = (z: Zone) => {
    setInitDialog({ open: true, zone: z });
    setInitForm({ namespace: `monitoring-${z.slug}`, release_name: 'vm-shared-stack', helm_values: defaultHelmValues });
    setInitPlan(null);
    setPreflightResult(null);
  };

  const openComponentsDialog = async (z: Zone) => {
    setComponentsDialog({ open: true, zone: z });
    setComponentsLoading(true);
    try {
      const { data: res } = await zoneAPI.getComponents(z.id);
      setComponents(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取组件状态失败'), { variant: 'error' });
      setComponents([]);
    } finally {
      setComponentsLoading(false);
    }
  };

  const runPreflight = async () => {
    if (!initDialog.zone) return;
    setInitLoading(true);
    try {
      let values: Record<string, unknown> | undefined;
      try { values = JSON.parse(initForm.helm_values); } catch { enqueueSnackbar('Helm Values JSON 格式错误', { variant: 'warning' }); setInitLoading(false); return; }
      const { data: res } = await zoneAPI.preflight(initDialog.zone.id, {
        dry_run: true,
        namespace: initForm.namespace,
        release_name: initForm.release_name,
        values,
      });
      setPreflightResult(res.data);
      if (res.data?.ok) {
        enqueueSnackbar('预检通过', { variant: 'success' });
      } else {
        enqueueSnackbar(`预检发现 ${res.data?.issues?.length || 0} 项问题`, { variant: 'warning' });
      }
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '预检失败'), { variant: 'error' });
    } finally { setInitLoading(false); }
  };

  const handleInitDryRun = async () => {
    if (!initDialog.zone) return;
    setInitLoading(true);
    try {
      let values: Record<string, unknown> | undefined;
      try { values = JSON.parse(initForm.helm_values); } catch { enqueueSnackbar('Helm Values JSON 格式错误', { variant: 'warning' }); setInitLoading(false); return; }
      const { data: res } = await zoneAPI.initShared(initDialog.zone.id, {
        dry_run: true,
        namespace: initForm.namespace,
        release_name: initForm.release_name,
        values,
      } as any);
      setInitPlan(res.data || res);
      enqueueSnackbar('Dry-run 成功，已生成部署计划', { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, 'Dry-run 失败'), { variant: 'error' });
    } finally { setInitLoading(false); }
  };

  const handleInitApply = async () => {
    if (!initDialog.zone) return;
    if (!preflightResult?.ok) {
      enqueueSnackbar('请先执行预检并确保通过后再初始化', { variant: 'warning' });
      return;
    }
    setInitLoading(true);
    try {
      let values: Record<string, unknown> | undefined;
      try { values = JSON.parse(initForm.helm_values); } catch { enqueueSnackbar('Helm Values JSON 格式错误', { variant: 'warning' }); setInitLoading(false); return; }
      await zoneAPI.initShared(initDialog.zone.id, {
        dry_run: false,
        namespace: initForm.namespace,
        release_name: initForm.release_name,
        values,
      } as any);
      enqueueSnackbar(`共享 VM 集群初始化已提交（${initDialog.zone.display_name}）`, { variant: 'success' });
      setInitConfirmOpen(false);
      setInitDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '初始化提交失败'), { variant: 'error' });
    } finally { setInitLoading(false); }
  };

  const handleSave = async () => {
    if (!form.slug || !form.display_name || !form.cluster_id) { enqueueSnackbar('Slug、显示名和集群 ID 为必填项', { variant: 'warning' }); return; }
    setSaving(true);
    try {
      let labels: Record<string, string> | undefined;
      let capacity: { max_instances: number; max_storage?: string } | undefined;
      if (form.labels.trim()) { try { labels = JSON.parse(form.labels); } catch { enqueueSnackbar('Labels 格式错误', { variant: 'warning' }); setSaving(false); return; } }
      if (form.max_instances) { capacity = { max_instances: parseInt(form.max_instances, 10) }; if (form.max_storage) capacity.max_storage = form.max_storage; }
      if (editingId) {
        await zoneAPI.update(editingId, { display_name: form.display_name, description: form.description || undefined, endpoint: form.endpoint || undefined, labels, capacity, status: form.status });
        enqueueSnackbar('可用区更新成功', { variant: 'success' });
      } else {
        await zoneAPI.create({ slug: form.slug, display_name: form.display_name, description: form.description || undefined, cluster_id: form.cluster_id, endpoint: form.endpoint || undefined, labels, capacity });
        enqueueSnackbar('可用区创建成功', { variant: 'success' });
      }
      setDialogOpen(false);
      fetch();
    } catch (err) { enqueueSnackbar(extractApiError(err, editingId ? '更新失败' : '创建失败'), { variant: 'error' }); }
    finally { setSaving(false); }
  };

  const handleDelete = async () => {
    if (!deleteDialog.zone) return;
    try { await zoneAPI.delete(deleteDialog.zone.id); enqueueSnackbar('可用区删除成功', { variant: 'success' }); setDeleteDialog({ open: false }); fetch(); }
    catch (err) { enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' }); }
  };

  const formatCapacityDisplay = (z: Zone) => {
    try {
      const cap = JSON.parse(z.capacity || '{}');
      if (!cap.max_instances && !cap.max_storage) return '-';
      const parts: string[] = [];
      if (cap.max_instances) parts.push(`最多 ${cap.max_instances} 实例`);
      if (cap.max_storage) parts.push(`存储 ${cap.max_storage}`);
      return parts.join(' / ');
    } catch { return '-'; }
  };

  if (loading && zones.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="可用区管理" subtitle="每个 Zone 绑定一个可观测集群。创建后初始化共享 VMCluster（含 Grafana + VMAuth）" actionLabel={isAdmin ? '创建可用区' : undefined} onAction={isAdmin ? openCreate : undefined} />

      {!isAdmin && <Alert severity="info" sx={{ mb: 2 }}>仅管理员可操作。当前仅提供只读视图。</Alert>}

      <FilterToolbar>
        <TextField placeholder="搜索 Zone 名称或 Slug..." size="small" value={search} onChange={(e) => { setSearch(e.target.value); setPage(0); }}
          InputProps={{ startAdornment: (<InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment>) }} sx={{ width: 280 }} />
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <InputLabel>状态</InputLabel>
          <Select value={statusFilter} label="状态" onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}>
            <MenuItem value="">全部</MenuItem>
            <MenuItem value="active">active</MenuItem><MenuItem value="creating">creating</MenuItem>
            <MenuItem value="degraded">degraded</MenuItem><MenuItem value="offline">offline</MenuItem><MenuItem value="failed">failed</MenuItem>
          </Select>
        </FormControl>
      </FilterToolbar>

      <DataTableCard pagination={total > 0 ? (
        <TablePagination component="div" count={total} page={page} onPageChange={(_, np) => setPage(np)} rowsPerPage={pageSize} onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }} rowsPerPageOptions={[10, 20, 50]} labelRowsPerPage="每页行数" />
      ) : null}>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell><TableCell>Slug</TableCell><TableCell>可观测集群</TableCell>
                <TableCell>接入点</TableCell><TableCell>容量</TableCell><TableCell>状态</TableCell><TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {zones.length === 0 ? (
                <TableRow><TableCell colSpan={7}><EmptyState title="暂无可用区" description="点击右上角按钮创建第一个可用区" /></TableCell></TableRow>
              ) : zones.map((z) => (
                <TableRow key={z.id}>
                  <TableCell sx={{ fontWeight: 500 }}>{z.display_name}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>{z.slug}</TableCell>
                  <TableCell sx={{ color: 'text.secondary' }}>{z.cluster_id && clusterNameById[z.cluster_id] ? clusterNameById[z.cluster_id] : z.cluster_id ? z.cluster_id.substring(0, 8) + '...' : '-'}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>{z.endpoint || '-'}</TableCell>
                  <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{formatCapacityDisplay(z)}</TableCell>
                  <TableCell><StatusChip status={z.status || 'active'} /></TableCell>
                  <TableCell align="right">
                    {isAdmin && (<>
                      <Tooltip title="在此可用区创建实例">
                        <IconButton size="small" onClick={() => navigate(`/instances/create?zone_id=${z.id}`)} color="primary"><AddCircleOutlineIcon fontSize="small" /></IconButton>
                      </Tooltip>
                      <Tooltip title="组件状态">
                        <IconButton size="small" onClick={() => openComponentsDialog(z)}><VisibilityOutlinedIcon fontSize="small" /></IconButton>
                      </Tooltip>
                      <Tooltip title="初始化共享 VM 集群">
                        <IconButton size="small" onClick={() => openInitDialog(z)}><CloudUploadOutlinedIcon fontSize="small" /></IconButton>
                      </Tooltip>
                      <Tooltip title="编辑">
                        <IconButton size="small" onClick={() => openEdit(z)}><EditOutlinedIcon fontSize="small" /></IconButton>
                      </Tooltip>
                      <Tooltip title="删除">
                        <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, zone: z })}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                      </Tooltip>
                    </>)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </DataTableCard>

      {/* Zone 创建/编辑弹窗 */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingId ? '编辑可用区' : '创建可用区'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="唯一标识 (slug)" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} sx={{ mb: 2 }} required disabled={!!editingId} helperText="小写字母与数字，创建后不可修改" />
          <TextField fullWidth size="small" label="显示名" value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} sx={{ mb: 2 }} required />
          <TextField fullWidth size="small" label="描述" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} sx={{ mb: 2 }} multiline minRows={2} />
          <FormControl fullWidth size="small" sx={{ mb: 2 }} required disabled={!!editingId}>
            <InputLabel>可观测集群</InputLabel>
            <Select value={form.cluster_id} label="可观测集群" onChange={(e) => setForm({ ...form, cluster_id: e.target.value })}>
              <MenuItem value="" disabled><Typography color="text.disabled">选择可观测集群</Typography></MenuItem>
              {clusters.map((c) => (<MenuItem key={c.id} value={c.id}>{c.display_name || c.name}{c.in_cluster ? ' · In-Cluster' : ''}</MenuItem>))}
            </Select>
          </FormControl>
          <TextField fullWidth size="small" label="接入点 URL" value={form.endpoint} onChange={(e) => setForm({ ...form, endpoint: e.target.value })} sx={{ mb: 2 }} helperText="可选" />
          <TextField fullWidth size="small" label="Labels (JSON)" value={form.labels} onChange={(e) => setForm({ ...form, labels: e.target.value })} sx={{ mb: 2 }} multiline minRows={2} helperText='{"region": "cn-east"}' />
          <Typography variant="subtitle2" sx={{ mb: 1, mt: 1 }}>容量配置</Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <TextField size="small" label="最大实例数" type="number" value={form.max_instances} onChange={(e) => setForm({ ...form, max_instances: e.target.value })} sx={{ flex: 1 }} helperText="0=不限制" />
            <TextField size="small" label="最大存储" value={form.max_storage} onChange={(e) => setForm({ ...form, max_storage: e.target.value })} sx={{ flex: 1 }} helperText="例: 500Gi" />
          </Box>
          {editingId && <TextField fullWidth size="small" label="状态" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })} sx={{ mb: 2, mt: 2 }} />}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.slug || !form.display_name}>{saving ? '保存中...' : editingId ? '更新' : '创建'}</Button>
        </DialogActions>
      </Dialog>

      {/* InitShared 配置弹窗 */}
      <Dialog open={initDialog.open} onClose={() => setInitDialog({ open: false })} maxWidth="md" fullWidth>
        <DialogTitle>初始化共享 VM 集群 — {initDialog.zone?.display_name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            将使用 Helm Chart <code>vm/victoria-metrics-k8s-stack</code> 在可用区集群中部署共享 VM 监控栈（含 vminsert / vmselect / vmstorage / vmauth / grafana）。
          </Alert>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField fullWidth size="small" label="Namespace" value={initForm.namespace}
                onChange={(e) => setInitForm({ ...initForm, namespace: e.target.value })} helperText="部署目标命名空间" />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField fullWidth size="small" label="Release Name" value={initForm.release_name}
                onChange={(e) => setInitForm({ ...initForm, release_name: e.target.value })} helperText="Helm release 名称" />
            </Grid>
          </Grid>
          <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>Helm Values (可编辑 JSON)</Typography>
          <TextField fullWidth multiline minRows={10} maxRows={20} size="small"
            value={initForm.helm_values}
            onChange={(e) => setInitForm({ ...initForm, helm_values: e.target.value })}
            sx={{ fontFamily: 'monospace', '& textarea': { fontSize: '0.8125rem', fontFamily: 'monospace' } }}
            helperText="自定义 Helm values 覆盖默认配置。留空或保持默认值则使用预设。"
          />
          {preflightResult && (
            <Alert severity={preflightResult.ok ? 'success' : 'warning'} sx={{ mt: 2 }}>
              {preflightResult.ok ? '预检通过，可以执行初始化' : `预检发现 ${preflightResult.issues?.length || 0} 项问题`}
              {!preflightResult.ok && preflightResult.issues?.map((issue, i) => (
                <Typography key={i} variant="caption" sx={{ display: 'block' }}>
                  · [{issue.component}] {issue.reason}{issue.message ? `: ${issue.message}` : ''}
                </Typography>
              ))}
            </Alert>
          )}
          {initPlan && (
            <Box component="pre" sx={{ mt: 2, p: 2, borderRadius: 1, backgroundColor: '#f8f9fa', fontSize: 11, overflowX: 'auto', m: 0, maxHeight: 300 }}>
              {JSON.stringify(initPlan, null, 2)}
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setInitDialog({ open: false })}>取消</Button>
          <Button variant="outlined" onClick={runPreflight} disabled={initLoading}>预检</Button>
          <Button variant="outlined" onClick={handleInitDryRun} disabled={initLoading}>{initLoading ? '执行中...' : 'Dry-run 预览'}</Button>
          <Button variant="contained" onClick={() => setInitConfirmOpen(true)} disabled={initLoading || !preflightResult?.ok}>应用初始化</Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog open={initConfirmOpen} title="确认初始化共享 VM 集群"
        message={`将在可用区「${initDialog.zone?.display_name}」的集群中执行 Helm install/upgrade。请确认已执行 dry-run 并核对预览内容。`}
        confirmLabel="确认初始化" severity="warning" loading={initLoading} onConfirm={handleInitApply} onCancel={() => setInitConfirmOpen(false)} />

      <ConfirmDialog open={deleteDialog.open} title="删除可用区"
        message={`确定要删除可用区「${deleteDialog.zone?.display_name}」吗？不可逆，且要求无活跃实例。`}
        severity="error" confirmLabel="删除" onConfirm={handleDelete} onCancel={() => setDeleteDialog({ open: false })} />

      <Dialog open={componentsDialog.open} onClose={() => setComponentsDialog({ open: false })} maxWidth="md" fullWidth>
        <DialogTitle>组件状态 — {componentsDialog.zone?.display_name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          {componentsLoading ? (
            <Typography variant="body2" color="text.secondary">加载中...</Typography>
          ) : components.length === 0 ? (
            <EmptyState title="暂无组件信息" description="可用区尚未初始化或后端未上报组件状态" />
          ) : (
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>组件</TableCell>
                    <TableCell>类型</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>版本</TableCell>
                    <TableCell>说明</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {components.map((c, idx) => (
                    <TableRow key={idx}>
                      <TableCell sx={{ fontWeight: 500 }}>{c.name}</TableCell>
                      <TableCell>{c.component}</TableCell>
                      <TableCell><StatusChip status={c.status} /></TableCell>
                      <TableCell>{c.version || '-'}</TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{c.message || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setComponentsDialog({ open: false })}>关闭</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
