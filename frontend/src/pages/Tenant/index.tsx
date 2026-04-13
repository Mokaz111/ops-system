import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Card,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Button,
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
  TablePagination,
  TableRow,
  TextField,
  Tooltip,
  InputAdornment,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { tenantAPI } from '../../api/tenant';
import { departmentAPI } from '../../api/department';
import type { Department, Tenant } from '../../types/api';

const templateLabels: Record<string, string> = {
  shared: '共享版',
  dedicated_single: '独享单节点',
  dedicated_cluster: '独享集群',
};

export default function TenantPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; tenant?: Tenant }>({ open: false });
  const [form, setForm] = useState({ tenant_name: '', dept_id: '', template_type: 'shared' });
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const fetchTenants = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await tenantAPI.list({ page: page + 1, page_size: pageSize, search });
      setTenants(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch {
      enqueueSnackbar('获取租户列表失败', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, enqueueSnackbar]);

  useEffect(() => { fetchTenants(); }, [fetchTenants]);

  useEffect(() => {
    const fetchDepartments = async () => {
      try {
        const { data: res } = await departmentAPI.tree();
        const flat = (rows: Department[]): Department[] =>
          rows.flatMap((dept) => [dept, ...(dept.children ? flat(dept.children) : [])]);
        setDepartments(flat(res.data || []));
      } catch {
        enqueueSnackbar('获取部门列表失败', { variant: 'warning' });
      }
    };
    fetchDepartments();
  }, [enqueueSnackbar]);

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editingId) {
        await tenantAPI.update(editingId, form);
        enqueueSnackbar('租户更新成功', { variant: 'success' });
      } else {
        await tenantAPI.create(form);
        enqueueSnackbar('租户创建成功', { variant: 'success' });
      }
      setDialogOpen(false);
      setEditingId(null);
      setForm({ tenant_name: '', dept_id: '', template_type: 'shared' });
      fetchTenants();
    } catch {
      enqueueSnackbar(editingId ? '更新失败' : '创建失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.tenant) return;
    try {
      await tenantAPI.delete(deleteDialog.tenant.id);
      enqueueSnackbar('租户删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchTenants();
    } catch {
      enqueueSnackbar('删除失败', { variant: 'error' });
    }
  };

  const openEdit = (tenant: Tenant) => {
    setEditingId(tenant.id);
    setForm({ tenant_name: tenant.tenant_name, dept_id: tenant.dept_id, template_type: tenant.template_type });
    setDialogOpen(true);
  };

  if (loading && tenants.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="租户管理" subtitle="管理平台所有租户及其资源配置" actionLabel="新建租户" onAction={() => { setEditingId(null); setForm({ tenant_name: '', dept_id: '', template_type: 'shared' }); setDialogOpen(true); }} />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2, display: 'flex', gap: 2 }}>
          <TextField
            placeholder="搜索租户..."
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
                <TableCell>租户名称</TableCell>
                <TableCell>VMUser ID</TableCell>
                <TableCell>模板类型</TableCell>
                <TableCell>Grafana Org</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {tenants.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState title="暂无租户" description="点击右上角按钮创建第一个租户" />
                  </TableCell>
                </TableRow>
              ) : (
                tenants.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell sx={{ fontWeight: 500 }}>{t.tenant_name}</TableCell>
                    <TableCell>
                      <Chip label={t.vmuser_id || '-'} size="small" variant="outlined" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }} />
                    </TableCell>
                    <TableCell>
                      <Chip label={templateLabels[t.template_type] || t.template_type} size="small" color="info" variant="outlined" />
                    </TableCell>
                    <TableCell>{t.grafana_org_id || '-'}</TableCell>
                    <TableCell><StatusChip status={t.status} /></TableCell>
                    <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(t.created_at).toLocaleDateString()}</TableCell>
                    <TableCell align="right">
                      <Tooltip title="详情功能开发中">
                        <span>
                          <IconButton size="small" disabled aria-label="租户详情（开发中）">
                            <VisibilityOutlinedIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="编辑">
                        <IconButton size="small" onClick={() => openEdit(t)} aria-label="编辑租户">
                          <EditOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="删除">
                        <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, tenant: t })} aria-label="删除租户">
                          <DeleteOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
        {total > 0 && (
          <TablePagination
            component="div"
            count={total}
            page={page}
            onPageChange={(_, p) => setPage(p)}
            rowsPerPage={pageSize}
            onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value)); setPage(0); }}
            rowsPerPageOptions={[10, 20, 50]}
            labelRowsPerPage="每页行数"
          />
        )}
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingId ? '编辑租户' : '新建租户'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="租户名称" value={form.tenant_name} onChange={(e) => setForm({ ...form, tenant_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>所属部门</InputLabel>
            <Select
              value={form.dept_id}
              label="所属部门"
              onChange={(e) => setForm({ ...form, dept_id: e.target.value })}
            >
              <MenuItem value="">请选择部门</MenuItem>
              {departments.map((dept) => (
                <MenuItem key={dept.id} value={dept.id}>
                  {dept.dept_name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl fullWidth size="small">
            <InputLabel>模板类型</InputLabel>
            <Select value={form.template_type} label="模板类型" onChange={(e) => setForm({ ...form, template_type: e.target.value })}>
              <MenuItem value="shared">共享版 (全局 VMCluster)</MenuItem>
              <MenuItem value="dedicated_single">独享单节点版 (VMSingle)</MenuItem>
              <MenuItem value="dedicated_cluster">独享集群版 (VMCluster)</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.tenant_name}>
            {saving ? '保存中...' : editingId ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除租户"
        message={`确定要删除租户「${deleteDialog.tenant?.tenant_name}」吗？此操作不可撤销。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
