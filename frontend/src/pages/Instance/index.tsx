import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
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
import { useAuthStore } from '../../stores/useAuthStore';
import { instanceAPI } from '../../api/instance';
import { clusterAPI, type Cluster } from '../../api/cluster';
import { zoneAPI, type Zone } from '../../api/zone';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import type { Instance } from '../../types/api';
import { parseSpec } from '../../utils/instance';

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
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; instance?: Instance }>({ open: false });
  const [saving, setSaving] = useState(false);

  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);

  const fetchInstances = useCallback(async () => {
    setLoading(true);
    try {
      // 本页只承载 metrics 数据面；日志实例在「日志 → 日志实例」独立管理。
      const { data: res } = await instanceAPI.list({
        page: page + 1,
        page_size: pageSize,
        search,
        instance_type: 'metrics',
        status: statusFilter || undefined,
      });
      setInstances(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取实例列表失败'), { variant: 'error' });
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
        const { data: res } = await clusterAPI.list({ page: 1, page_size: 100 });
        setClusters(res.data?.items || []);
      } catch {
        /* cluster list optional */
      }
    })();
    (async () => {
      try {
        const { data: res } = await zoneAPI.list({ page: 1, page_size: 100 });
        setZones(res.data?.items || []);
      } catch {
        /* zone list optional */
      }
    })();
    (async () => {
      try {
        const { data: res } = await grafanaInstanceAPI.list({ page: 1, page_size: 100 });
        setGrafanaHosts(res.data?.items || []);
      } catch {
        /* grafana host list optional */
      }
    })();
  }, []);

  const clusterNameById = useMemo(() => {
    return clusters.reduce<Record<string, string>>((acc, c) => {
      acc[c.id] = c.display_name || c.name;
      return acc;
    }, {});
  }, [clusters]);

  const zoneNameById = useMemo(() => {
    return zones.reduce<Record<string, string>>((acc, z) => {
      acc[z.id] = z.display_name || z.slug;
      return acc;
    }, {});
  }, [zones]);

  const grafanaInstanceNameById = useMemo(() => {
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

  const handleDelete = async () => {
    if (!deleteDialog.instance) return;
    setSaving(true);
    try {
      await instanceAPI.delete(deleteDialog.instance.id);
      enqueueSnackbar('实例删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchInstances();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading && instances.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="监控实例"
        subtitle="VictoriaMetrics 指标数据面：绑定 Zone 共享 VM 集群"
        actionLabel={isAdmin ? '创建监控实例' : undefined}
        onAction={isAdmin ? () => navigate('/instances/create') : undefined}
      />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2 }}>
          <Typography variant="subtitle2" sx={{ mb: 1.5 }}>监控实例概览（当前页）</Typography>
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
          placeholder="搜索监控实例名称..."
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
                <TableCell>监控实例名称</TableCell>
                <TableCell>模板</TableCell>
                <TableCell>规格</TableCell>
                <TableCell>命名空间</TableCell>
                <TableCell>可用区</TableCell>
                <TableCell>目标集群</TableCell>
                <TableCell>关联 Grafana</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10}>
                    <EmptyState title="暂无监控实例" description="点击右上角按钮创建第一个监控实例" />
                  </TableCell>
                </TableRow>
              ) : (
                instances.map((inst) => {
                  const spec = parseSpec(inst.spec);
                  return (
                    <TableRow key={inst.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{inst.instance_name}</TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem' }}>{inst.template_type}</TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                        {spec.cpu}C / {spec.memory}G / {spec.storage}Gi
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>{inst.namespace || '-'}</TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {inst.zone_id ? (zoneNameById[inst.zone_id] || inst.zone_id.slice(0, 8)) : '-'}
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {inst.cluster_id ? (clusterNameById[inst.cluster_id] || inst.cluster_id.slice(0, 8)) : '默认'}
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {inst.grafana_instance_id ? (grafanaInstanceNameById[inst.grafana_instance_id] || inst.grafana_instance_id.slice(0, 8)) : '默认'}
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

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除监控实例"
        message={`确定要删除监控实例「${deleteDialog.instance?.instance_name}」吗？关联的 Helm Release 也将被卸载。`}
        severity="error"
        confirmLabel="删除"
        loading={saving}
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
