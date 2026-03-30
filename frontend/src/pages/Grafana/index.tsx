import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Tab,
  Tabs,
  TextField,
  Tooltip,
  Typography,
  Divider,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import PersonAddOutlinedIcon from '@mui/icons-material/PersonAddOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { grafanaAPI } from '../../api/grafana';
import type { GrafanaDatasource, GrafanaOrg, GrafanaOrgUser } from '../../types/api';

export default function GrafanaPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(true);
  const [orgs, setOrgs] = useState<GrafanaOrg[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<GrafanaOrg | null>(null);
  const [tabIndex, setTabIndex] = useState(0);
  const [orgUsers, setOrgUsers] = useState<GrafanaOrgUser[]>([]);
  const [datasources, setDatasources] = useState<GrafanaDatasource[]>([]);

  const [createOrgOpen, setCreateOrgOpen] = useState(false);
  const [orgName, setOrgName] = useState('');
  const [addUserOpen, setAddUserOpen] = useState(false);
  const [userForm, setUserForm] = useState({ loginOrEmail: '', role: 'Viewer' });
  const [addDsOpen, setAddDsOpen] = useState(false);
  const [dsForm, setDsForm] = useState({ name: '', type: 'prometheus', url: '' });
  const [deleteOrgDialog, setDeleteOrgDialog] = useState<{ open: boolean; org?: GrafanaOrg }>({ open: false });
  const [saving, setSaving] = useState(false);

  const fetchOrgs = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await grafanaAPI.listOrgs();
      setOrgs(res.data || []);
      if (res.data?.length > 0 && !selectedOrg) {
        setSelectedOrg(res.data[0]);
      }
    } catch {
      enqueueSnackbar('获取 Grafana 组织列表失败', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar, selectedOrg]);

  useEffect(() => { fetchOrgs(); }, []);

  const fetchOrgDetails = useCallback(async () => {
    if (!selectedOrg) return;
    try {
      const [usersRes, dsRes] = await Promise.allSettled([
        grafanaAPI.listOrgUsers(selectedOrg.id),
        grafanaAPI.listDatasources(selectedOrg.id),
      ]);
      setOrgUsers(usersRes.status === 'fulfilled' ? usersRes.value.data.data || [] : []);
      setDatasources(dsRes.status === 'fulfilled' ? dsRes.value.data.data || [] : []);
    } catch {
      // ignore
    }
  }, [selectedOrg]);

  useEffect(() => { fetchOrgDetails(); }, [fetchOrgDetails]);

  const handleCreateOrg = async () => {
    setSaving(true);
    try {
      await grafanaAPI.createOrg(orgName);
      enqueueSnackbar('组织创建成功', { variant: 'success' });
      setCreateOrgOpen(false);
      setOrgName('');
      fetchOrgs();
    } catch {
      enqueueSnackbar('创建失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteOrg = async () => {
    if (!deleteOrgDialog.org) return;
    try {
      await grafanaAPI.deleteOrg(deleteOrgDialog.org.id);
      enqueueSnackbar('组织删除成功', { variant: 'success' });
      setDeleteOrgDialog({ open: false });
      if (selectedOrg?.id === deleteOrgDialog.org.id) setSelectedOrg(null);
      fetchOrgs();
    } catch {
      enqueueSnackbar('删除失败', { variant: 'error' });
    }
  };

  const handleAddUser = async () => {
    if (!selectedOrg) return;
    setSaving(true);
    try {
      await grafanaAPI.addOrgUser(selectedOrg.id, userForm);
      enqueueSnackbar('用户添加成功', { variant: 'success' });
      setAddUserOpen(false);
      setUserForm({ loginOrEmail: '', role: 'Viewer' });
      fetchOrgDetails();
    } catch {
      enqueueSnackbar('添加失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleRemoveUser = async (userId: number) => {
    if (!selectedOrg) return;
    try {
      await grafanaAPI.removeOrgUser(selectedOrg.id, userId);
      enqueueSnackbar('用户已移除', { variant: 'success' });
      fetchOrgDetails();
    } catch {
      enqueueSnackbar('移除失败', { variant: 'error' });
    }
  };

  const handleAddDatasource = async () => {
    if (!selectedOrg) return;
    setSaving(true);
    try {
      await grafanaAPI.createDatasource(selectedOrg.id, { ...dsForm, access: 'proxy', isDefault: false });
      enqueueSnackbar('数据源创建成功', { variant: 'success' });
      setAddDsOpen(false);
      setDsForm({ name: '', type: 'prometheus', url: '' });
      fetchOrgDetails();
    } catch {
      enqueueSnackbar('创建失败', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteDatasource = async (dsId: number) => {
    if (!selectedOrg) return;
    try {
      await grafanaAPI.deleteDatasource(selectedOrg.id, dsId);
      enqueueSnackbar('数据源已删除', { variant: 'success' });
      fetchOrgDetails();
    } catch {
      enqueueSnackbar('删除失败', { variant: 'error' });
    }
  };

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="Grafana 管理" subtitle="管理 Grafana 组织、用户权限和数据源配置" actionLabel="新建组织" onAction={() => setCreateOrgOpen(true)} />

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
                <Button size="small" variant="outlined" startIcon={<OpenInNewIcon />} onClick={() => window.open(`${import.meta.env.VITE_GRAFANA_URL || '/grafana'}/?orgId=${selectedOrg.id}`, '_blank')}>
                  打开 Grafana
                </Button>
              </Box>
              <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} sx={{ px: 2 }}>
                <Tab label="用户权限" />
                <Tab label="数据源" />
              </Tabs>
              <Divider />

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
                              <IconButton size="small" color="error" onClick={() => handleDeleteDatasource(ds.id)}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
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
          <TextField fullWidth label="用户名或邮箱" value={userForm.loginOrEmail} onChange={(e) => setUserForm({ ...userForm, loginOrEmail: e.target.value })} sx={{ mb: 2 }} required />
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
          <Button variant="contained" onClick={handleAddUser} disabled={saving || !userForm.loginOrEmail}>{saving ? '添加中...' : '添加'}</Button>
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

      <ConfirmDialog
        open={deleteOrgDialog.open}
        title="删除组织"
        message={`确定要删除 Grafana 组织「${deleteOrgDialog.org?.name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDeleteOrg}
        onCancel={() => setDeleteOrgDialog({ open: false })}
      />
    </Box>
  );
}
