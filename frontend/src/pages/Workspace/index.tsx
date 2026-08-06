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
  Alert,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Tab,
  Tabs,
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
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import PeopleOutlinedIcon from '@mui/icons-material/PeopleOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { workspaceAPI } from '../../api/workspace';
import { userAPI } from '../../api/user';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { isWorkspaceAdmin } from '../../utils/membership';
import type { User, Workspace, WorkspaceMember, WorkspaceMetrics } from '../../types/api';

const templateLabels: Record<string, string> = {
  shared: '共享版',
};

const isolationLabels: Record<string, string> = {
  shared: '共享',
};

const memberRoleLabels: Record<string, string> = {
  admin: '管理员',
  member: '成员',
  viewer: '只读',
};

export default function WorkspacePage() {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  // 工作空间 CRUD 是平台管理员专属；成员管理放开给该空间的 workspace admin（与后端一致）。
  const canManageMembersOf = (workspaceId?: string) =>
    isAdmin || (!!workspaceId && isWorkspaceAdmin(user, workspaceId));
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; tenant?: Workspace }>({ open: false });
  const [form, setForm] = useState({ workspace_name: '', template_type: 'shared', grafana_instance_id: '' });
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [detail, setDetail] = useState<{ open: boolean; tenant?: Workspace; metrics?: WorkspaceMetrics; loading?: boolean; tab?: 'overview' | 'members' }>({ open: false });

  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [memberForm, setMemberForm] = useState({ user_id: '', role: 'member' });
  const [memberSaving, setMemberSaving] = useState(false);
  const [removeMemberDialog, setRemoveMemberDialog] = useState<{ open: boolean; member?: WorkspaceMember }>({ open: false });

  const fetchWorkspaces = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await workspaceAPI.list({ page: page + 1, page_size: pageSize, search });
      setWorkspaces(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取工作空间列表失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, enqueueSnackbar]);

  useEffect(() => { fetchWorkspaces(); }, [fetchWorkspaces]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await grafanaInstanceAPI.list({ page: 1, page_size: 100 });
        setGrafanaHosts(res.data?.items || []);
      } catch {
        /* grafana hosts optional */
      }
    })();
  }, []);

  const fetchMembers = useCallback(async (workspaceId: string) => {
    setMembersLoading(true);
    try {
      const [membersRes, usersRes] = await Promise.all([
        workspaceAPI.listMembers(workspaceId, { page: 1, page_size: 100 }),
        isAdmin || isWorkspaceAdmin(user, workspaceId) ? userAPI.list({ page: 1, page_size: 200 }) : Promise.resolve(null),
      ]);
      setMembers(membersRes.data.data?.items || []);
      if (usersRes) setAllUsers(usersRes.data.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取成员列表失败'), { variant: 'error' });
    } finally {
      setMembersLoading(false);
    }
  }, [enqueueSnackbar, isAdmin, user]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = { ...form, grafana_instance_id: form.grafana_instance_id || undefined };
      if (editingId) {
        await workspaceAPI.update(editingId, payload);
        enqueueSnackbar('工作空间更新成功', { variant: 'success' });
      } else {
        await workspaceAPI.create(payload as Parameters<typeof workspaceAPI.create>[0]);
        enqueueSnackbar('工作空间创建成功', { variant: 'success' });
      }
      setDialogOpen(false);
      setEditingId(null);
      setForm({ workspace_name: '', template_type: 'shared', grafana_instance_id: '' });
      fetchWorkspaces();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingId ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.tenant) return;
    try {
      await workspaceAPI.delete(deleteDialog.tenant.id);
      enqueueSnackbar('工作空间删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchWorkspaces();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const openEdit = (tenant: Workspace) => {
    setEditingId(tenant.id);
    setForm({ workspace_name: tenant.workspace_name, template_type: tenant.template_type, grafana_instance_id: tenant.grafana_instance_id || '' });
    setDialogOpen(true);
  };

  const openDetail = async (tenant: Workspace, tab: 'overview' | 'members' = 'overview') => {
    setDetail({ open: true, tenant, loading: tab === 'overview', tab });
    if (tab === 'overview') {
      try {
        const { data: res } = await workspaceAPI.metrics(tenant.id);
        setDetail({ open: true, tenant, metrics: res.data, loading: false, tab });
      } catch (err) {
        enqueueSnackbar(extractApiError(err, '获取工作空间指标失败'), { variant: 'warning' });
        setDetail({ open: true, tenant, loading: false, tab });
      }
    }
    if (tab === 'members' || isAdmin) {
      fetchMembers(tenant.id);
    }
  };

  const handleAddMember = async () => {
    if (!detail.tenant || !memberForm.user_id) return;
    setMemberSaving(true);
    try {
      await workspaceAPI.addMember(detail.tenant.id, memberForm);
      enqueueSnackbar('成员已添加', { variant: 'success' });
      setMemberForm({ user_id: '', role: 'member' });
      fetchMembers(detail.tenant.id);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '添加成员失败'), { variant: 'error' });
    } finally {
      setMemberSaving(false);
    }
  };

  // 后端成员路由以 user_id 定位（/members/:userId），不能传 membership 主键。
  const handleUpdateMemberRole = async (member: WorkspaceMember, role: string) => {
    if (!detail.tenant) return;
    try {
      await workspaceAPI.updateMember(detail.tenant.id, member.user_id, { role });
      enqueueSnackbar('成员角色已更新', { variant: 'success' });
      fetchMembers(detail.tenant.id);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新角色失败'), { variant: 'error' });
    }
  };

  const handleRemoveMember = async () => {
    if (!detail.tenant || !removeMemberDialog.member) return;
    try {
      await workspaceAPI.removeMember(detail.tenant.id, removeMemberDialog.member.user_id);
      enqueueSnackbar('成员已移除', { variant: 'success' });
      setRemoveMemberDialog({ open: false });
      fetchMembers(detail.tenant.id);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '移除成员失败'), { variant: 'error' });
    }
  };

  if (loading && workspaces.length === 0) return <LoadingScreen />;

  const existingMemberUserIds = new Set(members.map((m) => m.user_id));
  // 后端成员接口只返回 user_id，这里用用户列表映射出用户名。
  const userNameById = new Map(allUsers.map((u) => [u.id, u.username]));

  return (
    <Box>
      <PageHeader title="工作空间管理" subtitle="管理平台所有工作空间及其资源配置" actionLabel={isAdmin ? "新建工作空间" : undefined} onAction={isAdmin ? () => { setEditingId(null); setForm({ workspace_name: '', template_type: 'shared', grafana_instance_id: '' }); setDialogOpen(true); } : undefined} />

      <Card sx={{ mb: 2 }}>
        <Box sx={{ p: 2, display: 'flex', gap: 2 }}>
          <TextField
            placeholder="搜索工作空间..."
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
                <TableCell>工作空间名称</TableCell>
                <TableCell>VMUser ID</TableCell>
                <TableCell>模板类型</TableCell>
                <TableCell>隔离/命名空间</TableCell>
                <TableCell>Grafana Org</TableCell>
                <TableCell>关联 Grafana</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {workspaces.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9}>
                    <EmptyState title="暂无工作空间" description="点击右上角按钮创建第一个工作空间" />
                  </TableCell>
                </TableRow>
              ) : (
                workspaces.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell sx={{ fontWeight: 500 }}>{t.workspace_name}</TableCell>
                    <TableCell>
                      <Chip label={t.vmuser_id || '-'} size="small" variant="outlined" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }} />
                    </TableCell>
                    <TableCell>
                      <Chip label={templateLabels[t.template_type] || t.template_type} size="small" color="info" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">{isolationLabels[t.isolation_level || 'shared'] || t.isolation_level || '共享'}</Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                        {t.vm_namespace || '-'}
                      </Typography>
                    </TableCell>
                    <TableCell>{t.grafana_org_id || '-'}</TableCell>
                    <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                      {t.grafana_instance_id ? (grafanaHosts.find(h => h.id === t.grafana_instance_id)?.name || t.grafana_instance_id.slice(0, 8)) : '默认'}
                    </TableCell>
                    <TableCell><StatusChip status={t.status} /></TableCell>
                    <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>{new Date(t.created_at).toLocaleDateString()}</TableCell>
                    <TableCell align="right">
                      <Tooltip title="为此工作空间创建监控实例">
                        <IconButton size="small" onClick={() => navigate(`/instances/create?workspace_id=${t.id}`)} aria-label="为此工作空间创建监控实例" color="primary">
                          <AddCircleOutlineIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="成员管理">
                        <IconButton size="small" onClick={() => openDetail(t, 'members')} aria-label="成员管理">
                          <PeopleOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="查看 VM 路由与指标">
                        <IconButton size="small" onClick={() => openDetail(t, 'overview')} aria-label="查看工作空间详情">
                          <VisibilityOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      {isAdmin && (
                        <Tooltip title="编辑">
                          <IconButton size="small" onClick={() => openEdit(t)} aria-label="编辑工作空间">
                            <EditOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                      {isAdmin && (
                        <Tooltip title="删除">
                          <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, tenant: t })} aria-label="删除工作空间">
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
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
        <DialogTitle>{editingId ? '编辑工作空间' : '新建工作空间'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="工作空间名称" value={form.workspace_name} onChange={(e) => setForm({ ...form, workspace_name: e.target.value })} sx={{ mb: 2.5 }} required />
          <FormControl fullWidth size="small">
            <InputLabel>关联 Grafana（可选）</InputLabel>
            <Select value={form.grafana_instance_id} label="关联 Grafana（可选）" onChange={(e) => setForm({ ...form, grafana_instance_id: e.target.value })}>
              <MenuItem value="">使用平台默认</MenuItem>
              {grafanaHosts.map((h) => (
                <MenuItem key={h.id} value={h.id}>
                  {h.name}
                  {h.source === 'platform' ? ' · 平台' : ' · 外部'}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving || !form.workspace_name}>
            {saving ? '保存中...' : editingId ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={detail.open} onClose={() => setDetail({ open: false })} maxWidth="md" fullWidth>
        <DialogTitle>{detail.tenant?.workspace_name}</DialogTitle>
        <DialogContent sx={{ pt: '8px !important' }}>
          <Tabs
            value={detail.tab || 'overview'}
            onChange={(_, tab) => {
              setDetail((prev) => ({ ...prev, tab }));
              if (tab === 'members' && detail.tenant) fetchMembers(detail.tenant.id);
              if (tab === 'overview' && detail.tenant && !detail.metrics) openDetail(detail.tenant, 'overview');
            }}
            sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
          >
            <Tab value="overview" label="概览" />
            <Tab value="members" label="成员" />
          </Tabs>

          {detail.tab === 'members' && detail.tenant && (
            <Box>
              {canManageMembersOf(detail.tenant.id) && (
                <Box sx={{ display: 'flex', gap: 2, mb: 2, flexWrap: 'wrap' }}>
                  <FormControl size="small" sx={{ minWidth: 220, flex: 1 }}>
                    <InputLabel>添加用户</InputLabel>
                    <Select
                      value={memberForm.user_id}
                      label="添加用户"
                      onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })}
                    >
                      {allUsers.filter((u) => !existingMemberUserIds.has(u.id)).map((u) => (
                        <MenuItem key={u.id} value={u.id}>{u.username}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl size="small" sx={{ minWidth: 140 }}>
                    <InputLabel>角色</InputLabel>
                    <Select value={memberForm.role} label="角色" onChange={(e) => setMemberForm({ ...memberForm, role: e.target.value })}>
                      <MenuItem value="admin">管理员</MenuItem>
                      <MenuItem value="member">成员</MenuItem>
                      <MenuItem value="viewer">只读</MenuItem>
                    </Select>
                  </FormControl>
                  <Button variant="contained" onClick={handleAddMember} disabled={memberSaving || !memberForm.user_id}>
                    添加
                  </Button>
                </Box>
              )}
              {membersLoading ? (
                <Typography variant="body2" color="text.secondary">加载中...</Typography>
              ) : members.length === 0 ? (
                <EmptyState title="暂无成员" description="添加用户到此工作空间并分配角色" />
              ) : (
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>用户</TableCell>
                      <TableCell>角色</TableCell>
                      <TableCell>加入时间</TableCell>
                      {canManageMembersOf(detail.tenant.id) && <TableCell align="right">操作</TableCell>}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {members.map((m) => (
                      <TableRow key={m.id}>
                        <TableCell>{userNameById.get(m.user_id) || m.user_id.slice(0, 8)}</TableCell>
                        <TableCell>
                          {canManageMembersOf(detail.tenant?.id) ? (
                            <Select
                              size="small"
                              value={m.role}
                              onChange={(e) => handleUpdateMemberRole(m, e.target.value)}
                              sx={{ minWidth: 120 }}
                            >
                              <MenuItem value="admin">管理员</MenuItem>
                              <MenuItem value="member">成员</MenuItem>
                              <MenuItem value="viewer">只读</MenuItem>
                            </Select>
                          ) : (
                            memberRoleLabels[m.role] || m.role
                          )}
                        </TableCell>
                        <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                          {new Date(m.created_at).toLocaleDateString()}
                        </TableCell>
                        {canManageMembersOf(detail.tenant?.id) && (
                          <TableCell align="right">
                            <IconButton size="small" color="error" onClick={() => setRemoveMemberDialog({ open: true, member: m })}>
                              <DeleteOutlinedIcon fontSize="small" />
                            </IconButton>
                          </TableCell>
                        )}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </Box>
          )}

          {detail.tab !== 'members' && detail.tenant && (
            <Box>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid size={{ xs: 12, md: 4 }}>
                  <Card variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="caption" color="text.secondary">工作空间状态</Typography>
                    <Box sx={{ mt: 1 }}><StatusChip status={detail.tenant.status} /></Box>
                  </Card>
                </Grid>
                <Grid size={{ xs: 12, md: 4 }}>
                  <Card variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="caption" color="text.secondary">Series</Typography>
                    <Typography variant="h6">{detail.metrics?.series_count ?? (detail.loading ? '...' : '-')}</Typography>
                  </Card>
                </Grid>
                <Grid size={{ xs: 12, md: 4 }}>
                  <Card variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="caption" color="text.secondary">Ingest QPS</Typography>
                    <Typography variant="h6">{detail.metrics?.ingest_qps?.toFixed?.(2) ?? (detail.loading ? '...' : '-')}</Typography>
                  </Card>
                </Grid>
              </Grid>
              {detail.metrics?.note && <Alert severity="info" sx={{ mb: 2 }}>{detail.metrics.note}</Alert>}
              <Box sx={{ display: 'grid', gap: 1.5 }}>
                {[
                  ['VMUser', detail.tenant.vmuser_id],
                  ['命名空间', detail.tenant.vm_namespace || '-'],
                  ['Remote Write', detail.tenant.vm_insert_url || detail.tenant.insert_url || '-'],
                  ['Prometheus Query', detail.tenant.vm_select_url || '-'],
                ].map(([label, value]) => (
                  <Box key={label} sx={{ display: 'grid', gridTemplateColumns: '160px 1fr', gap: 1 }}>
                    <Typography variant="body2" color="text.secondary">{label}</Typography>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>{value}</Typography>
                  </Box>
                ))}
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDetail({ open: false })}>关闭</Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除工作空间"
        message={`确定要删除工作空间「${deleteDialog.tenant?.workspace_name}」吗？此操作不可撤销。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />

      <ConfirmDialog
        open={removeMemberDialog.open}
        title="移除成员"
        message={`确定要将该用户从工作空间移除吗？`}
        severity="warning"
        confirmLabel="移除"
        onConfirm={handleRemoveMember}
        onCancel={() => setRemoveMemberDialog({ open: false })}
      />
    </Box>
  );
}
