import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
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
  Typography,
} from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { tenantAPI } from '../../api/tenant';
import type { Tenant } from '../../types/api';
import { useAuthStore } from '../../stores/useAuthStore';

interface FormState {
  name: string;
  scope: 'platform' | 'tenant';
  tenant_id: string;
  url: string;
  admin_user: string;
  admin_token: string;
}

const defaultForm: FormState = {
  name: '',
  scope: 'platform',
  tenant_id: '',
  url: '',
  admin_user: 'admin',
  admin_token: '',
};

export default function GrafanaHostPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const [hosts, setHosts] = useState<GrafanaHost[]>([]);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; host?: GrafanaHost }>({ open: false });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(defaultForm);
  const [saving, setSaving] = useState(false);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const [hostsRes, tenantsRes] = await Promise.all([
        grafanaHostAPI.list({ page: 1, page_size: 100 }),
        tenantAPI.list({ page: 1, page_size: 100 }).catch(() => ({ data: { data: { items: [] } } })),
      ]);
      setHosts(hostsRes.data.data?.items || []);
      setTenants(tenantsRes.data.data?.items || []);
    } catch {
      enqueueSnackbar('获取 Grafana 主机列表失败', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => {
    fetch();
  }, [fetch]);

  const tenantNameById = tenants.reduce<Record<string, string>>((acc, t) => {
    acc[t.id] = t.tenant_name;
    return acc;
  }, {});

  const openCreate = () => {
    setEditingId(null);
    setForm(defaultForm);
    setDialogOpen(true);
  };

  const openEdit = (h: GrafanaHost) => {
    setEditingId(h.id);
    setForm({
      name: h.name,
      scope: h.scope,
      tenant_id: h.tenant_id || '',
      url: h.url,
      admin_user: h.admin_user || 'admin',
      admin_token: '',
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!form.name || !form.url) {
      enqueueSnackbar('名称和 URL 必填', { variant: 'warning' });
      return;
    }
    if (form.scope === 'tenant' && !form.tenant_id) {
      enqueueSnackbar('租户级主机必须选择所属租户', { variant: 'warning' });
      return;
    }
    setSaving(true);
    try {
      if (editingId) {
        await grafanaHostAPI.update(editingId, {
          name: form.name,
          url: form.url,
          admin_user: form.admin_user,
          admin_token: form.admin_token || undefined,
        });
        enqueueSnackbar('Grafana 主机更新成功', { variant: 'success' });
      } else {
        await grafanaHostAPI.create({
          name: form.name,
          scope: form.scope,
          tenant_id: form.scope === 'tenant' ? form.tenant_id : undefined,
          url: form.url,
          admin_user: form.admin_user,
          admin_token: form.admin_token || undefined,
        });
        enqueueSnackbar('Grafana 主机登记成功', { variant: 'success' });
      }
      setDialogOpen(false);
      fetch();
    } catch {
      enqueueSnackbar(editingId ? '更新失败' : '创建失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.host) return;
    try {
      await grafanaHostAPI.delete(deleteDialog.host.id);
      enqueueSnackbar('Grafana 主机删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetch();
    } catch {
      enqueueSnackbar('删除失败', { variant: 'error' });
    }
  };

  if (loading && hosts.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="Grafana 主机"
        subtitle="登记平台或租户自建的 Grafana 实例；安装模板时可选择目标 Grafana"
        actionLabel={isAdmin ? '登记 Grafana 主机' : undefined}
        onAction={isAdmin ? openCreate : undefined}
      />

      {!isAdmin && (
        <Alert severity="info" sx={{ mb: 2 }}>
          仅管理员可登记/编辑/删除 Grafana 主机。当前仅提供只读视图。
        </Alert>
      )}

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>范围</TableCell>
                <TableCell>所属租户</TableCell>
                <TableCell>地址</TableCell>
                <TableCell>管理员</TableCell>
                <TableCell>状态</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {hosts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState title="暂无 Grafana 主机" description="未登记时将使用 config.yaml 中的平台默认 Grafana" />
                  </TableCell>
                </TableRow>
              ) : (
                hosts.map((h) => (
                  <TableRow key={h.id}>
                    <TableCell sx={{ fontWeight: 500 }}>{h.name}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={h.scope === 'platform' ? '平台' : '租户'}
                        color={h.scope === 'platform' ? 'primary' : 'secondary'}
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary' }}>
                      {h.tenant_id ? (tenantNameById[h.tenant_id] || h.tenant_id.slice(0, 8)) : '-'}
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>
                      <Typography
                        component="a"
                        href={h.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, color: 'primary.main', textDecoration: 'none', fontSize: '0.8125rem' }}
                      >
                        {h.url}
                        <OpenInNewIcon sx={{ fontSize: 14 }} />
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ fontSize: '0.8125rem' }}>{h.admin_user || '-'}</TableCell>
                    <TableCell><StatusChip status={h.status || 'active'} /></TableCell>
                    <TableCell align="right">
                      {isAdmin && (
                        <>
                          <Tooltip title="编辑">
                            <IconButton size="small" onClick={() => openEdit(h)}>
                              <EditOutlinedIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="删除">
                            <IconButton
                              size="small"
                              color="error"
                              onClick={() => setDeleteDialog({ open: true, host: h })}
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

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingId ? '编辑 Grafana 主机' : '登记 Grafana 主机'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField
            fullWidth
            size="small"
            label="名称"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            sx={{ mb: 2 }}
            required
          />
          <FormControl fullWidth size="small" sx={{ mb: 2 }} disabled={!!editingId}>
            <InputLabel>范围</InputLabel>
            <Select
              label="范围"
              value={form.scope}
              onChange={(e) => setForm({ ...form, scope: e.target.value as 'platform' | 'tenant' })}
            >
              <MenuItem value="platform">平台共享</MenuItem>
              <MenuItem value="tenant">租户专属</MenuItem>
            </Select>
          </FormControl>
          {form.scope === 'tenant' && (
            <FormControl fullWidth size="small" sx={{ mb: 2 }} disabled={!!editingId}>
              <InputLabel>所属租户</InputLabel>
              <Select
                label="所属租户"
                value={form.tenant_id}
                onChange={(e) => setForm({ ...form, tenant_id: e.target.value })}
              >
                <MenuItem value="">请选择</MenuItem>
                {tenants.map((t) => (
                  <MenuItem key={t.id} value={t.id}>
                    {t.tenant_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
          <TextField
            fullWidth
            size="small"
            label="URL"
            value={form.url}
            onChange={(e) => setForm({ ...form, url: e.target.value })}
            sx={{ mb: 2 }}
            required
            placeholder="http://grafana.monitoring.svc.cluster.local:3000"
          />
          <TextField
            fullWidth
            size="small"
            label="管理员账号"
            value={form.admin_user}
            onChange={(e) => setForm({ ...form, admin_user: e.target.value })}
            sx={{ mb: 2 }}
          />
          <TextField
            fullWidth
            size="small"
            label={editingId ? 'API Token（留空保持原值）' : 'API Token'}
            value={form.admin_token}
            onChange={(e) => setForm({ ...form, admin_token: e.target.value })}
            sx={{ mb: 1 }}
            type="password"
            helperText="建议使用 Grafana Service Account Token；数据库中将加密存储"
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={saving || !form.name || !form.url}
          >
            {saving ? '保存中...' : editingId ? '更新' : '登记'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除 Grafana 主机"
        message={`确定要删除 Grafana 主机「${deleteDialog.host?.name}」吗？关联的模板安装将回退到平台默认 Grafana。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
