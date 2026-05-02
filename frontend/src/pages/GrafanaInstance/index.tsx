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
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { extractApiError } from '../../api';
import type { Instance, InstanceSpec } from '../../types/api';

function parseSpec(spec: string): InstanceSpec {
  try {
    return JSON.parse(spec);
  } catch {
    return { cpu: 0, memory: 0, storage: 0, retention: 0 };
  }
}

const statusFilterItems = [
  { key: '', label: '全部状态' },
  { key: 'running', label: '运行中' },
  { key: 'creating', label: '创建中' },
  { key: 'stopped', label: '已停止' },
  { key: 'error', label: '异常' },
];

export default function GrafanaInstancePage() {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [createOpen, setCreateOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [saving, setSaving] = useState(false);

  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaHost[]>([]);
  const [createForm, setCreateForm] = useState({
    tenant_id: '',
    instance_name: '',
    grafana_host_id: '',
    cpu: '2',
    memory: '4',
    storage: '50',
    retention: '15',
  });

  const fetchInstances = useCallback(async () => {
    setLoading(true);
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
      setLoading(false);
    }
  }, [page, pageSize, search, statusFilter, enqueueSnackbar]);

  useEffect(() => {
    fetchInstances();
  }, [fetchInstances]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await grafanaHostAPI.list({ page: 1, page_size: 100 });
        setGrafanaHosts(res.data?.items || []);
      } catch {
        /* grafana host list optional */
      }
    })();
  }, []);

  const grafanaHostNameById = useMemo(() => {
    return grafanaHosts.reduce<Record<string, string>>((acc, h) => {
      acc[h.id] = h.name;
      return acc;
    }, {});
  }, [grafanaHosts]);

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

  if (loading && instances.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="Grafana 实例"
        subtitle="管理 Grafana 可视化实例，创建后可关联 Grafana 主机进行 Dashboard 下发"
        actionLabel="创建实例"
        onAction={() => setCreateOpen(true)}
      />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2 }}>
          <Typography variant="subtitle2" sx={{ mb: 1.5 }}>Grafana 实例概览（当前页）</Typography>
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
                <TableCell>规格</TableCell>
                <TableCell>命名空间</TableCell>
                <TableCell>关联 Grafana</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8}>
                    <EmptyState title="暂无 Grafana 实例" description="点击右上角按钮创建第一个 Grafana 实例" />
                  </TableCell>
                </TableRow>
              ) : (
                instances.map((inst) => {
                  const spec = parseSpec(inst.spec);
                  return (
                    <TableRow key={inst.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{inst.instance_name}</TableCell>
                      <TableCell>
                        <Chip label="Grafana" size="small" color="success" variant="outlined" />
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                        {spec.cpu}C / {spec.memory}G / {spec.storage}Gi
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{inst.namespace || '-'}</TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {inst.grafana_host_id ? (grafanaHostNameById[inst.grafana_host_id] || inst.grafana_host_id.slice(0, 8)) : '默认'}
                      </TableCell>
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
                        <Tooltip title={inst.url ? '打开 Grafana' : '暂无地址'}>
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
        <DialogTitle>创建 Grafana 实例</DialogTitle>
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
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>关联 Grafana 主机</InputLabel>
            <Select
              value={createForm.grafana_host_id}
              label="关联 Grafana 主机"
              onChange={(e) => setCreateForm({ ...createForm, grafana_host_id: e.target.value })}
            >
              <MenuItem value="">继承租户默认</MenuItem>
              {grafanaHosts.map((h) => (
                <MenuItem key={h.id} value={h.id}>
                  {h.name}
                  <Chip
                    size="small"
                    label={h.scope === 'platform' ? '平台' : '租户'}
                    variant="outlined"
                    sx={{ ml: 1, height: 18, fontSize: '0.65rem' }}
                  />
                </MenuItem>
              ))}
            </Select>
          </FormControl>
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

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除 Grafana 实例"
        message={`确定要删除 Grafana 实例「${deleteDialog.instance?.instance_name}」吗？关联的 Helm Release 也将被卸载。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
