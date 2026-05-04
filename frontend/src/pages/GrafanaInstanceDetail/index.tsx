import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
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
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import ReplayIcon from '@mui/icons-material/Replay';
import SystemUpdateAltIcon from '@mui/icons-material/SystemUpdateAlt';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import PersonAddOutlinedIcon from '@mui/icons-material/PersonAddOutlined';
import UploadOutlinedIcon from '@mui/icons-material/UploadOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import HealthAndSafetyOutlinedIcon from '@mui/icons-material/HealthAndSafetyOutlined';
import CheckCircleOutlineOutlinedIcon from '@mui/icons-material/CheckCircleOutlineOutlined';
import ErrorOutlineOutlinedIcon from '@mui/icons-material/ErrorOutlineOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import StatusChip from '../../components/common/StatusChip';
import LoadingScreen from '../../components/common/LoadingScreen';
import EmptyState from '../../components/common/EmptyState';
import DetailTabs from '../../components/common/DetailTabs';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { instanceAPI } from '../../api/instance';
import { grafanaAPI } from '../../api/grafana';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { ssoLoginToGrafana } from '../../api/grafanaSso';
import { extractApiError } from '../../api';
import type { Instance, GrafanaDashboard, GrafanaDatasource, GrafanaOrg, GrafanaOrgUser } from '../../types/api';
import { parseSpec } from '../../utils/instance';

export default function GrafanaInstanceDetailPage() {
  const navigate = useNavigate();
  const { instanceId = '' } = useParams();
  const { enqueueSnackbar } = useSnackbar();
  const [activeTab, setActiveTab] = useState('base');

  // ---- Instance ----
  const [loading, setLoading] = useState(true);
  const [instance, setInstance] = useState<Instance | null>(null);

  // ---- Grafana Instance ----
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);
  const hostId = useMemo(() => {
    if (instance?.grafana_instance_id) return instance.grafana_instance_id;
    const platform = grafanaHosts.find((h) => h.scope === 'platform');
    return platform?.id || '';
  }, [instance, grafanaHosts]);

  // ---- Orgs ----
  const [orgs, setOrgs] = useState<GrafanaOrg[]>([]);
  const [selectedOrgId, setSelectedOrgId] = useState<number | null>(null);

  // ---- Datasources ----
  const [datasources, setDatasources] = useState<GrafanaDatasource[]>([]);
  const [dsOpen, setDsOpen] = useState(false);
  const [dsForm, setDsForm] = useState({ name: '', type: 'prometheus', url: '' });
  const [editDsDialog, setEditDsDialog] = useState<{ open: boolean; ds?: GrafanaDatasource }>({ open: false });
  const [editDsForm, setEditDsForm] = useState({ name: '', type: '', url: '', access: 'proxy', isDefault: false });

  // ---- Org Users ----
  const [orgUsers, setOrgUsers] = useState<GrafanaOrgUser[]>([]);
  const [addUserOpen, setAddUserOpen] = useState(false);
  const [userForm, setUserForm] = useState({ login_or_email: '', role: 'Viewer' });

  // ---- Health ----
  const [health, setHealth] = useState<{ status: string; message?: string } | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);

  // ---- Dashboards ----
  const [dashboards, setDashboards] = useState<GrafanaDashboard[]>([]);
  const [importDashboardOpen, setImportDashboardOpen] = useState(false);
  const [importJson, setImportJson] = useState('');
  const [deleteDbDialog, setDeleteDbDialog] = useState<{ open: boolean; db?: GrafanaDashboard }>({ open: false });

  // ---- Edit / Rebuild / Upgrade ----
  const [editOpen, setEditOpen] = useState(false);
  const [editForm, setEditForm] = useState({ instance_name: '', grafana_instance_id: '', cpu: '', memory: '', storage: '', retention: '' });
  const [rebuildDialog, setRebuildDialog] = useState(false);
  const [saving, setSaving] = useState(false);

  // ---- Fetch instance ----
  useEffect(() => {
    if (!instanceId) return;
    setLoading(true);
    (async () => {
      try {
        const { data: res } = await instanceAPI.get(instanceId);
        setInstance(res.data || null);
      } catch (err) {
        enqueueSnackbar(extractApiError(err, '获取实例详情失败'), { variant: 'error' });
        navigate('/grafana-instances');
      } finally {
        setLoading(false);
      }
    })();
  }, [instanceId, enqueueSnackbar, navigate]);

  // ---- Fetch Grafana hosts ----
  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await grafanaInstanceAPI.list({ page: 1, page_size: 100 });
        setGrafanaHosts(res.data?.items || []);
      } catch { /* optional */ }
    })();
  }, []);

  // ---- Fetch orgs when hostId is known ----
  const fetchOrgs = useCallback(async () => {
    if (!hostId) return;
    try {
      const { data: res } = await grafanaAPI.listOrgs(hostId);
      setOrgs(res.data || []);
    } catch { /* optional */ }
  }, [hostId]);

  useEffect(() => { fetchOrgs(); }, [fetchOrgs]);

  // ---- Fetch datasources when org selected ----
  useEffect(() => {
    if (!hostId || selectedOrgId == null) return;
    (async () => {
      try {
        const { data: res } = await grafanaAPI.listDatasources(hostId, selectedOrgId);
        setDatasources(res.data || []);
      } catch { /* optional */ }
    })();
  }, [hostId, selectedOrgId]);

  // ---- Fetch org users when org selected ----
  useEffect(() => {
    if (!hostId || selectedOrgId == null) return;
    (async () => {
      try {
        const { data: res } = await grafanaAPI.listOrgUsers(hostId, selectedOrgId);
        setOrgUsers(res.data || []);
      } catch { /* optional */ }
    })();
  }, [hostId, selectedOrgId]);

  // ---- Fetch dashboards when org selected ----
  useEffect(() => {
    if (!hostId || selectedOrgId == null) return;
    (async () => {
      try {
        const { data: res } = await grafanaAPI.listDashboards(hostId, selectedOrgId);
        setDashboards(res.data || []);
      } catch { /* optional */ }
    })();
  }, [hostId, selectedOrgId]);

  const fetchDashboards = useCallback(async () => {
    if (!hostId || selectedOrgId == null) return;
    try {
      const { data: res } = await grafanaAPI.listDashboards(hostId, selectedOrgId);
      setDashboards(res.data || []);
    } catch { /* optional */ }
  }, [hostId, selectedOrgId]);

  const spec = useMemo(() => parseSpec(instance?.spec || '{}'), [instance?.spec]);

  if (loading) return <LoadingScreen />;
  if (!instance) {
    return (
      <Box>
        <Typography color="error">实例不存在或已删除</Typography>
        <Button variant="outlined" sx={{ mt: 2 }} onClick={() => navigate('/grafana-instances')}>返回列表</Button>
      </Box>
    );
  }

  // ---- Handlers ----
  const handleEdit = async () => {
    setSaving(true);
    try {
      const newSpec = JSON.stringify({
        cpu: parseInt(editForm.cpu, 10),
        memory: parseInt(editForm.memory, 10),
        storage: parseInt(editForm.storage, 10),
        retention: parseInt(editForm.retention, 10),
      });
      await instanceAPI.update(instance.id, {
        instance_name: editForm.instance_name || undefined,
        grafana_instance_id: editForm.grafana_instance_id || undefined,
        spec: newSpec,
      });
      enqueueSnackbar('实例更新成功', { variant: 'success' });
      setEditOpen(false);
      window.location.reload();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleRebuild = async () => {
    try {
      await instanceAPI.rebuild(instance.id);
      enqueueSnackbar('重建请求已提交', { variant: 'success' });
      setRebuildDialog(false);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '重建失败'), { variant: 'error' });
    }
  };

  const handleUpgrade = async () => {
    try {
      await instanceAPI.upgrade(instance.id);
      enqueueSnackbar('升级请求已提交', { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '升级失败'), { variant: 'error' });
    }
  };

  // ---- Datasource handlers ----
  const handleCreateDs = async () => {
    if (!selectedOrgId) return;
    try {
      await grafanaAPI.createDatasource(hostId, selectedOrgId, dsForm);
      enqueueSnackbar('数据源创建成功', { variant: 'success' });
      setDsOpen(false);
      setDsForm({ name: '', type: 'prometheus', url: '' });
      const { data: res } = await grafanaAPI.listDatasources(hostId, selectedOrgId);
      setDatasources(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建数据源失败'), { variant: 'error' });
    }
  };

  const handleUpdateDs = async () => {
    if (!selectedOrgId || !editDsDialog.ds) return;
    try {
      await grafanaAPI.updateDatasource(hostId, selectedOrgId, editDsDialog.ds.id, {
        name: editDsForm.name,
        type: editDsForm.type,
        url: editDsForm.url,
        access: editDsForm.access,
        isDefault: editDsForm.isDefault,
      });
      enqueueSnackbar('数据源更新成功', { variant: 'success' });
      setEditDsDialog({ open: false });
      const { data: res } = await grafanaAPI.listDatasources(hostId, selectedOrgId);
      setDatasources(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新数据源失败'), { variant: 'error' });
    }
  };

  const handleDeleteDs = async (ds: GrafanaDatasource) => {
    if (!selectedOrgId) return;
    try {
      await grafanaAPI.deleteDatasource(hostId, selectedOrgId, ds.id);
      enqueueSnackbar('数据源已删除', { variant: 'success' });
      const { data: res } = await grafanaAPI.listDatasources(hostId, selectedOrgId);
      setDatasources(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除数据源失败'), { variant: 'error' });
    }
  };

  // ---- Org user handlers ----
  const handleAddUser = async () => {
    if (!selectedOrgId) return;
    try {
      await grafanaAPI.addOrgUser(hostId, selectedOrgId, userForm);
      enqueueSnackbar('用户添加成功', { variant: 'success' });
      setAddUserOpen(false);
      setUserForm({ login_or_email: '', role: 'Viewer' });
      const { data: res } = await grafanaAPI.listOrgUsers(hostId, selectedOrgId);
      setOrgUsers(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '添加用户失败'), { variant: 'error' });
    }
  };

  const handleRemoveUser = async (userId: number) => {
    if (!selectedOrgId) return;
    try {
      await grafanaAPI.removeOrgUser(hostId, selectedOrgId, userId);
      enqueueSnackbar('用户已移除', { variant: 'success' });
      const { data: res } = await grafanaAPI.listOrgUsers(hostId, selectedOrgId);
      setOrgUsers(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '移除用户失败'), { variant: 'error' });
    }
  };

  // ---- Dashboard handlers ----
  const handleImportDashboard = async () => {
    if (!selectedOrgId) return;
    let jsonData: object;
    try { jsonData = JSON.parse(importJson); } catch {
      enqueueSnackbar('JSON 格式无效', { variant: 'error' });
      return;
    }
    try {
      await grafanaAPI.importDashboard(hostId, selectedOrgId, jsonData);
      enqueueSnackbar('Dashboard 导入成功', { variant: 'success' });
      setImportDashboardOpen(false);
      setImportJson('');
      await fetchDashboards();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '导入失败'), { variant: 'error' });
    }
  };

  const handleDeleteDashboard = async () => {
    if (!selectedOrgId || !deleteDbDialog.db) return;
    try {
      await grafanaAPI.deleteDashboard(hostId, selectedOrgId, deleteDbDialog.db.uid);
      enqueueSnackbar('Dashboard 已删除', { variant: 'success' });
      setDeleteDbDialog({ open: false });
      await fetchDashboards();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  // ---- Health check ----
  const handleHealthCheck = async () => {
    if (!hostId) return;
    setHealthLoading(true);
    try {
      const { data: res } = await grafanaAPI.healthCheck(hostId);
      setHealth(res.data || null);
    } catch (err) {
      setHealth({ status: 'error', message: extractApiError(err, '健康检查失败') });
    } finally {
      setHealthLoading(false);
    }
  };

  const hostName = instance.grafana_instance_id
    ? (grafanaHosts.find((h) => h.id === instance.grafana_instance_id)?.name || instance.grafana_instance_id.slice(0, 8))
    : '平台默认';

  return (
    <Box>
      <PageHeader
        title={instance.instance_name}
        subtitle="Grafana 实例详情"
        extra={
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button startIcon={<ArrowBackIcon />} variant="outlined" onClick={() => navigate('/grafana-instances')}>
              返回列表
            </Button>
            <Button
              variant="outlined"
              startIcon={<LoginOutlinedIcon />}
              onClick={() => {
                ssoLoginToGrafana(instanceAPI.login(instance.id)).catch((err) =>
                  enqueueSnackbar(extractApiError(err, '获取登录信息失败'), { variant: 'error' }),
                );
              }}
            >
              登录 Grafana
            </Button>
          </Box>
        }
      />

      <DetailTabs
        value={activeTab}
        onChange={setActiveTab}
        items={[
          // ===== 基本信息 =====
          {
            key: 'base',
            label: '基本信息',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="subtitle2">实例配置</Typography>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button
                      size="small"
                      startIcon={<EditOutlinedIcon />}
                      onClick={() => {
                        setEditForm({
                          instance_name: instance.instance_name,
                          grafana_instance_id: instance.grafana_instance_id || '',
                          cpu: String(spec.cpu),
                          memory: String(spec.memory),
                          storage: String(spec.storage),
                          retention: String(spec.retention),
                        });
                        setEditOpen(true);
                      }}
                    >
                      编辑
                    </Button>
                    <Button
                      size="small"
                      color="warning"
                      startIcon={<ReplayIcon />}
                      onClick={() => setRebuildDialog(true)}
                      disabled={instance.status !== 'running' && instance.status !== 'failed'}
                    >
                      重建
                    </Button>
                    <Button
                      size="small"
                      color="primary"
                      startIcon={<SystemUpdateAltIcon />}
                      onClick={handleUpgrade}
                      disabled={instance.status !== 'running'}
                    >
                      升级
                    </Button>
                  </Box>
                </Box>
                <Grid container spacing={2}>
                  <Grid size={{ xs: 12, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">实例名称</Typography>
                    <Typography variant="body1" sx={{ fontWeight: 500 }}>{instance.instance_name}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">类型</Typography>
                    <Chip label="Grafana" size="small" color="success" variant="outlined" />
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">状态</Typography>
                    <StatusChip status={instance.status} />
                  </Grid>
                  <Grid size={{ xs: 6, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">模板</Typography>
                    <Typography variant="body2">{instance.template_type}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">命名空间</Typography>
                    <Typography variant="body2">{instance.namespace || '-'}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">关联 Grafana 主机</Typography>
                    <Typography variant="body2">{hostName}</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">CPU</Typography>
                    <Typography variant="body2">{spec.cpu} Core</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">内存</Typography>
                    <Typography variant="body2">{spec.memory} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">存储</Typography>
                    <Typography variant="body2">{spec.storage} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 6, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">创建时间</Typography>
                    <Typography variant="body2">{new Date(instance.created_at).toLocaleString()}</Typography>
                  </Grid>
                </Grid>
              </Card>
            ),
          },
          // ===== 数据源 =====
          {
            key: 'datasources',
            label: '数据源',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    <Typography variant="subtitle2">数据源管理</Typography>
                    {orgs.length > 0 && (
                      <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>选择组织</InputLabel>
                        <Select
                          value={selectedOrgId ?? ''}
                          label="选择组织"
                          onChange={(e) => setSelectedOrgId(e.target.value ? Number(e.target.value) : null)}
                        >
                          <MenuItem value="">请选择</MenuItem>
                          {orgs.map((org) => (
                            <MenuItem key={org.id} value={org.id}>{org.name}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    )}
                  </Box>
                  {selectedOrgId && (
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<StorageOutlinedIcon />}
                      onClick={() => setDsOpen(true)}
                    >
                      创建数据源
                    </Button>
                  )}
                </Box>
                {!hostId ? (
                  <Typography variant="body2" color="text.secondary">未关联 Grafana 主机，无法管理数据源。</Typography>
                ) : !selectedOrgId ? (
                  <Typography variant="body2" color="text.secondary">请先选择组织。</Typography>
                ) : datasources.length === 0 ? (
                  <EmptyState title="暂无数据源" description="点击右上角创建数据源" />
                ) : (
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>名称</TableCell>
                          <TableCell>类型</TableCell>
                          <TableCell>URL</TableCell>
                          <TableCell>默认</TableCell>
                          <TableCell align="right">操作</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {datasources.map((ds) => (
                          <TableRow key={ds.id}>
                            <TableCell sx={{ fontWeight: 500 }}>{ds.name}</TableCell>
                            <TableCell><Chip size="small" label={ds.type} variant="outlined" /></TableCell>
                            <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>{ds.url}</TableCell>
                            <TableCell>{ds.isDefault ? <Chip size="small" label="默认" color="primary" /> : '-'}</TableCell>
                            <TableCell align="right">
                              <IconButton size="small" onClick={() => {
                                setEditDsForm({ name: ds.name, type: ds.type, url: ds.url, access: ds.access, isDefault: ds.isDefault });
                                setEditDsDialog({ open: true, ds });
                              }}><EditOutlinedIcon fontSize="small" /></IconButton>
                              <IconButton size="small" color="error" onClick={() => handleDeleteDs(ds)}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Card>
            ),
          },
          // ===== 组织 =====
          {
            key: 'orgs',
            label: '组织',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    <Typography variant="subtitle2">组织管理</Typography>
                    {orgs.length > 0 && (
                      <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>选择组织</InputLabel>
                        <Select
                          value={selectedOrgId ?? ''}
                          label="选择组织"
                          onChange={(e) => setSelectedOrgId(e.target.value ? Number(e.target.value) : null)}
                        >
                          <MenuItem value="">请选择</MenuItem>
                          {orgs.map((org) => (
                            <MenuItem key={org.id} value={org.id}>{org.name}</MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    )}
                  </Box>
                  {selectedOrgId && (
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<PersonAddOutlinedIcon />}
                      onClick={() => setAddUserOpen(true)}
                    >
                      添加用户
                    </Button>
                  )}
                </Box>
                {!hostId ? (
                  <Typography variant="body2" color="text.secondary">未关联 Grafana 主机，无法管理组织。</Typography>
                ) : !selectedOrgId ? (
                  <Typography variant="body2" color="text.secondary">请先选择组织，查看其成员。</Typography>
                ) : orgUsers.length === 0 ? (
                  <EmptyState title="暂无成员" description="点击右上角添加用户到该组织" />
                ) : (
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>用户名</TableCell>
                          <TableCell>邮箱</TableCell>
                          <TableCell>角色</TableCell>
                          <TableCell align="right">操作</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {orgUsers.map((u) => (
                          <TableRow key={u.userId}>
                            <TableCell sx={{ fontWeight: 500 }}>{u.login}</TableCell>
                            <TableCell>{u.email || '-'}</TableCell>
                            <TableCell><Chip size="small" label={u.role} variant="outlined" /></TableCell>
                            <TableCell align="right">
                              <IconButton size="small" color="error" onClick={() => handleRemoveUser(u.userId)}><DeleteOutlinedIcon fontSize="small" /></IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Card>
            ),
          },
          // ===== Dashboard =====
          {
            key: 'dashboards',
            label: 'Dashboard',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    <Typography variant="subtitle2">Dashboard 管理</Typography>
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                      <InputLabel>选择组织</InputLabel>
                      <Select
                        value={selectedOrgId ?? ''}
                        label="选择组织"
                        onChange={(e) => setSelectedOrgId(e.target.value ? Number(e.target.value) : null)}
                      >
                        <MenuItem value="">请选择</MenuItem>
                        {orgs.map((org) => (
                          <MenuItem key={org.id} value={org.id}>{org.name}</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Box>
                  {selectedOrgId && (
                    <Button
                      size="small"
                      variant="outlined"
                      startIcon={<UploadOutlinedIcon />}
                      onClick={() => setImportDashboardOpen(true)}
                    >
                      导入 Dashboard
                    </Button>
                  )}
                </Box>
                {!hostId ? (
                  <Typography variant="body2" color="text.secondary">未关联 Grafana 主机，无法管理 Dashboard。</Typography>
                ) : !selectedOrgId ? (
                  <Typography variant="body2" color="text.secondary">请先选择组织。</Typography>
                ) : dashboards.length === 0 ? (
                  <EmptyState title="暂无 Dashboard" description="点击右上角导入 Dashboard JSON" />
                ) : (
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
                        {dashboards.map((db) => (
                          <TableRow key={db.uid}>
                            <TableCell sx={{ fontWeight: 500 }}>{db.title}</TableCell>
                            <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8125rem' }}>{db.uid}</TableCell>
                            <TableCell>
                              {db.tags?.length ? db.tags.map((t) => <Chip key={t} size="small" label={t} variant="outlined" sx={{ mr: 0.5 }} />) : '-'}
                            </TableCell>
                            <TableCell align="right">
                              {db.url && (
                                <IconButton size="small" component="a" href={db.url} target="_blank" rel="noopener noreferrer">
                                  <OpenInNewIcon fontSize="small" />
                                </IconButton>
                              )}
                              <IconButton size="small" color="error" onClick={() => setDeleteDbDialog({ open: true, db })}>
                                <DeleteOutlinedIcon fontSize="small" />
                              </IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Card>
            ),
          },
          // ===== 配置 =====
          {
            key: 'config',
            label: '配置',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>实例配置</Typography>
                <Grid container spacing={2}>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>CPU</Typography>
                    <Typography variant="body1">{spec.cpu} Core</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>内存</Typography>
                    <Typography variant="body1">{spec.memory} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>存储</Typography>
                    <Typography variant="body1">{spec.storage} Gi</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>数据保留</Typography>
                    <Typography variant="body1">{spec.retention} 天</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>Grafana 主机</Typography>
                    <Typography variant="body1">{hostName}</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>命名空间</Typography>
                    <Typography variant="body1">{instance.namespace || '-'}</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>Release 名称</Typography>
                    <Typography variant="body1">{instance.release_name || '-'}</Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>访问地址</Typography>
                    <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{instance.url || '-'}</Typography>
                  </Grid>
                </Grid>
              </Card>
            ),
          },
          // ===== 健康检查 =====
          {
            key: 'health',
            label: '健康检查',
            content: (
              <Card sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="subtitle2">Grafana 健康状态</Typography>
                  <Button
                    size="small"
                    variant="outlined"
                    startIcon={<HealthAndSafetyOutlinedIcon />}
                    onClick={handleHealthCheck}
                    disabled={!hostId || healthLoading}
                  >
                    {healthLoading ? '检查中...' : '执行健康检查'}
                  </Button>
                </Box>
                {!hostId ? (
                  <Typography variant="body2" color="text.secondary">未关联 Grafana 主机，无法进行健康检查。</Typography>
                ) : !health ? (
                  <Typography variant="body2" color="text.secondary">点击"执行健康检查"检测 Grafana 实例状态。</Typography>
                ) : (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, p: 3, bgcolor: health.status === 'ok' ? 'success.main' : 'error.main', borderRadius: 2, color: '#fff' }}>
                    {health.status === 'ok' ? (
                      <CheckCircleOutlineOutlinedIcon sx={{ fontSize: 40 }} />
                    ) : (
                      <ErrorOutlineOutlinedIcon sx={{ fontSize: 40 }} />
                    )}
                    <Box>
                      <Typography variant="h6">{health.status === 'ok' ? 'Grafana 运行正常' : 'Grafana 异常'}</Typography>
                      {health.message && <Typography variant="body2" sx={{ opacity: 0.9 }}>{health.message}</Typography>}
                    </Box>
                  </Box>
                )}
              </Card>
            ),
          },
        ]}
      />

      {/* ===== Edit instance dialog ===== */}
      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>编辑 Grafana 实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="实例名称" value={editForm.instance_name} onChange={(e) => setEditForm({ ...editForm, instance_name: e.target.value })} sx={{ mb: 2.5 }} />
          <FormControl fullWidth size="small" sx={{ mb: 2.5 }}>
            <InputLabel>关联 Grafana 主机</InputLabel>
            <Select value={editForm.grafana_instance_id} label="关联 Grafana 主机" onChange={(e) => setEditForm({ ...editForm, grafana_instance_id: e.target.value })}>
              <MenuItem value="">平台默认</MenuItem>
              {grafanaHosts.map((h) => <MenuItem key={h.id} value={h.id}>{h.name}</MenuItem>)}
            </Select>
          </FormControl>
          <Typography variant="subtitle2" sx={{ mb: 1.5, color: 'text.secondary' }}>资源配置</Typography>
          <Grid container spacing={2}>
            <Grid size={{ xs: 6 }}>
              <TextField fullWidth size="small" label="CPU (核)" type="number" value={editForm.cpu} onChange={(e) => setEditForm({ ...editForm, cpu: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 6 }}>
              <TextField fullWidth size="small" label="内存 (GB)" type="number" value={editForm.memory} onChange={(e) => setEditForm({ ...editForm, memory: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 6 }}>
              <TextField fullWidth size="small" label="存储 (GB)" type="number" value={editForm.storage} onChange={(e) => setEditForm({ ...editForm, storage: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 6 }}>
              <TextField fullWidth size="small" label="保留 (天)" type="number" value={editForm.retention} onChange={(e) => setEditForm({ ...editForm, retention: e.target.value })} />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setEditOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleEdit} disabled={saving}>{saving ? '保存中...' : '保存'}</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Rebuild confirm ===== */}
      <ConfirmDialog
        open={rebuildDialog}
        title="重建 Grafana 实例"
        message={`确定要重建 Grafana 实例「${instance.instance_name}」吗？将重新部署 Helm Release。`}
        severity="warning"
        confirmLabel="重建"
        onConfirm={handleRebuild}
        onCancel={() => setRebuildDialog(false)}
      />

      {/* ===== Create Datasource ===== */}
      <Dialog open={dsOpen} onClose={() => setDsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>创建数据源</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="名称" value={dsForm.name} onChange={(e) => setDsForm({ ...dsForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>类型</InputLabel>
            <Select value={dsForm.type} label="类型" onChange={(e) => setDsForm({ ...dsForm, type: e.target.value })}>
              <MenuItem value="prometheus">Prometheus</MenuItem>
              <MenuItem value="loki">Loki</MenuItem>
              <MenuItem value="elasticsearch">Elasticsearch</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth size="small" label="URL" value={dsForm.url} onChange={(e) => setDsForm({ ...dsForm, url: e.target.value })} sx={{ mb: 1 }} required placeholder="http://victoria-metrics:8428" />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDsOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleCreateDs} disabled={!dsForm.name || !dsForm.url}>创建</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Edit Datasource ===== */}
      <Dialog open={editDsDialog.open} onClose={() => setEditDsDialog({ open: false })} maxWidth="sm" fullWidth>
        <DialogTitle>编辑数据源</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="名称" value={editDsForm.name} onChange={(e) => setEditDsForm({ ...editDsForm, name: e.target.value })} sx={{ mb: 2 }} required />
          <TextField fullWidth size="small" label="类型" value={editDsForm.type} onChange={(e) => setEditDsForm({ ...editDsForm, type: e.target.value })} sx={{ mb: 2 }} required />
          <TextField fullWidth size="small" label="URL" value={editDsForm.url} onChange={(e) => setEditDsForm({ ...editDsForm, url: e.target.value })} sx={{ mb: 2 }} required />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>访问模式</InputLabel>
            <Select value={editDsForm.access} label="访问模式" onChange={(e) => setEditDsForm({ ...editDsForm, access: e.target.value })}>
              <MenuItem value="proxy">Proxy</MenuItem>
              <MenuItem value="direct">Direct</MenuItem>
            </Select>
          </FormControl>
          <FormControl fullWidth size="small">
            <InputLabel>默认</InputLabel>
            <Select value={editDsForm.isDefault ? 'true' : 'false'} label="默认" onChange={(e) => setEditDsForm({ ...editDsForm, isDefault: e.target.value === 'true' })}>
              <MenuItem value="false">否</MenuItem>
              <MenuItem value="true">是</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setEditDsDialog({ open: false })}>取消</Button>
          <Button variant="contained" onClick={handleUpdateDs} disabled={!editDsForm.name || !editDsForm.url}>保存</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Add User ===== */}
      <Dialog open={addUserOpen} onClose={() => setAddUserOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>添加用户</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth size="small" label="登录名或邮箱" value={userForm.login_or_email} onChange={(e) => setUserForm({ ...userForm, login_or_email: e.target.value })} sx={{ mb: 2 }} required />
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
          <Button variant="contained" onClick={handleAddUser} disabled={!userForm.login_or_email}>添加</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Import Dashboard ===== */}
      <Dialog open={importDashboardOpen} onClose={() => setImportDashboardOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>导入 Dashboard</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField
            fullWidth
            multiline
            minRows={8}
            maxRows={20}
            label="Dashboard JSON"
            value={importJson}
            onChange={(e) => setImportJson(e.target.value)}
            placeholder='粘贴 Grafana Dashboard JSON 模型...'
            sx={{ mb: 2 }}
          />
          <Typography variant="caption" color="text.secondary">
            在 Grafana 中导出 Dashboard JSON，粘贴到此处后点击导入。支持覆盖同 UID 的已有 Dashboard。
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setImportDashboardOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleImportDashboard} disabled={!importJson.trim()}>导入</Button>
        </DialogActions>
      </Dialog>

      {/* ===== Delete Dashboard ===== */}
      <ConfirmDialog
        open={deleteDbDialog.open}
        title="删除 Dashboard"
        message={`确定要删除 Dashboard「${deleteDbDialog.db?.title}」吗？该操作不可回退。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDeleteDashboard}
        onCancel={() => setDeleteDbDialog({ open: false })}
      />
    </Box>
  );
}
