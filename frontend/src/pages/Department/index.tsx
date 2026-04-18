import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Button,
  Card,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  FormControl,
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
import SearchIcon from '@mui/icons-material/Search';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { departmentAPI } from '../../api/department';
import { extractApiError } from '../../api';
import type { Department } from '../../types/api';

export default function DepartmentPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; dept?: Department }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState({ dept_name: '', parent_id: '' });
  const [saving, setSaving] = useState(false);
  const deptNameById = departments.reduce<Record<string, string>>((acc, dept) => {
    acc[dept.id] = dept.dept_name;
    return acc;
  }, {});

  const fetchDepartments = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await departmentAPI.list({ search });
      setDepartments(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取部门列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [search, enqueueSnackbar]);

  useEffect(() => { fetchDepartments(); }, [fetchDepartments]);

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editingId) {
        await departmentAPI.update(editingId, { dept_name: form.dept_name, parent_id: form.parent_id || null });
        enqueueSnackbar('部门更新成功', { variant: 'success' });
      } else {
        await departmentAPI.create({ dept_name: form.dept_name, parent_id: form.parent_id || null });
        enqueueSnackbar('部门创建成功', { variant: 'success' });
      }
      setDialogOpen(false);
      setEditingId(null);
      fetchDepartments();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingId ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.dept) return;
    try {
      await departmentAPI.delete(deleteDialog.dept.id);
      enqueueSnackbar('部门删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchDepartments();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  if (loading && departments.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="部门管理" subtitle="管理组织部门结构" actionLabel="新建部门" onAction={() => { setEditingId(null); setForm({ dept_name: '', parent_id: '' }); setDialogOpen(true); }} />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2 }}>
          <TextField
            placeholder="搜索部门..."
            size="small"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
            sx={{ width: 280 }}
          />
        </Box>
      </Card>

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>部门ID</TableCell>
                <TableCell>部门名称</TableCell>
                <TableCell>上级部门</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {departments.length === 0 ? (
                <TableRow><TableCell colSpan={6}><EmptyState title="暂无部门" /></TableCell></TableRow>
              ) : departments.map((d) => (
                <TableRow key={d.id}>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}>
                    {d.id}
                  </TableCell>
                  <TableCell sx={{ fontWeight: 500 }}>{d.dept_name}</TableCell>
                  <TableCell sx={{ color: 'text.secondary' }}>
                    {d.parent_id ? (deptNameById[d.parent_id] || '-') : '-'}
                  </TableCell>
                  <TableCell><StatusChip status={d.status || 'active'} /></TableCell>
                  <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(d.created_at).toLocaleDateString()}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="编辑">
                      <IconButton size="small" onClick={() => { setEditingId(d.id); setForm({ dept_name: d.dept_name, parent_id: d.parent_id || '' }); setDialogOpen(true); }}>
                        <EditOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="删除">
                      <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, dept: d })}>
                        <DeleteOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingId ? '编辑部门' : '新建部门'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="部门名称" value={form.dept_name} onChange={(e) => setForm({ ...form, dept_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <FormControl fullWidth size="small">
            <InputLabel>上级部门（可选）</InputLabel>
            <Select
              value={form.parent_id}
              label="上级部门（可选）"
              onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
            >
              <MenuItem value="">无（作为顶级部门）</MenuItem>
              {departments
                .filter((dept) => dept.id !== editingId)
                .map((dept) => (
                  <MenuItem key={dept.id} value={dept.id}>
                    {dept.dept_name}
                  </MenuItem>
                ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.dept_name}>
            {saving ? '保存中...' : editingId ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除部门"
        message={`确定要删除部门「${deleteDialog.dept?.dept_name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
