import { useCallback, useEffect, useMemo, useState } from 'react';
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
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import DataTableCard from '../../components/common/DataTableCard';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import { ssoLoginToGrafana } from '../../api/grafanaSso';
import { useAuthStore } from '../../stores/useAuthStore';

const sourceFilterItems = [
  { key: '', label: '全部来源' },
  { key: 'platform', label: '平台共享' },
  { key: 'external', label: '外部登记' },
];

interface HostFormState {
  name: string;
  source: 'platform' | 'external';
  url: string;
  admin_user: string;
  admin_password: string;
  admin_token: string;
}

const defaultHostForm: HostFormState = {
  name: '',
  source: 'external',
  url: '',
  admin_user: 'admin',
  admin_password: '',
  admin_token: '',
};

export default function GrafanaInstancePage() {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const [hosts, setHosts] = useState<GrafanaInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [sourceFilter, setSourceFilter] = useState('');
  const [saving, setSaving] = useState(false);

  const [hostDialogOpen, setHostDialogOpen] = useState(false);
  const [hostEditingId, setHostEditingId] = useState<string | null>(null);
  const [hostForm, setHostForm] = useState<HostFormState>(defaultHostForm);
  const [hostDeleteDialog, setHostDeleteDialog] = useState<{ open: boolean; host?: GrafanaInstance }>({ open: false });

  const fetchHosts = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await grafanaInstanceAPI.list({
        page: page + 1,
        page_size: pageSize,
        source: sourceFilter || undefined,
      });
      const items = res.data?.items || [];
      setHosts(items);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取 Grafana 实例列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, sourceFilter, enqueueSnackbar]);

  useEffect(() => { fetchHosts(); }, [fetchHosts]);

  const filteredHosts = useMemo(() => {
    if (!search.trim()) return hosts;
    const q = search.trim().toLowerCase();
    return hosts.filter((h) =>
      h.name.toLowerCase().includes(q) || h.url.toLowerCase().includes(q),
    );
  }, [hosts, search]);

  const stats = useMemo(() => ({
    total: total,
    platform: hosts.filter((h) => h.source === 'platform').length,
    external: hosts.filter((h) => h.source === 'external').length,
    active: hosts.filter((h) => h.status === 'active').length,
  }), [hosts, total]);

  const openHostCreate = () => {
    setHostEditingId(null);
    setHostForm(defaultHostForm);
    setHostDialogOpen(true);
  };

  const openHostEdit = (h: GrafanaInstance) => {
    setHostEditingId(h.id);
    setHostForm({
      name: h.name,
      source: h.source,
      url: h.url,
      admin_user: h.admin_user || 'admin',
      admin_password: '',
      admin_token: '',
    });
    setHostDialogOpen(true);
  };

  const handleHostSave = async () => {
    if (!hostForm.name || !hostForm.url) {
      enqueueSnackbar('名称和 URL 必填', { variant: 'warning' });
      return;
    }
    setSaving(true);
    try {
      if (hostEditingId) {
        await grafanaInstanceAPI.update(hostEditingId, {
          name: hostForm.name,
          url: hostForm.url,
          admin_user: hostForm.admin_user,
          admin_password: hostForm.admin_password || undefined,
          admin_token: hostForm.admin_token || undefined,
        });
        enqueueSnackbar('实例更新成功', { variant: 'success' });
      } else {
        await grafanaInstanceAPI.create({
          name: hostForm.name,
          source: hostForm.source,
          url: hostForm.url,
          admin_user: hostForm.admin_user,
          admin_password: hostForm.admin_password || undefined,
          admin_token: hostForm.admin_token || undefined,
        });
        enqueueSnackbar('实例登记成功', { variant: 'success' });
      }
      setHostDialogOpen(false);
      fetchHosts();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, hostEditingId ? '更新失败' : '登记失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleHostDelete = async () => {
    if (!hostDeleteDialog.host) return;
    setSaving(true);
    try {
      await grafanaInstanceAPI.delete(hostDeleteDialog.host.id);
      enqueueSnackbar('实例删除成功', { variant: 'success' });
      setHostDeleteDialog({ open: false });
      fetchHosts();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading && hosts.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="Grafana 实例"
        subtitle="登记与管理 Grafana 连接信息，支持 SSO 一键登录"
        actionLabel={isAdmin ? '登记实例' : undefined}
        onAction={isAdmin ? openHostCreate : undefined}
      />

      <Grid container spacing={1.5} sx={{ mb: 2 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <Card variant="outlined" sx={{ p: 1.5 }}>
            <Typography variant="caption" color="text.secondary">实例总数</Typography>
            <Typography variant="h6">{stats.total}</Typography>
          </Card>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Card variant="outlined" sx={{ p: 1.5 }}>
            <Typography variant="caption" color="text.secondary">平台共享</Typography>
            <Typography variant="h6" color="primary.main">{stats.platform}</Typography>
          </Card>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Card variant="outlined" sx={{ p: 1.5 }}>
            <Typography variant="caption" color="text.secondary">外部登记</Typography>
            <Typography variant="h6">{stats.external}</Typography>
          </Card>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Card variant="outlined" sx={{ p: 1.5 }}>
            <Typography variant="caption" color="text.secondary">活跃</Typography>
            <Typography variant="h6" color="success.main">{stats.active}</Typography>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <FilterToolbar>
          <TextField
            placeholder="搜索名称或 URL..."
            size="small"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
            sx={{ width: 280 }}
          />
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <InputLabel>来源</InputLabel>
            <Select value={sourceFilter} label="来源" onChange={(e) => { setSourceFilter(e.target.value); setPage(0); }}>
              {sourceFilterItems.map((item) => <MenuItem key={item.key || 'all'} value={item.key}>{item.label}</MenuItem>)}
            </Select>
          </FormControl>
        </FilterToolbar>

        <DataTableCard
          pagination={total > 0 ? (
            <TablePagination
              component="div"
              count={total}
              page={page}
              onPageChange={(_, np) => setPage(np)}
              rowsPerPage={pageSize}
              onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }}
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
                  <TableCell>来源</TableCell>
                  <TableCell>URL</TableCell>
                  <TableCell>可用区</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredHosts.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6}>
                      <EmptyState
                        title="暂无 Grafana 实例"
                        description={isAdmin ? '点击右上角「登记实例」添加第一个 Grafana 连接' : '暂无可用 Grafana 实例'}
                      />
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredHosts.map((h) => (
                    <TableRow key={h.id}>
                      <TableCell sx={{ fontWeight: 500 }}>{h.name}</TableCell>
                      <TableCell>
                        <Chip
                          label={h.source === 'platform' ? '平台共享' : '外部登记'}
                          size="small"
                          color={h.source === 'platform' ? 'primary' : 'default'}
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {h.url}
                      </TableCell>
                      <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                        {h.zone_id ? h.zone_id.slice(0, 8) : '-'}
                      </TableCell>
                      <TableCell><StatusChip status={h.status || 'active'} /></TableCell>
                      <TableCell align="right">
                        <Tooltip title="详情">
                          <IconButton size="small" onClick={() => navigate(`/grafana-instances/${h.id}?type=managed`)}>
                            <InfoOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="登录">
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={() => {
                              ssoLoginToGrafana(grafanaInstanceAPI.login(h.id)).catch((err) =>
                                enqueueSnackbar(extractApiError(err, '获取登录信息失败'), { variant: 'error' }),
                              );
                            }}
                          >
                            <LoginOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        {isAdmin && (
                          <>
                            <Tooltip title="编辑">
                              <IconButton size="small" onClick={() => openHostEdit(h)}>
                                <EditOutlinedIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="删除">
                              <IconButton size="small" color="error" onClick={() => setHostDeleteDialog({ open: true, host: h })}>
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
        </DataTableCard>
      </Card>

      <Dialog open={hostDialogOpen} onClose={() => setHostDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{hostEditingId ? '编辑 Grafana 实例' : '登记 Grafana 实例'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="名称" value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }} disabled={!!hostEditingId}>
            <InputLabel>来源</InputLabel>
            <Select label="来源" value={hostForm.source} onChange={(e) => setHostForm({ ...hostForm, source: e.target.value as 'platform' | 'external' })}>
              <MenuItem value="platform">平台共享</MenuItem>
              <MenuItem value="external">外部登记</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth size="small" label="URL" value={hostForm.url} onChange={(e) => setHostForm({ ...hostForm, url: e.target.value })} sx={{ mb: 2 }} required placeholder="http://grafana.monitoring.svc.cluster.local:3000" />
          <TextField fullWidth size="small" label="管理员账号" value={hostForm.admin_user} onChange={(e) => setHostForm({ ...hostForm, admin_user: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth size="small" label={hostEditingId ? '管理员密码（留空保持原值）' : '管理员密码'} value={hostForm.admin_password} onChange={(e) => setHostForm({ ...hostForm, admin_password: e.target.value })} sx={{ mb: 2 }} type="password" helperText="Grafana Admin 登陆密码，用于一键 SSO 登录" />
          <TextField fullWidth size="small" label={hostEditingId ? 'API Token（留空保持原值）' : 'API Token'} value={hostForm.admin_token} onChange={(e) => setHostForm({ ...hostForm, admin_token: e.target.value })} sx={{ mb: 1 }} type="password" helperText="建议使用 Grafana Service Account Token；数据库中将加密存储" />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setHostDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleHostSave} disabled={saving || !hostForm.name || !hostForm.url}>
            {saving ? '保存中...' : hostEditingId ? '更新' : '登记'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={hostDeleteDialog.open}
        title="删除 Grafana 实例"
        message={`确定要删除实例「${hostDeleteDialog.host?.name}」吗？关联的模板安装将回退到平台默认 Grafana。`}
        severity="error"
        confirmLabel="删除"
        loading={saving}
        onConfirm={handleHostDelete}
        onCancel={() => setHostDeleteDialog({ open: false })}
      />
    </Box>
  );
}
