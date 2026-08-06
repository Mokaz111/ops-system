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
  Typography,
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
import { extractApiError } from '../../api';
import { formatMemberships } from '../../utils/membership';
import type { User } from '../../types/api';

const roleLabels: Record<string, { label: string; color: 'primary' | 'secondary' | 'default' }> = {
  admin: { label: '管理员', color: 'primary' },
  user: { label: '普通用户', color: 'default' },
};

export default function UserPage() {
  const { enqueueSnackbar } = useSnackbar();
  type UserForm = {
    username: string;
    email: string;
    phone: string;
    role: User['role'];
    password: string;
  };
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; user?: User }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<UserForm>({ username: '', email: '', phone: '', role: 'user', password: '' });

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

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editingId) {
        await userAPI.update(editingId, {
          username: form.username,
          email: form.email,
          phone: form.phone,
          role: form.role,
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
      <PageHeader title="用户管理" subtitle="管理平台用户与工作空间成员关系" actionLabel="新建用户" onAction={() => { setEditingId(null); setForm({ username: '', email: '', phone: '', role: 'user', password: '' }); setDialogOpen(true); }} />

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
                <TableCell>邮箱</TableCell>
                <TableCell>平台角色</TableCell>
                <TableCell>工作空间成员</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.length === 0 ? (
                <TableRow><TableCell colSpan={7}><EmptyState title="暂无用户" /></TableCell></TableRow>
              ) : users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell sx={{ fontWeight: 500 }}>{u.username}</TableCell>
                  <TableCell sx={{ color: 'text.secondary' }}>{u.email || '-'}</TableCell>
                  <TableCell>
                    <Chip
                      label={roleLabels[u.role]?.label || u.role}
                      size="small"
                      color={roleLabels[u.role]?.color || 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary', maxWidth: 240 }}>
                    <Typography variant="caption" noWrap title={formatMemberships(u)}>
                      {formatMemberships(u)}
                    </Typography>
                  </TableCell>
                  <TableCell><StatusChip status={u.status || 'active'} /></TableCell>
                  <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(u.created_at).toLocaleDateString()}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="编辑">
                      <IconButton size="small" onClick={() => {
                        setEditingId(u.id);
                        setForm({ username: u.username, email: u.email || '', phone: u.phone || '', role: u.role, password: '' });
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
          <TextField fullWidth label="邮箱" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth label="手机号" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} sx={{ mb: 2 }} />
          {!editingId && (
            <TextField fullWidth label="密码" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} sx={{ mb: 2 }} required />
          )}
          <FormControl fullWidth size="small">
            <InputLabel>平台角色</InputLabel>
            <Select value={form.role} label="平台角色" onChange={(e) => setForm({ ...form, role: e.target.value as User['role'] })}>
              <MenuItem value="admin">管理员</MenuItem>
              <MenuItem value="user">普通用户</MenuItem>
            </Select>
          </FormControl>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1.5 }}>
            工作空间成员关系请在工作空间管理页的「成员」标签中配置。
          </Typography>
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
        message={`确定要删除用户「${deleteDialog.user?.username}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
