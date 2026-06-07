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
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import { businessClusterAPI, type BusinessCluster, type CreateBusinessClusterRequest } from '../../api/businessCluster';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';

const agentStatusLabels: Record<string, { label: string; color: 'success' | 'warning' | 'error' | 'default' }> = {
  pending: { label: '待部署', color: 'default' },
  deploying: { label: '部署中', color: 'warning' },
  active: { label: '运行中', color: 'success' },
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
  const isAdmin = user?.role === 'admin' || user?.role === 'tenant_admin';
  const [searchParams] = useSearchParams();

  const tenantFilter = searchParams.get('tenant_id') || '';
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
  const [form, setForm] = useState<CreateBusinessClusterRequest>(defaultForm);
  const [saving, setSaving] = useState(false);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await businessClusterAPI.list({
        page: page + 1,
        page_size: pageSize,
        search,
        tenant_id: tenantFilter || undefined,
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

  const parseLabels = (labelsJson: string): Record<string, string> => {
    try { return JSON.parse(labelsJson || '{}'); } catch { return {}; }
  };

  if (loading && clusters.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="业务集群管理"
        subtitle="管理租户接入的业务 Kubernetes 集群，通过 VMAgent CR 采集监控数据"
        actionLabel={isAdmin ? '接入业务集群' : undefined}
        onAction={isAdmin ? () => {
          setForm({ ...defaultForm, instance_id: instanceFilter });
          setDialogOpen(true);
        } : undefined}
      />

      {!isAdmin && (
        <Alert severity="info" sx={{ mb: 2 }}>
          仅租户管理员可接入/移除业务集群。当前仅提供只读视图。
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
                <TableCell>Agent 状态</TableCell>
                <TableCell>标签</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {clusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState
                      title="暂无业务集群"
                      description={isAdmin ? '点击右上角按钮接入第一个业务集群' : '当前租户下没有已接入的业务集群'}
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
                        {isAdmin && (
                          <Tooltip title="移除">
                            <IconButton
                              size="small"
                              color="error"
                              onClick={() => setDeleteDialog({ open: true, cluster: c })}
                            >
                              <DeleteOutlinedIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
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
