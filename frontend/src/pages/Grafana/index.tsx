import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemText,
  MenuItem,
  Select,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import PersonAddOutlinedIcon from '@mui/icons-material/PersonAddOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import ExtensionOutlinedIcon from '@mui/icons-material/ExtensionOutlined';
import UploadOutlinedIcon from '@mui/icons-material/UploadOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { grafanaAPI } from '../../api/grafana';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import type { GrafanaDashboard, GrafanaDatasource, GrafanaOrg, GrafanaOrgUser, GrafanaPlugin } from '../../types/api';

export default function GrafanaPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState('');
  const [orgs, setOrgs] = useState<GrafanaOrg[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<GrafanaOrg | null>(null);
  const [tabIndex, setTabIndex] = useState(0);
  const [orgUsers, setOrgUsers] = useState<GrafanaOrgUser[]>([]);
  const [datasources, setDatasources] = useState<GrafanaDatasource[]>([]);
  const [dashboards, setDashboards] = useState<GrafanaDashboard[]>([]);
  const [plugins, setPlugins] = useState<GrafanaPlugin[]>([]);

  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);
  const [selectedHostId, setSelectedHostId] = useState('');

  const [createOrgOpen, setCreateOrgOpen] = useState(false);
  const [orgName, setOrgName] = useState('');
  const [addUserOpen, setAddUserOpen] = useState(false);
  const [userForm, setUserForm] = useState({ login_or_email: '', role: 'Viewer' });
  const [addDsOpen, setAddDsOpen] = useState(false);
  const [dsForm, setDsForm] = useState({ name: '', type: 'prometheus', url: '' });
  const [editDsDialog, setEditDsDialog] = useState<{ open: boolean; ds?: GrafanaDatasource }>({ open: false });
  const [editDsForm, setEditDsForm] = useState({ name: '', type: '', url: '', access: 'proxy', isDefault: false });
  const [deleteOrgDialog, setDeleteOrgDialog] = useState<{ open: boolean; org?: GrafanaOrg }>({ open: false });
  const [deleteDashboardDialog, setDeleteDashboardDialog] = useState<{ open: boolean; db?: GrafanaDashboard }>({ open: false });
  const [importDashboardOpen, setImportDashboardOpen] = useState(false);
  const [importJson, setImportJson] = useState('');
  const [installPluginOpen, setInstallPluginOpen] = useState(false);
  const [pluginForm, setPluginForm] = useState({ id: '', version: '' });
  const [saving, setSaving] = useState(false);

  const fetchGrafanaHosts = useCallback(async () => {
    try {
      const { data: res } = await grafanaInstanceAPI.list({ page: 1, page_size: 100 });
      setGrafanaHosts(res.data?.items || []);
    } catch {
      // optional
    }
  }, []);

  useEffect(() => { fetchGrafanaHosts(); }, [fetchGrafanaHosts]);

  // Default to platform host when none selected
  const hostId = selectedHostId || grafanaHosts.find((h) => h.scope === 'platform')?.id || '';

  const fetchOrgs = useCallback(async () => {
    if (!hostId) { setLoading(false); return; }
    setLoading(true);
    setFetchError('');
    try {
      const { data: res } = await grafanaAPI.listOrgs(hostId);
      setOrgs(res.data || []);
      if (res.data?.length > 0 && !selectedOrg) {
        setSelectedOrg(res.data[0]);
      }
    } catch (err) {
      const msg = extractApiError(err, '获取 Grafana 组织列表失败');
      setFetchError(msg);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostId]);

  useEffect(() => { fetchOrgs(); }, [fetchOrgs]);

  const fetchOrgDetails = useCallback(async () => {
    if (!selectedOrg || !hostId) return;
    try {
      const [usersRes, dsRes, dashRes] = await Promise.allSettled([
        grafanaAPI.listOrgUsers(hostId, selectedOrg.id),
        grafanaAPI.listDatasources(hostId, selectedOrg.id),
        grafanaAPI.listDashboards(hostId, selectedOrg.id),
      ]);
      setOrgUsers(usersRes.status === 'fulfilled' ? usersRes.value.data.data || [] : []);
      setDatasources(dsRes.status === 'fulfilled' ? dsRes.value.data.data || [] : []);
      setDashboards(dashRes.status === 'fulfilled' ? (dashRes.value.data as any)?.data || [] : []);
    } catch {
      // ignore
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedOrg, hostId]);

  useEffect(() => { fetchOrgDetails(); }, [fetchOrgDetails]);

  const fetchPlugins = useCallback(async () => {
    if (!hostId) return;
    try {
      const { data: res } = await grafanaAPI.listPlugins(hostId);
      setPlugins(res.data || []);
    } catch {
      // plugins optional
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostId]);

  useEffect(() => { fetchPlugins(); }, [fetchPlugins]);

  const handleCreateOrg = async () => {
    setSaving(true);
    try {
      await grafanaAPI.createOrg(hostId, orgName);
      enqueueSnackbar('组织创建成功', { variant: 'success' });
      setCreateOrgOpen(false);
      setOrgName('');
      fetchOrgs();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteOrg = async () => {
    if (!deleteOrgDialog.org) return;
    try {
      await grafanaAPI.deleteOrg(hostId, deleteOrgDialog.org.id);
      enqueueSnackbar('组织删除成功', { variant: 'success' });
      setDeleteOrgDialog({ open: false });
      if (selectedOrg?.id === deleteOrgDialog.org.id) setSelectedOrg(null);
      fetchOrgs();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const handleAddUser = async () => {
    if (!selectedOrg) return;
    setSaving(true);
    try {
      await grafanaAPI.addOrgUser(hostId, selectedOrg.id, userForm);
      enqueueSnackbar('用户添加成功', { variant: 'success' });
      setAddUserOpen(false);
      setUserForm({ login_or_email: '', role: 'Viewer' });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '添加失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleRemoveUser = async (userId: number) => {
    if (!selectedOrg) return;
    try {
      await grafanaAPI.removeOrgUser(hostId, selectedOrg.id, userId);
      enqueueSnackbar('用户已移除', { variant: 'success' });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '移除失败'), { variant: 'error' });
    }
  };

  const handleAddDatasource = async () => {
    if (!selectedOrg) return;
    setSaving(true);
    try {
      await grafanaAPI.createDatasource(hostId, selectedOrg.id, { ...dsForm, access: 'proxy', isDefault: false });
      enqueueSnackbar('数据源创建成功', { variant: 'success' });
      setAddDsOpen(false);
      setDsForm({ name: '', type: 'prometheus', url: '' });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleEditDatasource = async () => {
    if (!selectedOrg || !editDsDialog.ds) return;
    setSaving(true);
    try {
      await grafanaAPI.updateDatasource(hostId, selectedOrg.id, editDsDialog.ds.id, editDsForm);
      enqueueSnackbar('数据源已更新', { variant: 'success' });
      setEditDsDialog({ open: false });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteDatasource = async (dsId: number) => {
    if (!selectedOrg) return;
    try {
      await grafanaAPI.deleteDatasource(hostId, selectedOrg.id, dsId);
      enqueueSnackbar('数据源已删除', { variant: 'success' });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const handleImportDashboard = async () => {
    if (!selectedOrg || !importJson.trim()) return;
    setSaving(true);
    try {
      const dashboard = JSON.parse(importJson);
      await grafanaAPI.importDashboard(hostId, selectedOrg.id, dashboard);
      enqueueSnackbar('Dashboard 导入成功', { variant: 'success' });
      setImportDashboardOpen(false);
      setImportJson('');
      fetchOrgDetails();
    } catch (err) {
      if (err instanceof SyntaxError) {
        enqueueSnackbar('JSON 格式无效', { variant: 'error' });
      } else {
        enqueueSnackbar(extractApiError(err, '导入失败'), { variant: 'error' });
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteDashboard = async () => {
    if (!selectedOrg || !deleteDashboardDialog.db) return;
    try {
      await grafanaAPI.deleteDashboard(hostId, selectedOrg.id, deleteDashboardDialog.db.uid);
      enqueueSnackbar('Dashboard 已删除', { variant: 'success' });
      setDeleteDashboardDialog({ open: false });
      fetchOrgDetails();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const handleInstallPlugin = async () => {
    setSaving(true);
    try {
      await grafanaAPI.installPlugin(hostId, pluginForm.id, pluginForm.version || undefined);
      enqueueSnackbar('插件安装请求已提交', { variant: 'success' });
      setInstallPluginOpen(false);
      setPluginForm({ id: '', version: '' });
      fetchPlugins();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '安装失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleUninstallPlugin = async (pluginId: string) => {
    try {
      await grafanaAPI.uninstallPlugin(hostId, pluginId);
      enqueueSnackbar('插件已卸载', { variant: 'success' });
      fetchPlugins();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '卸载失败'), { variant: 'error' });
    }
  };

  const selectedHost = grafanaHosts.find((h) => h.id === selectedHostId);
  const grafanaUrl = selectedHost?.url || import.meta.env.VITE_GRAFANA_URL || '';

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="Grafana 管理" subtitle="管理 Grafana 组织、用户权限、数据源、Dashboard 和插件" actionLabel="新建组织" onAction={() => setCreateOrgOpen(true)} />

      <Box sx={{ mb: 2.5, display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
        <FormControl size="small" sx={{ minWidth: 280 }}>
          <InputLabel>选择 Grafana 实例</InputLabel>
          <Select
            value={selectedHostId}
            label="选择 Grafana 实例"
            onChange={(e) => {
              setSelectedHostId(e.target.value);
              setSelectedOrg(null);
              setTabIndex(0);
              setFetchError('');
            }}
          >
            <MenuItem value="">平台默认 Grafana</MenuItem>
            {grafanaHosts.map((h) => (
              <MenuItem key={h.id} value={h.id}>
                {h.name}
                <Chip size="small" label={h.scope === 'platform' ? '平台' : '租户'} variant="outlined" sx={{ ml: 1, height: 18, fontSize: '0.65rem' }} />
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        {selectedHost && (
          <Typography variant="body2" color="text.secondary" sx={{ wordBreak: 'break-all' }}>
            {selectedHost.url}
          </Typography>
        )}
        {grafanaHosts.length === 0 && !loading && (
          <Typography variant="caption" color="text.disabled">
            暂无已登记的 Grafana 实例，请先在「Grafana 实例」页面登记外部 Grafana。
          </Typography>
        )}
      </Box>

      {fetchError && (
        <Alert severity="error" sx={{ mb: 2.5 }} onClose={() => setFetchError('')}>
          {fetchError}
        </Alert>
      )}

      <Grid container spacing={2.5}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent sx={{ p: 0 }}>
              <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="subtitle1">组织列表</Typography>
                <Chip label={orgs.length} size="small" color="primary" variant="outlined" />
              </Box>
              <Divider />
              {orgs.length === 0 ? (
                <EmptyState title="暂无组织" />
              ) : (
                <List disablePadding>
                  {orgs.map((org) => (
                    <ListItem
                      key={org.id}
                      component="div"
                      onClick={() => { setSelectedOrg(org); setTabIndex(0); }}
                      sx={{
                        cursor: 'pointer',
                        backgroundColor: selectedOrg?.id === org.id ? 'action.selected' : 'transparent',
                        '&:hover': { backgroundColor: 'action.hover' },
                        borderLeft: selectedOrg?.id === org.id ? '3px solid' : '3px solid transparent',
                        borderColor: selectedOrg?.id === org.id ? 'primary.main' : 'transparent',
                      }}
                      secondaryAction={
                        <Tooltip title="删除组织">
                          <IconButton edge="end" size="small" color="error" onClick={(e) => { e.stopPropagation(); setDeleteOrgDialog({ open: true, org }); }}>
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      }
                    >
                      <ListItemText
                        primary={org.name}
                        secondary={`ID: ${org.id}`}
                        primaryTypographyProps={{ fontWeight: selectedOrg?.id === org.id ? 600 : 400, fontSize: '0.875rem' }}
                      />
                    </ListItem>
                  ))}
                </List>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 8 }}>
          {selectedOrg ? (
            <Card>
              <Box sx={{ px: 2, pt: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="subtitle1">{selectedOrg.name}</Typography>
                {grafanaUrl ? (
                  <Button size="small" variant="outlined" startIcon={<OpenInNewIcon />} onClick={() => window.open(`${grafanaUrl}/?orgId=${selectedOrg.id}`, '_blank')}>
                    打开 Grafana
                  </Button>
                ) : (
                  <Typography variant="caption" color="text.disabled">未配置 Grafana 地址</Typography>
                )}
              </Box>
              <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} sx={{ px: 2 }}>
                <Tab label="组织用户" />
                <Tab label="数据源" />
                <Tab label="Dashboard" />
                <Tab label="插件" />
              </Tabs>
              <Divider />

              {/* Tab 0: Users */}
              {tabIndex === 0 && (
                <Box sx={{ p: 2 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
                    <Button size="small" startIcon={<PersonAddOutlinedIcon />} onClick={() => setAddUserOpen(true)}>添加用户</Button>
                  </Box>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>用户</TableCell>
                          <TableCell>邮箱</TableCell>
                          <TableCell>角色</TableCell>
                          <TableCell align="right">操作</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {orgUsers.length === 0 ? (
                          <TableRow><TableCell colSpan={4}><EmptyState title="暂无用户" /></TableCell></TableRow>
                        ) : orgUsers.map((u) => (
                          <TableRow key={u.userId}>
                            <TableCell>{u.login}</TableCell>
                            <TableCell sx={{ color: 'text.secondary' }}>{u.email}</TableCell>
                            <TableCell><Chip label={u.role} size="small" variant="outlined" /></TableCell>
                            <TableCell align="right">
                              <IconButton size="small" color="error" onClick={() => handleRemoveUser(u.userId)}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Box>
              )}

              {/* Tab 1: Datasources */}
              {tabIndex === 1 && (
                <Box sx={{ p: 2 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
                    <Button size="small" startIcon={<StorageOutlinedIcon />} onClick={() => setAddDsOpen(true)}>添加数据源</Button>
                  </Box>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>名称</TableCell>
                          <TableCell>类型</TableCell>
                          <TableCell>URL</TableCell>
                          <TableCell align="right">操作</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {datasources.length === 0 ? (
                          <TableRow><TableCell colSpan={4}><EmptyState title="暂无数据源" /></TableCell></TableRow>
                        ) : datasources.map((ds) => (
                          <TableRow key={ds.id}>
                            <TableCell sx={{ fontWeight: 500 }}>{ds.name}</TableCell>
                            <TableCell><Chip label={ds.type} size="small" variant="outlined" /></TableCell>
                            <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem', color: 'text.secondary' }}>{ds.url}</TableCell>
                            <TableCell align="right">
                              <Tooltip title="编辑">
                                <IconButton size="small" onClick={() => {
                                  setEditDsForm({ name: ds.name, type: ds.type, url: ds.url, access: ds.access, isDefault: ds.isDefault });
                                  setEditDsDialog({ open: true, ds });
                                }}>
                                  <EditOutlinedIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                              <IconButton size="small" color="error" onClick={() => handleDeleteDatasource(ds.id)}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Box>
              )}

              {/* Tab 2: Dashboards */}
              {tabIndex === 2 && (
                <Box sx={{ p: 2 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
                    <Button size="small" startIcon={<UploadOutlinedIcon />} onClick={() => setImportDashboardOpen(true)}>导入 Dashboard</Button>
                  </Box>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>标题</TableCell>
                          <TableCell>UID</TableCell>
                          <TableCell>标签</TableCell>
                          <TableCell align="right">操作</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {dashboards.length === 0 ? (
                          <TableRow><TableCell colSpan={4}><EmptyState title="暂无 Dashboard" description="点击右上角按钮导入 Dashboard JSON" /></TableCell></TableRow>
                        ) : dashboards.map((db) => (
                          <TableRow key={db.uid}>
                            <TableCell sx={{ fontWeight: 500 }}>{db.title}</TableCell>
                            <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}>{db.uid}</TableCell>
                            <TableCell>
                              {db.tags?.map((tag) => <Chip key={tag} label={tag} size="small" variant="outlined" sx={{ mr: 0.5 }} />)}
                            </TableCell>
                            <TableCell align="right">
                              <Tooltip title="打开">
                                <IconButton size="small" onClick={() => window.open(`${grafanaUrl}${db.url}`, '_blank')}>
                                  <OpenInNewIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                              <Tooltip title="删除">
                                <IconButton size="small" color="error" onClick={() => setDeleteDashboardDialog({ open: true, db })}>
                                  <DeleteOutlinedIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Box>
              )}

              {/* Tab 3: Plugins */}
              {tabIndex === 3 && (
                <Box sx={{ p: 2 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
                    <Button size="small" startIcon={<ExtensionOutlinedIcon />} onClick={() => setInstallPluginOpen(true)}>安装插件</Button>
                  </Box>
                  {plugins.length === 0 ? (
                    <EmptyState title="暂无已安装插件" />
                  ) : (
                    <Grid container spacing={1.5}>
                      {plugins.map((p) => (
                        <Grid size={{ xs: 12, sm: 6 }} key={p.id}>
                          <Card variant="outlined">
                            <Box sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                              <Box>
                                <Typography variant="body2" sx={{ fontWeight: 500 }}>{p.name}</Typography>
                                <Typography variant="caption" color="text.secondary">{p.id}@{p.version}</Typography>
                                <Box sx={{ mt: 0.5 }}>
                                  <Chip label={p.type} size="small" sx={{ mr: 0.5, height: 18, fontSize: '0.65rem' }} />
                                  <Chip label={p.enabled ? '已启用' : '已禁用'} size="small" color={p.enabled ? 'success' : 'default'} variant="outlined" sx={{ height: 18, fontSize: '0.65rem' }} />
                                </Box>
                              </Box>
                              <Tooltip title="卸载插件">
                                <IconButton size="small" color="error" onClick={() => handleUninstallPlugin(p.id)}>
                                  <DeleteOutlinedIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                            </Box>
                          </Card>
                        </Grid>
                      ))}
                    </Grid>
                  )}
                </Box>
              )}
            </Card>
          ) : (
            <Card><CardContent><EmptyState title="请选择组织" description="从左侧列表选择一个 Grafana 组织查看详情" /></CardContent></Card>
          )}
        </Grid>
      </Grid>

      {/* Create Org */}
      <Dialog open={createOrgOpen} onClose={() => setCreateOrgOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>新建 Grafana 组织</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="组织名称" value={orgName} onChange={(e) => setOrgName(e.target.value)} required />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCreateOrgOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleCreateOrg} disabled={saving || !orgName}>{saving ? '创建中...' : '创建'}</Button>
        </DialogActions>
      </Dialog>

      {/* Add User */}
      <Dialog open={addUserOpen} onClose={() => setAddUserOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>添加用户到 {selectedOrg?.name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="用户名或邮箱" value={userForm.login_or_email} onChange={(e) => setUserForm({ ...userForm, login_or_email: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small">
            <InputLabel>角色</InputLabel>
            <Select value={userForm.role} label="角色" onChange={(e) => setUserForm({ ...userForm, role: e.target.value })}>
              <MenuItem value="Viewer">Viewer</MenuItem>
              <MenuItem value="Editor">Editor</MenuItem>
              <MenuItem value="Admin">Admin</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setAddUserOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleAddUser} disabled={saving || !userForm.login_or_email}>{saving ? '添加中...' : '添加'}</Button>
        </DialogActions>
      </Dialog>

      {/* Add Datasource */}
      <Dialog open={addDsOpen} onClose={() => setAddDsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>添加数据源到 {selectedOrg?.name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="数据源名称" value={dsForm.name} onChange={(e) => setDsForm({ ...dsForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>类型</InputLabel>
            <Select value={dsForm.type} label="类型" onChange={(e) => setDsForm({ ...dsForm, type: e.target.value })}>
              <MenuItem value="prometheus">Prometheus</MenuItem>
              <MenuItem value="loki">Loki</MenuItem>
              <MenuItem value="elasticsearch">Elasticsearch</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth label="URL" value={dsForm.url} onChange={(e) => setDsForm({ ...dsForm, url: e.target.value })} placeholder="http://vm-select:8481" required />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setAddDsOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleAddDatasource} disabled={saving || !dsForm.name || !dsForm.url}>{saving ? '创建中...' : '创建'}</Button>
        </DialogActions>
      </Dialog>

      {/* Edit Datasource */}
      <Dialog open={editDsDialog.open} onClose={() => setEditDsDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>编辑数据源</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="数据源名称" value={editDsForm.name} onChange={(e) => setEditDsForm({ ...editDsForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>类型</InputLabel>
            <Select value={editDsForm.type} label="类型" onChange={(e) => setEditDsForm({ ...editDsForm, type: e.target.value })}>
              <MenuItem value="prometheus">Prometheus</MenuItem>
              <MenuItem value="loki">Loki</MenuItem>
              <MenuItem value="elasticsearch">Elasticsearch</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth label="URL" value={editDsForm.url} onChange={(e) => setEditDsForm({ ...editDsForm, url: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>访问方式</InputLabel>
            <Select value={editDsForm.access} label="访问方式" onChange={(e) => setEditDsForm({ ...editDsForm, access: e.target.value })}>
              <MenuItem value="proxy">Proxy</MenuItem>
              <MenuItem value="direct">Direct</MenuItem>
            </Select>
          </FormControl>
          <FormControl fullWidth size="small">
            <InputLabel>是否默认</InputLabel>
            <Select value={editDsForm.isDefault ? 'true' : 'false'} label="是否默认" onChange={(e) => setEditDsForm({ ...editDsForm, isDefault: e.target.value === 'true' })}>
              <MenuItem value="false">否</MenuItem>
              <MenuItem value="true">是</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setEditDsDialog({ open: false })}>取消</Button>
          <Button variant="contained" onClick={handleEditDatasource} disabled={saving || !editDsForm.name || !editDsForm.url}>{saving ? '保存中...' : '保存'}</Button>
        </DialogActions>
      </Dialog>

      {/* Import Dashboard */}
      <Dialog open={importDashboardOpen} onClose={() => setImportDashboardOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>导入 Dashboard 到 {selectedOrg?.name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>粘贴 Grafana Dashboard JSON（可从 Grafana 导出或使用预置模板）</Typography>
          <TextField
            fullWidth
            multiline
            minRows={10}
            maxRows={20}
            label="Dashboard JSON"
            value={importJson}
            onChange={(e) => setImportJson(e.target.value)}
            placeholder='{"title": "My Dashboard", "panels": [...], ...}'
            sx={{ '& .MuiInputBase-root': { fontFamily: 'monospace', fontSize: '0.8125rem' } }}
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setImportDashboardOpen(false)}>取消</Button>
          <Button variant="contained" startIcon={<UploadOutlinedIcon />} onClick={handleImportDashboard} disabled={saving || !importJson.trim()}>{saving ? '导入中...' : '导入'}</Button>
        </DialogActions>
      </Dialog>

      {/* Install Plugin */}
      <Dialog open={installPluginOpen} onClose={() => setInstallPluginOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>安装 Grafana 插件</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="插件 ID" value={pluginForm.id} onChange={(e) => setPluginForm({ ...pluginForm, id: e.target.value })} sx={{ mb: 2 }} required helperText="例如: grafana-piechart-panel" />
          <TextField fullWidth label="版本（可选）" value={pluginForm.version} onChange={(e) => setPluginForm({ ...pluginForm, version: e.target.value })} helperText="留空安装最新版本" />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setInstallPluginOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleInstallPlugin} disabled={saving || !pluginForm.id}>{saving ? '安装中...' : '安装'}</Button>
        </DialogActions>
      </Dialog>

      {/* Delete Org */}
      <ConfirmDialog
        open={deleteOrgDialog.open}
        title="删除组织"
        message={`确定要删除 Grafana 组织「${deleteOrgDialog.org?.name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDeleteOrg}
        onCancel={() => setDeleteOrgDialog({ open: false })}
      />

      {/* Delete Dashboard */}
      <ConfirmDialog
        open={deleteDashboardDialog.open}
        title="删除 Dashboard"
        message={`确定要删除 Dashboard「${deleteDashboardDialog.db?.title}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDeleteDashboard}
        onCancel={() => setDeleteDashboardDialog({ open: false })}
      />
    </Box>
  );
}
