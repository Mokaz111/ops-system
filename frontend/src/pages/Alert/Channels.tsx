import { useCallback, useEffect, useState } from 'react';
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
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import FilterToolbar from '../../components/common/FilterToolbar';
import { alertAPI } from '../../api/alert';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import type { NotificationChannel } from '../../types/api';
import { channelTypeLabels, useWorkspaceOptions, WorkspaceFilterSelect } from './shared';

export default function AlertChannelsPage() {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const { workspaces, tenantName } = useWorkspaceOptions();
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(true);

  const globalWorkspaceId = useWorkspaceStore((s) => s.currentId);
  const [tenantFilter, setTenantFilter] = useState(globalWorkspaceId);
  useEffect(() => { setTenantFilter(globalWorkspaceId); }, [globalWorkspaceId]);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<NotificationChannel | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; channel?: NotificationChannel }>({ open: false });
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    workspace_id: '',
    channel_name: '',
    channel_type: 'webhook',
    config: '{}',
    enabled: true,
  });

  const fetchChannels = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await alertAPI.listChannels({ page: 1, page_size: 100, workspace_id: tenantFilter || undefined });
      setChannels(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取通知渠道失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar, tenantFilter]);

  useEffect(() => { fetchChannels(); }, [fetchChannels]);

  const openCreate = () => {
    setEditing(null);
    setForm({ workspace_id: tenantFilter || '', channel_name: '', channel_type: 'webhook', config: '{}', enabled: true });
    setDialogOpen(true);
  };

  const openEdit = (ch: NotificationChannel) => {
    setEditing(ch);
    setForm({
      workspace_id: ch.tenant_id,
      channel_name: ch.channel_name,
      channel_type: ch.channel_type,
      config: ch.config || '{}',
      enabled: ch.enabled,
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editing) {
        await alertAPI.updateChannel(editing.id, {
          channel_name: form.channel_name,
          channel_type: form.channel_type,
          config: form.config,
          enabled: form.enabled,
        });
        enqueueSnackbar('通知渠道已更新', { variant: 'success' });
      } else {
        await alertAPI.createChannel({
          tenant_id: form.workspace_id,
          channel_name: form.channel_name,
          channel_type: form.channel_type,
          config: form.config,
          enabled: form.enabled,
        });
        enqueueSnackbar('通知渠道已创建', { variant: 'success' });
      }
      setDialogOpen(false);
      fetchChannels();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editing ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.channel) return;
    try {
      await alertAPI.deleteChannel(deleteDialog.channel.id);
      enqueueSnackbar('通知渠道已删除', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchChannels();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  if (loading && channels.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="通知渠道"
        subtitle="告警规则触发后的通知目标（Webhook / 邮件 / 钉钉 / Slack / 短信）"
        actionLabel={isAdmin ? '新建渠道' : undefined}
        onAction={isAdmin ? openCreate : undefined}
      />

      <FilterToolbar>
        <Button startIcon={<ArrowBackIcon />} onClick={() => navigate('/alerts/rules')}>
          返回告警规则
        </Button>
        <Box sx={{ flex: 1 }} />
        <WorkspaceFilterSelect value={tenantFilter} onChange={setTenantFilter} workspaces={workspaces} />
      </FilterToolbar>

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>工作空间</TableCell>
                <TableCell>类型</TableCell>
                <TableCell>状态</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {channels.length === 0 ? (
                <TableRow><TableCell colSpan={5}><EmptyState title="暂无通知渠道" description="创建通知渠道供告警规则绑定" /></TableCell></TableRow>
              ) : channels.map((ch) => (
                <TableRow key={ch.id}>
                  <TableCell sx={{ fontWeight: 600 }}>{ch.channel_name}</TableCell>
                  <TableCell>{tenantName(ch.tenant_id)}</TableCell>
                  <TableCell>{channelTypeLabels[ch.channel_type] || ch.channel_type}</TableCell>
                  <TableCell>
                    <Chip size="small" label={ch.enabled ? '启用' : '停用'} color={ch.enabled ? 'success' : 'default'} variant="outlined" />
                  </TableCell>
                  <TableCell align="right">
                    {isAdmin && (
                      <>
                        <Tooltip title="编辑">
                          <IconButton size="small" onClick={() => openEdit(ch)}>
                            <EditOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="删除">
                          <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, channel: ch })}>
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editing ? '编辑通知渠道' : '新建通知渠道'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>工作空间</InputLabel>
            <Select value={form.workspace_id} label="工作空间" onChange={(e) => setForm({ ...form, workspace_id: e.target.value })} disabled={!!editing}>
              {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
            </Select>
          </FormControl>
          <TextField fullWidth size="small" label="渠道名称" value={form.channel_name} onChange={(e) => setForm({ ...form, channel_name: e.target.value })} sx={{ mb: 2 }} />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>类型</InputLabel>
            <Select value={form.channel_type} label="类型" onChange={(e) => setForm({ ...form, channel_type: e.target.value })}>
              <MenuItem value="webhook">Webhook</MenuItem>
              <MenuItem value="email">邮件</MenuItem>
              <MenuItem value="dingtalk">钉钉</MenuItem>
              <MenuItem value="slack">Slack</MenuItem>
              <MenuItem value="sms">短信</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth multiline minRows={4} size="small" label="配置 (JSON)" value={form.config} onChange={(e) => setForm({ ...form, config: e.target.value })} sx={{ fontFamily: 'monospace' }} />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" disabled={saving || !form.workspace_id || !form.channel_name} onClick={handleSave}>
            {saving ? '保存中...' : editing ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除通知渠道"
        message={`确定要删除渠道「${deleteDialog.channel?.channel_name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
