import { useCallback, useEffect, useState } from 'react';
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
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { userAPI } from '../../api/user';
import { workspaceAPI } from '../../api/workspace';
import { extractApiError } from '../../api';
import type { Workspace, User } from '../../types/api';

const roleLabels: Record<string, { label: string; color: 'primary' | 'secondary' | 'default' }> = {
  admin: { label: '管理员', color: 'primary' },
  user: { label: '普通用户', color: 'default' },
};

export default function UserPage() {
  const { enqueueSnackbar } = useSnackbar();
  type UserForm = {
    username: string;
    display_name: string;
    email: string;
    phone: string;
    role: User['role'];
    workspace_id: string;
    password: string;
  };
  const [users, setUsers] = useState<User[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; user?: User }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<UserForm>({ username: '', display_name: '', email: '', phone: '', role: 'user', workspace_id: '', password: '' });

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await userAPI.list({ page: page + 1, page_size: pageSize, search });
      setUsers(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取用户列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, enqueueSnackbar]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await workspaceAPI.list({ page: 1, page_size: 200 });
        setWorkspaces(res.data?.items || []);
      } catch {
        /* tenant binding is optional on this page */
      }
    })();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editingId) {
        await userAPI.update(editingId, {
          username: form.username,
          display_name: form.display_name,
          email: form.email,
          phone: form.phone,
          role: form.role,
          workspace_id: form.workspace_id || null,
        });
        enqueueSnackbar('用户更新成功', { variant: 'success' });
      } else {
        await userAPI.create(form);
        enqueueSnackbar('用户创建成功', { variant: 'success' });
      }
      setDialogOpen(false);
      setEditingId(null);
      fetchUsers();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingId ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.user) return;
    try {
      await userAPI.delete(deleteDialog.user.id);
      enqueueSnackbar('用户删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchUsers();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  if (loading && users.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="用户管理" subtitle="管理平台用户、工作空间成员角色与服务访问边界" actionLabel="新建用户" onAction={() => { setEditingId(null); setForm({ username: '', display_name: '', email: '', phone: '', role: 'user', workspace_id: '', password: '' }); setDialogOpen(true); }} />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2 }}>
          <TextField
            placeholder="搜索用户..."
            size="small"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
            sx={{ width: 280 }}
          />
        </Box>
      </Card>

      <Card>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>用户名</TableCell>
                <TableCell>显示名</TableCell>
                <TableCell>邮箱</TableCell>
                <TableCell>角色</TableCell>
                <TableCell>绑定工作空间</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.length === 0 ? (
                <TableRow><TableCell colSpan={8}><EmptyState title="暂无用户" /></TableCell></TableRow>
              ) : users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell sx={{ fontWeight: 500 }}>{u.username}</TableCell>
                  <TableCell>{u.display_name || '-'}</TableCell>
                  <TableCell sx={{ color: 'text.secondary' }}>{u.email || '-'}</TableCell>
                  <TableCell>
                    <Chip
                      label={roleLabels[u.role]?.label || u.role}
                      size="small"
                      color={roleLabels[u.role]?.color || 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{u.workspace_id ? (workspaces.find((t) => t.id === u.workspace_id)?.workspace_name || u.workspace_id.slice(0, 8)) : '-'}</TableCell>
                  <TableCell><StatusChip status={u.status || 'active'} /></TableCell>
                  <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(u.created_at).toLocaleDateString()}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="编辑">
                      <IconButton size="small" onClick={() => {
                        setEditingId(u.id);
                        setForm({ username: u.username, display_name: u.display_name || '', email: u.email || '', phone: u.phone || '', role: u.role, workspace_id: u.workspace_id || '', password: '' });
                        setDialogOpen(true);
                      }}>
                        <EditOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="删除">
                      <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, user: u })}>
                        <DeleteOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
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

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingId ? '编辑用户' : '新建用户'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="用户名" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} sx={{ mb: 2 }} required disabled={!!editingId} />
          <TextField fullWidth label="显示名" value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth label="邮箱" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth label="手机号" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} sx={{ mb: 2 }} />
          {!editingId && (
            <TextField fullWidth label="密码" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} sx={{ mb: 2 }} required />
          )}
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>角色</InputLabel>
            <Select value={form.role} label="角色" onChange={(e) => setForm({ ...form, role: e.target.value as User['role'] })}>
              <MenuItem value="admin">管理员</MenuItem>
              <MenuItem value="user">普通用户</MenuItem>
            </Select>
          </FormControl>
          <FormControl fullWidth size="small">
            <InputLabel>绑定工作空间（可选）</InputLabel>
            <Select value={form.workspace_id} label="绑定工作空间（可选）" onChange={(e) => setForm({ ...form, workspace_id: e.target.value })}>
              <MenuItem value="">平台级用户</MenuItem>
              {workspaces.map((tenant) => (
                <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.username}>
            {saving ? '保存中...' : editingId ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除用户"
        message={`确定要删除用户「${deleteDialog.user?.display_name || deleteDialog.user?.username}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
