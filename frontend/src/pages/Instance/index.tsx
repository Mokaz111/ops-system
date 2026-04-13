import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Card,
  Chip,
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
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import { instanceAPI } from '../../api/instance';
import type { Instance, InstanceSpec } from '../../types/api';

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

function isInstanceLevelScalable(inst: Instance): boolean {
  return inst.status === 'running' && inst.instance_type !== 'visual' && inst.template_type === 'dedicated_single';
}

const typeFilterItems = [
  { key: '', label: '全部' },
  { key: 'metrics', label: 'Metrics' },
  { key: 'logs', label: 'Logs' },
  { key: 'visual', label: 'Grafana' },
];

const statusFilterItems = [
  { key: '', label: '全部状态' },
  { key: 'running', label: '运行中' },
  { key: 'creating', label: '创建中' },
  { key: 'stopped', label: '已停止' },
  { key: 'error', label: '异常' },
];

export default function InstancePage() {
  const navigate = useNavigate();
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
  const [scaleDialog, setScaleDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [saving, setSaving] = useState(false);

  const [createForm, setCreateForm] = useState({
    tenant_id: '',
    instance_name: '',
    instance_type: 'metrics',
    template_type: 'dedicated_single',
    cpu: '2',
    memory: '4',
    storage: '50',
    retention: '15',
  });
  const [scaleForm, setScaleForm] = useState({
    scale_type: 'vertical' as 'horizontal' | 'vertical' | 'storage',
    replicas: 1,
    cpu: 2,
    memory: 4,
    storage: 50,
  });

  const fetchInstances = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await instanceAPI.list({
        page: page + 1,
        page_size: pageSize,
        search,
        instance_type: typeFilter || undefined,
        status: statusFilter || undefined,
      });
      setInstances(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch {
      enqueueSnackbar('获取实例列表失败', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, typeFilter, statusFilter, enqueueSnackbar]);

  useEffect(() => {
    fetchInstances();
  }, [fetchInstances]);

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

  const typeStats = useMemo(() => {
    return instances.reduce<Record<string, number>>((acc, item) => {
      acc[item.instance_type] = (acc[item.instance_type] || 0) + 1;
      return acc;
    }, {});
  }, [instances]);

  const handleCreate = async () => {
    setSaving(true);
    try {
      const spec = JSON.stringify({
        cpu: parseInt(createForm.cpu, 10),
        memory: parseInt(createForm.memory, 10),
        storage: parseInt(createForm.storage, 10),
        retention: parseInt(createForm.retention, 10),
      });
      await instanceAPI.create({
        tenant_id: createForm.tenant_id,
        instance_name: createForm.instance_name,
        instance_type: createForm.instance_type,
        template_type: createForm.template_type,
        spec,
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
      const payload = {
        scale_type: scaleForm.scale_type,
        replicas: scaleForm.scale_type === 'horizontal' ? Math.max(1, scaleForm.replicas) : undefined,
        cpu: scaleForm.scale_type === 'vertical' ? String(scaleForm.cpu) : undefined,
        memory: scaleForm.scale_type === 'vertical' ? `${scaleForm.memory}Gi` : undefined,
        storage: scaleForm.scale_type === 'storage' ? `${scaleForm.storage}Gi` : undefined,
      };
      await instanceAPI.scale(scaleDialog.instance.id, payload);
      enqueueSnackbar('伸缩请求已提交', { variant: 'success' });
      setScaleDialog({ open: false });
      fetchInstances();
    } catch {
      enqueueSnackbar('伸缩失败', { variant: 'error' });
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
      <PageHeader
        title="实例管理"
        subtitle="统一管理实例生命周期，提供详情、监控、伸缩与删除操作"
        actionLabel="创建实例"
        onAction={() => setCreateOpen(true)}
      />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2 }}>
          <Typography variant="subtitle2" sx={{ mb: 1.5 }}>接入数据概览（当前页）</Typography>
          <Grid container spacing={1.5} sx={{ mb: 2 }}>
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
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            {typeFilterItems.map((item) => (
              <Chip
                key={item.key || 'all'}
                label={`${item.label} (${item.key ? (typeStats[item.key] || 0) : statusStats.total})`}
                color={typeFilter === item.key ? 'primary' : 'default'}
                variant={typeFilter === item.key ? 'filled' : 'outlined'}
                onClick={() => {
                  setTypeFilter(item.key);
                  setPage(0);
                }}
              />
            ))}
          </Box>
        </Box>
      </Card>

      <FilterToolbar>
        <TextField
          placeholder="搜索实例名称..."
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
          <InputLabel>类型</InputLabel>
          <Select
            value={typeFilter}
            label="类型"
            onChange={(e) => {
              setTypeFilter(e.target.value);
              setPage(0);
            }}
          >
            {typeFilterItems.map((item) => (
              <MenuItem key={item.key || 'all'} value={item.key}>{item.label}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <InputLabel>状态</InputLabel>
          <Select
            value={statusFilter}
            label="状态"
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(0);
            }}
          >
            {statusFilterItems.map((item) => (
              <MenuItem key={item.key || 'all'} value={item.key}>{item.label}</MenuItem>
            ))}
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
                  const typeMeta = typeLabels[inst.instance_type];
                  return (
                    <TableRow key={inst.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{inst.instance_name}</TableCell>
                      <TableCell>
                        <Chip
                          label={typeMeta?.label || inst.instance_type}
                          size="small"
                          color={typeMeta?.color || 'default'}
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem' }}>{inst.template_type}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                        {spec.cpu}C / {spec.memory}G / {spec.storage}Gi
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{inst.namespace || '-'}</TableCell>
                      <TableCell><StatusChip status={inst.status} /></TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                        {new Date(inst.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="详情">
                          <IconButton size="small" onClick={() => navigate(`/instances/${inst.id}`)} aria-label="查看实例详情">
                            <InfoOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title={inst.url ? '打开监控' : '暂无监控地址'}>
                          <span>
                            <IconButton
                              size="small"
                              disabled={!inst.url}
                              onClick={() => inst.url && window.open(inst.url, '_blank')}
                            >
                              <OpenInNewIcon fontSize="small" />
                            </IconButton>
                          </span>
                        </Tooltip>
                        {isInstanceLevelScalable(inst) ? (
                          <Tooltip title="伸缩">
                            <IconButton
                              size="small"
                              onClick={() => {
                                const s = parseSpec(inst.spec);
                                setScaleForm({
                                  scale_type: 'vertical',
                                  replicas: s.replicas || 1,
                                  cpu: s.cpu || 1,
                                  memory: s.memory || 1,
                                  storage: s.storage || 1,
                                });
                                setScaleDialog({ open: true, instance: inst });
                              }}
                            >
                              <ScaleIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        ) : (
                          <Tooltip title="共享版/独享集群版由平台管理员在集群层扩容">
                            <span>
                              <IconButton size="small" disabled aria-label="该模板不支持实例级伸缩">
                                <ScaleIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                        )}
                        <Tooltip title="删除">
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => setDeleteDialog({ open: true, instance: inst })}
                          >
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
      </DataTableCard>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>创建实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField
            fullWidth
            label="租户 ID"
            value={createForm.tenant_id}
            onChange={(e) => setCreateForm({ ...createForm, tenant_id: e.target.value })}
            sx={{ mb: 2.5 }}
            required
            helperText="关联租户的 UUID"
          />
          <TextField
            fullWidth
            label="实例名称"
            value={createForm.instance_name}
            onChange={(e) => setCreateForm({ ...createForm, instance_name: e.target.value })}
            sx={{ mb: 2.5 }}
            required
          />
          <Grid container spacing={2} sx={{ mb: 2.5 }}>
            <Grid size={{ xs: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>实例类型</InputLabel>
                <Select
                  value={createForm.instance_type}
                  label="实例类型"
                  onChange={(e) => setCreateForm({ ...createForm, instance_type: e.target.value })}
                >
                  <MenuItem value="metrics">Metrics (VictoriaMetrics)</MenuItem>
                  <MenuItem value="logs">Logs (VictoriaLogs)</MenuItem>
                  <MenuItem value="visual">Grafana</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>模板类型</InputLabel>
                <Select
                  value={createForm.template_type}
                  label="模板类型"
                  onChange={(e) => setCreateForm({ ...createForm, template_type: e.target.value })}
                >
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
              <TextField
                fullWidth
                size="small"
                label="CPU (核)"
                type="number"
                value={createForm.cpu}
                onChange={(e) => setCreateForm({ ...createForm, cpu: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="内存 (GB)"
                type="number"
                value={createForm.memory}
                onChange={(e) => setCreateForm({ ...createForm, memory: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="存储 (GB)"
                type="number"
                value={createForm.storage}
                onChange={(e) => setCreateForm({ ...createForm, storage: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="保留 (天)"
                type="number"
                value={createForm.retention}
                onChange={(e) => setCreateForm({ ...createForm, retention: e.target.value })}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>取消</Button>
          <Button
            variant="contained"
            onClick={handleCreate}
            disabled={saving || !createForm.tenant_id || !createForm.instance_name}
          >
            {saving ? '创建中...' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={scaleDialog.open} onClose={() => setScaleDialog({ open: false })} maxWidth="xs" fullWidth>
        <DialogTitle>实例伸缩 - {scaleDialog.instance?.instance_name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>伸缩类型</InputLabel>
            <Select
              value={scaleForm.scale_type}
              label="伸缩类型"
              onChange={(e) => setScaleForm({ ...scaleForm, scale_type: e.target.value as 'horizontal' | 'vertical' | 'storage' })}
            >
              <MenuItem value="vertical">垂直伸缩（CPU/内存）</MenuItem>
              <MenuItem value="storage">存储扩容</MenuItem>
            </Select>
          </FormControl>
          {scaleForm.scale_type === 'vertical' && (
            <>
              <TextField
                fullWidth
                size="small"
                label="CPU (核)"
                type="number"
                value={scaleForm.cpu}
                onChange={(e) => setScaleForm({ ...scaleForm, cpu: parseInt(e.target.value, 10) || 1 })}
                sx={{ mb: 2 }}
              />
              <TextField
                fullWidth
                size="small"
                label="内存 (Gi)"
                type="number"
                value={scaleForm.memory}
                onChange={(e) => setScaleForm({ ...scaleForm, memory: parseInt(e.target.value, 10) || 1 })}
              />
            </>
          )}
          {scaleForm.scale_type === 'storage' && (
            <TextField
              fullWidth
              size="small"
              label="存储 (Gi)"
              type="number"
              value={scaleForm.storage}
              onChange={(e) => setScaleForm({ ...scaleForm, storage: parseInt(e.target.value, 10) || 1 })}
            />
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setScaleDialog({ open: false })}>取消</Button>
          <Button variant="contained" onClick={handleScale} disabled={saving}>
            {saving ? '提交中...' : '确认伸缩'}
          </Button>
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
