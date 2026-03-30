import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Button,
  FormControl,
  Grid,
  IconButton,
  InputAdornment,
  InputLabel,
  LinearProgress,
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
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import ScaleIcon from '@mui/icons-material/TuneOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { instanceAPI } from '../../api/instance';
import type { Instance, InstanceSpec } from '../../types/api';

const typeLabels: Record<string, { label: string; color: 'primary' | 'secondary' | 'success' | 'warning' }> = {
  metrics: { label: 'Metrics', color: 'primary' },
  logs: { label: 'Logs', color: 'secondary' },
  visual: { label: 'Grafana', color: 'success' },
  alert: { label: 'Alert', color: 'warning' },
};

function parseSpec(spec: string): InstanceSpec {
  try { return JSON.parse(spec); }
  catch { return { cpu: 0, memory: 0, storage: 0, retention: 0 }; }
}

function ResourceBar({ label, used, total, unit }: { label: string; used: number; total: number; unit: string }) {
  const pct = total > 0 ? Math.min((used / total) * 100, 100) : 0;
  const color = pct > 85 ? 'error' : pct > 60 ? 'warning' : 'primary';
  return (
    <Box sx={{ mb: 1.5 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
        <Typography variant="body2" color="text.secondary">{label}</Typography>
        <Typography variant="body2">{used} / {total} {unit} ({pct.toFixed(0)}%)</Typography>
      </Box>
      <LinearProgress variant="determinate" value={pct} color={color} sx={{ height: 6, borderRadius: 3 }} />
    </Box>
  );
}

export default function InstancePage() {
  const { enqueueSnackbar } = useSnackbar();
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [createOpen, setCreateOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [detailDialog, setDetailDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [scaleDialog, setScaleDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [saving, setSaving] = useState(false);

  const [createForm, setCreateForm] = useState({
    tenant_id: '', instance_name: '', instance_type: 'metrics', template_type: 'dedicated_single',
    cpu: '2', memory: '4', storage: '50', retention: '15',
  });
  const [scaleForm, setScaleForm] = useState({ cpu: 2, memory: 4, storage: 50 });

  const fetchInstances = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await instanceAPI.list({
        page: page + 1, page_size: pageSize, search,
        instance_type: typeFilter || undefined, status: statusFilter || undefined,
      });
      setInstances(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch {
      enqueueSnackbar('获取实例列表失败', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, typeFilter, statusFilter, enqueueSnackbar]);

  useEffect(() => { fetchInstances(); }, [fetchInstances]);

  const handleCreate = async () => {
    setSaving(true);
    try {
      const spec = JSON.stringify({
        cpu: parseInt(createForm.cpu), memory: parseInt(createForm.memory),
        storage: parseInt(createForm.storage), retention: parseInt(createForm.retention),
      });
      await instanceAPI.create({
        tenant_id: createForm.tenant_id, instance_name: createForm.instance_name,
        instance_type: createForm.instance_type, template_type: createForm.template_type, spec,
      });
      enqueueSnackbar('实例创建成功', { variant: 'success' });
      setCreateOpen(false);
      fetchInstances();
    } catch {
      enqueueSnackbar('创建失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleScale = async () => {
    if (!scaleDialog.instance) return;
    setSaving(true);
    try {
      await instanceAPI.scale(scaleDialog.instance.id, scaleForm);
      enqueueSnackbar('扩容请求已提交', { variant: 'success' });
      setScaleDialog({ open: false });
      fetchInstances();
    } catch {
      enqueueSnackbar('扩容失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.instance) return;
    try {
      await instanceAPI.delete(deleteDialog.instance.id);
      enqueueSnackbar('实例删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchInstances();
    } catch {
      enqueueSnackbar('删除失败', { variant: 'error' });
    }
  };

  if (loading && instances.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="实例管理" subtitle="管理 VM 实例的完整生命周期：创建、扩容、停止、销毁" actionLabel="创建实例" onAction={() => setCreateOpen(true)} />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
          <TextField
            placeholder="搜索实例..."
            size="small"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
            sx={{ width: 240 }}
          />
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>类型</InputLabel>
            <Select value={typeFilter} label="类型" onChange={(e) => { setTypeFilter(e.target.value); setPage(0); }}>
              <MenuItem value="">全部</MenuItem>
              <MenuItem value="metrics">Metrics</MenuItem>
              <MenuItem value="logs">Logs</MenuItem>
              <MenuItem value="visual">Grafana</MenuItem>
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>状态</InputLabel>
            <Select value={statusFilter} label="状态" onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}>
              <MenuItem value="">全部</MenuItem>
              <MenuItem value="running">运行中</MenuItem>
              <MenuItem value="creating">创建中</MenuItem>
              <MenuItem value="stopped">已停止</MenuItem>
              <MenuItem value="error">异常</MenuItem>
            </Select>
          </FormControl>
        </Box>
      </Card>

      <Card>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>实例名称</TableCell>
                <TableCell>类型</TableCell>
                <TableCell>模板</TableCell>
                <TableCell>规格</TableCell>
                <TableCell>命名空间</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8}>
                    <EmptyState title="暂无实例" description="点击右上角按钮创建第一个实例" />
                  </TableCell>
                </TableRow>
              ) : (
                instances.map((inst) => {
                  const spec = parseSpec(inst.spec);
                  return (
                    <TableRow key={inst.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{inst.instance_name}</TableCell>
                      <TableCell>
                        <Chip label={typeLabels[inst.instance_type]?.label || inst.instance_type} size="small" color={typeLabels[inst.instance_type]?.color || 'default'} variant="outlined" />
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem' }}>{inst.template_type}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                        {spec.cpu}C / {spec.memory}G / {spec.storage}Gi
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{inst.namespace || '-'}</TableCell>
                      <TableCell><StatusChip status={inst.status} /></TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(inst.created_at).toLocaleDateString()}</TableCell>
                      <TableCell align="right">
                        <Tooltip title="详情">
                          <IconButton size="small" onClick={() => setDetailDialog({ open: true, instance: inst })}>
                            <InfoOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        {inst.instance_type === 'visual' && inst.url && (
                          <Tooltip title="打开 Grafana">
                            <IconButton size="small" onClick={() => window.open(inst.url, '_blank')}>
                              <OpenInNewIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                        {inst.status === 'running' && inst.instance_type !== 'visual' && (
                          <Tooltip title="扩容">
                            <IconButton size="small" onClick={() => {
                              const s = parseSpec(inst.spec);
                              setScaleForm({ cpu: s.cpu, memory: s.memory, storage: s.storage });
                              setScaleDialog({ open: true, instance: inst });
                            }}>
                              <ScaleIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                        <Tooltip title="删除">
                          <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, instance: inst })}>
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
        {total > 0 && (
          <TablePagination
            component="div" count={total} page={page} onPageChange={(_, p) => setPage(p)}
            rowsPerPage={pageSize} onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value)); setPage(0); }}
            rowsPerPageOptions={[10, 20, 50]} labelRowsPerPage="每页行数"
          />
        )}
      </Card>

      {/* Create dialog */}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>创建实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="租户 ID" value={createForm.tenant_id} onChange={(e) => setCreateForm({ ...createForm, tenant_id: e.target.value })} sx={{ mb: 2.5 }} required helperText="关联租户的 UUID" />
          <TextField fullWidth label="实例名称" value={createForm.instance_name} onChange={(e) => setCreateForm({ ...createForm, instance_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <Grid container spacing={2} sx={{ mb: 2.5 }}>
            <Grid size={{ xs: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>实例类型</InputLabel>
                <Select value={createForm.instance_type} label="实例类型" onChange={(e) => setCreateForm({ ...createForm, instance_type: e.target.value })}>
                  <MenuItem value="metrics">Metrics (VictoriaMetrics)</MenuItem>
                  <MenuItem value="logs">Logs (VictoriaLogs)</MenuItem>
                  <MenuItem value="visual">Grafana</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>模板类型</InputLabel>
                <Select value={createForm.template_type} label="模板类型" onChange={(e) => setCreateForm({ ...createForm, template_type: e.target.value })}>
                  <MenuItem value="shared">共享版</MenuItem>
                  <MenuItem value="dedicated_single">独享单节点</MenuItem>
                  <MenuItem value="dedicated_cluster">独享集群</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
          <Typography variant="subtitle2" sx={{ mb: 1.5, color: 'text.secondary' }}>资源配置</Typography>
          <Grid container spacing={2}>
            <Grid size={{ xs: 3 }}>
              <TextField fullWidth size="small" label="CPU (核)" type="number" value={createForm.cpu} onChange={(e) => setCreateForm({ ...createForm, cpu: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField fullWidth size="small" label="内存 (GB)" type="number" value={createForm.memory} onChange={(e) => setCreateForm({ ...createForm, memory: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField fullWidth size="small" label="存储 (GB)" type="number" value={createForm.storage} onChange={(e) => setCreateForm({ ...createForm, storage: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField fullWidth size="small" label="保留 (天)" type="number" value={createForm.retention} onChange={(e) => setCreateForm({ ...createForm, retention: e.target.value })} />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving || !createForm.tenant_id || !createForm.instance_name}>
            {saving ? '创建中...' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Detail dialog */}
      <Dialog open={detailDialog.open} onClose={() => setDetailDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>实例详情</DialogTitle>
        {detailDialog.instance && (() => {
          const inst = detailDialog.instance;
          const spec = parseSpec(inst.spec);
          return (
            <DialogContent>
              <Box sx={{ mb: 2 }}>
                <Typography variant="body2" color="text.secondary">实例名称</Typography>
                <Typography variant="body1" sx={{ fontWeight: 500 }}>{inst.instance_name}</Typography>
              </Box>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid size={{ xs: 4 }}>
                  <Typography variant="body2" color="text.secondary">类型</Typography>
                  <Chip label={typeLabels[inst.instance_type]?.label} size="small" color={typeLabels[inst.instance_type]?.color} />
                </Grid>
                <Grid size={{ xs: 4 }}>
                  <Typography variant="body2" color="text.secondary">状态</Typography>
                  <StatusChip status={inst.status} />
                </Grid>
                <Grid size={{ xs: 4 }}>
                  <Typography variant="body2" color="text.secondary">命名空间</Typography>
                  <Typography variant="body2">{inst.namespace || '-'}</Typography>
                </Grid>
              </Grid>
              <Card variant="outlined" sx={{ p: 2, mb: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1.5 }}>资源使用</Typography>
                <ResourceBar label="CPU" used={Math.round(spec.cpu * 0.6)} total={spec.cpu} unit="Core" />
                <ResourceBar label="内存" used={Math.round(spec.memory * 0.52)} total={spec.memory} unit="GiB" />
                <ResourceBar label="存储" used={Math.round(spec.storage * 0.64)} total={spec.storage} unit="GiB" />
              </Card>
              {inst.url && (
                <Box>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>访问地址</Typography>
                  <Chip label={inst.url} size="small" variant="outlined" sx={{ fontFamily: 'monospace' }} onClick={() => window.open(inst.url, '_blank')} />
                </Box>
              )}
            </DialogContent>
          );
        })()}
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDetailDialog({ open: false })}>关闭</Button>
        </DialogActions>
      </Dialog>

      {/* Scale dialog */}
      <Dialog open={scaleDialog.open} onClose={() => setScaleDialog({ open: false })} maxWidth="xs" fullWidth>
        <DialogTitle>实例扩容 - {scaleDialog.instance?.instance_name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="CPU (核)" type="number" value={scaleForm.cpu} onChange={(e) => setScaleForm({ ...scaleForm, cpu: parseInt(e.target.value) || 0 })} sx={{ mb: 2 }} />
          <TextField fullWidth size="small" label="内存 (GB)" type="number" value={scaleForm.memory} onChange={(e) => setScaleForm({ ...scaleForm, memory: parseInt(e.target.value) || 0 })} sx={{ mb: 2 }} />
          <TextField fullWidth size="small" label="存储 (GB)" type="number" value={scaleForm.storage} onChange={(e) => setScaleForm({ ...scaleForm, storage: parseInt(e.target.value) || 0 })} />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setScaleDialog({ open: false })}>取消</Button>
          <Button variant="contained" onClick={handleScale} disabled={saving}>{saving ? '提交中...' : '确认扩容'}</Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除实例"
        message={`确定要删除实例「${deleteDialog.instance?.instance_name}」吗？关联的 Helm Release 也将被卸载。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
