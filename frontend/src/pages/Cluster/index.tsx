import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
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
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';

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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; cluster?: Cluster }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(defaultForm);
  const [saving, setSaving] = useState(false);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await clusterAPI.list({ page: 1, page_size: 100 });
      setClusters(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取集群列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => {
    fetch();
  }, [fetch]);

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
      enqueueSnackbar('集群名称必填', { variant: 'warning' });
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
        enqueueSnackbar('集群更新成功', { variant: 'success' });
      } else {
        await clusterAPI.create({
          name: form.name,
          display_name: form.display_name || undefined,
          description: form.description || undefined,
          in_cluster: form.in_cluster,
          kubeconfig: form.kubeconfig || undefined,
          kubeconfig_path: form.kubeconfig_path || undefined,
        });
        enqueueSnackbar('集群注册成功', { variant: 'success' });
      }
      setDialogOpen(false);
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingId ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.cluster) return;
    try {
      await clusterAPI.delete(deleteDialog.cluster.id);
      enqueueSnackbar('集群删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败（该集群可能仍有实例在使用）'), { variant: 'error' });
    }
  };

  if (loading && clusters.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="集群管理"
        subtitle="注册并管理作为监控目标的 Kubernetes 集群；实例通过 cluster_id 绑定目标集群"
        actionLabel={isAdmin ? '注册集群' : undefined}
        onAction={isAdmin ? openCreate : undefined}
      />

      {!isAdmin && (
        <Alert severity="info" sx={{ mb: 2 }}>
          仅管理员可注册/编辑/删除集群。当前仅提供只读视图。
        </Alert>
      )}

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>显示名</TableCell>
                <TableCell>模式</TableCell>
                <TableCell>Kubeconfig 路径</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {clusters.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState title="暂无集群" description="点击右上角按钮注册第一个目标集群（或继续使用平台默认集群）" />
                  </TableCell>
                </TableRow>
              ) : (
                clusters.map((c) => (
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
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                      {c.kubeconfig_path || '-'}
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
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingId ? '编辑集群' : '注册集群'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField
            fullWidth
            size="small"
            label="集群唯一标识 (name)"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            sx={{ mb: 2 }}
            required
            disabled={!!editingId}
            helperText="小写字母与数字，创建后不可修改"
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
        title="删除集群"
        message={`确定要删除集群「${deleteDialog.cluster?.name}」吗？关联的实例将回退到平台默认集群。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
