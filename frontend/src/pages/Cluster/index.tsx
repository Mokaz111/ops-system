import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { clusterAPI, type Cluster } from '../../api/cluster';
import { zoneAPI, type Zone } from '../../api/zone';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';

interface FormState {
  name: string;
  display_name: string;
  description: string;
  in_cluster: boolean;
  kubeconfig: string;
  kubeconfig_path: string;
  status: string;
}

const defaultForm: FormState = {
  name: '',
  display_name: '',
  description: '',
  in_cluster: false,
  kubeconfig: '',
  kubeconfig_path: '',
  status: 'active',
};

export default function ClusterPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);
  const [zones, setZones] = useState<Zone[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; cluster?: Cluster }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(defaultForm);
  const [saving, setSaving] = useState(false);

  const zoneByClusterId = useMemo(() => {
    const map: Record<string, Zone> = {};
    for (const z of zones) {
      if (z.cluster_id) map[z.cluster_id] = z;
    }
    return map;
  }, [zones]);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await clusterAPI.list({ page: 1, page_size: 100 });
      setClusters(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取可观测集群列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => { fetch(); }, [fetch]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await zoneAPI.list({ page: 1, page_size: 100 });
        setZones(res.data?.items || []);
      } catch { /* zones optional for binding display */ }
    })();
  }, []);

  const openCreate = () => {
    setEditingId(null);
    setForm(defaultForm);
    setDialogOpen(true);
  };

  const openEdit = (c: Cluster) => {
    setEditingId(c.id);
    setForm({
      name: c.name,
      display_name: c.display_name || '',
      description: c.description || '',
      in_cluster: c.in_cluster,
      kubeconfig: '',
      kubeconfig_path: c.kubeconfig_path || '',
      status: c.status || 'active',
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!form.name) {
      enqueueSnackbar('可观测集群名称必填', { variant: 'warning' });
      return;
    }
    if (!form.in_cluster && !form.kubeconfig && !form.kubeconfig_path) {
      enqueueSnackbar('请至少提供 kubeconfig 文本或路径，或勾选 in-cluster', { variant: 'warning' });
      return;
    }
    setSaving(true);
    try {
      if (editingId) {
        await clusterAPI.update(editingId, {
          display_name: form.display_name,
          description: form.description,
          in_cluster: form.in_cluster,
          kubeconfig: form.kubeconfig || undefined,
          kubeconfig_path: form.kubeconfig_path || undefined,
          status: form.status,
        });
        enqueueSnackbar('可观测集群更新成功', { variant: 'success' });
      } else {
        await clusterAPI.create({
          name: form.name,
          display_name: form.display_name || undefined,
          description: form.description || undefined,
          in_cluster: form.in_cluster,
          kubeconfig: form.kubeconfig || undefined,
          kubeconfig_path: form.kubeconfig_path || undefined,
        });
        enqueueSnackbar('可观测集群注册成功', { variant: 'success' });
      }
      setDialogOpen(false);
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingId ? '更新可观测集群失败' : '注册可观测集群失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.cluster) return;
    try {
      await clusterAPI.delete(deleteDialog.cluster.id);
      enqueueSnackbar('可观测集群删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败（该集群可能绑定了可用区）'), { variant: 'error' });
    }
  };

  if (loading && clusters.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="可观测集群"
        subtitle="管理承载 VictoriaMetrics / Grafana / 告警引擎的 K8s 集群。每个可用区绑定一个可观测集群，在可用区管理中关联。"
        actionLabel={isAdmin ? '注册集群' : undefined}
        onAction={isAdmin ? openCreate : undefined}
      />

      {!isAdmin && (
        <Alert severity="info" sx={{ mb: 2 }}>
          仅管理员可注册/编辑/删除可观测集群。当前仅提供只读视图。
        </Alert>
      )}

      <FilterToolbar>
        <Typography variant="body2" color="text.secondary">
          可观测集群是 Zone 的基础设施底座，注册后需在可用区管理中绑定 Zone 才能部署实例。
        </Typography>
      </FilterToolbar>

      <DataTableCard>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>显示名</TableCell>
                <TableCell>模式</TableCell>
                <TableCell>关联可用区</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {clusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState title="暂无可观测集群" description="点击右上角按钮注册第一个可观测集群，再到可用区管理中完成绑定" />
                  </TableCell>
                </TableRow>
              ) : (
                clusters.map((c) => {
                  const zone = zoneByClusterId[c.id];
                  return (
                    <TableRow key={c.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{c.name}</TableCell>
                      <TableCell sx={{ color: 'text.secondary' }}>{c.display_name || '-'}</TableCell>
                      <TableCell>
                        {c.in_cluster ? (
                          <Typography variant="caption" color="success.main">In-Cluster</Typography>
                        ) : (
                          <Typography variant="caption" color="text.secondary">External</Typography>
                        )}
                      </TableCell>
                      <TableCell>
                        {zone ? (
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {zone.display_name || zone.slug}
                          </Typography>
                        ) : (
                          <Typography variant="caption" color="text.disabled">
                            未绑定
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell><StatusChip status={c.status || 'active'} /></TableCell>
                      <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                        {new Date(c.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell align="right">
                        {isAdmin && (
                          <>
                            <Tooltip title="编辑">
                              <IconButton size="small" onClick={() => openEdit(c)}>
                                <EditOutlinedIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="删除">
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

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingId ? '编辑可观测集群' : '注册可观测集群'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            注册后可观测集群后，需到「可用区管理」中将 Zone 与该集群绑定，才能在此集群上部署实例。
          </Alert>
          <TextField
            fullWidth
            size="small"
            label="可观测集群唯一标识 (name)"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            sx={{ mb: 2 }}
            required
            disabled={!!editingId}
            helperText="小写字母与数字，创建后不可修改。注册后在可用区管理中绑定 Zone"
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
            label="描述"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            sx={{ mb: 2 }}
            multiline
            minRows={2}
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={form.in_cluster}
                onChange={(e) => setForm({ ...form, in_cluster: e.target.checked })}
              />
            }
            label="使用 Pod 内 ServiceAccount（In-Cluster 模式）"
            sx={{ mb: 1 }}
          />
          {!form.in_cluster && (
            <>
              <TextField
                fullWidth
                size="small"
                label="Kubeconfig 文件路径（推荐）"
                value={form.kubeconfig_path}
                onChange={(e) => setForm({ ...form, kubeconfig_path: e.target.value })}
                sx={{ mb: 2 }}
                helperText="容器内可访问的绝对路径，例如 /etc/opsconfig/kubeconfig.yaml"
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
                helperText="目前仅作为展示存档，应用以 Kubeconfig 路径为主"
              />
            </>
          )}
          {editingId && (
            <TextField
              fullWidth
              size="small"
              label="状态"
              value={form.status}
              onChange={(e) => setForm({ ...form, status: e.target.value })}
              sx={{ mb: 2 }}
              helperText="active / inactive"
            />
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.name}>
            {saving ? '保存中...' : editingId ? '更新' : '注册'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除可观测集群"
        message={`确定要删除可观测集群「${deleteDialog.cluster?.name}」吗？如有 Zone 绑定，需先在可用区管理中解除关联。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
