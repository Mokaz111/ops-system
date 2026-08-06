import { useCallback, useEffect, useMemo, useState } from 'react';
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
  OutlinedInput,
  Select,
  Stack,
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
  Typography,
  Alert,
  Checkbox,
  ListItemText,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import { useSearchParams } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { alertAPI } from '../../api/alert';
import { workspaceAPI } from '../../api/workspace';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import type { AlertEvent, AlertRule, NotificationChannel, Workspace } from '../../types/api';

const levelMeta: Record<string, { label: string; color: 'error' | 'warning' | 'info' | 'default' }> = {
  critical: { label: '严重', color: 'error' },
  warning: { label: '警告', color: 'warning' },
  info: { label: '信息', color: 'info' },
};

const channelWorkspaceId = (ch: NotificationChannel) => ch.workspace_id || ch.tenant_id || '';

const channelTypeLabels: Record<string, string> = {
  dingtalk: '钉钉',
  email: '邮件',
  slack: 'Slack',
  sms: '短信',
  webhook: 'Webhook',
};

function parseChannelIds(raw: string): string[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(String) : [];
  } catch {
    return raw.split(',').map((s) => s.trim()).filter(Boolean);
  }
}

export default function AlertPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [searchParams] = useSearchParams();
  const instanceId = searchParams.get('instance_id');
  const instanceName = searchParams.get('instance_name');
  const [pageTab, setPageTab] = useState<'rules' | 'channels' | 'events'>('rules');

  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [events, setEvents] = useState<AlertEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [eventPage, setEventPage] = useState(0);
  const [eventPageSize, setEventPageSize] = useState(10);
  const [eventTotal, setEventTotal] = useState(0);
  const [tenantFilter, setWorkspaceFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState('');
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [channelDialogOpen, setChannelDialogOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; type: 'rule' | 'channel'; rule?: AlertRule; channel?: NotificationChannel }>({ open: false, type: 'rule' });
  const [saving, setSaving] = useState(false);
  const [ruleForm, setRuleForm] = useState({
    workspace_id: '',
    rule_name: '',
    rule_type: 'metrics',
    query: '',
    level: 'warning',
    annotations: '',
    enabled: true,
    channel_ids: [] as string[],
  });
  const [channelForm, setChannelForm] = useState({
    workspace_id: '',
    channel_name: '',
    channel_type: 'webhook',
    config: '{}',
    enabled: true,
  });

  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await alertAPI.listRules({
        page: page + 1,
        page_size: pageSize,
        workspace_id: tenantFilter || undefined,
        level: levelFilter || undefined,
      });
      setRules(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取告警规则失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar, levelFilter, page, pageSize, tenantFilter]);

  const fetchChannels = useCallback(async () => {
    try {
      const { data: res } = await alertAPI.listChannels({
        page: 1,
        page_size: 100,
        workspace_id: tenantFilter || undefined,
      });
      setChannels(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取通知渠道失败'), { variant: 'error' });
    }
  }, [enqueueSnackbar, tenantFilter]);

  const fetchEvents = useCallback(async () => {
    try {
      const { data: res } = await alertAPI.listEvents({
        page: eventPage + 1,
        page_size: eventPageSize,
        workspace_id: tenantFilter || undefined,
      });
      setEvents(res.data?.items || []);
      setEventTotal(res.data?.total || 0);
    } catch {
      setEvents([]);
      setEventTotal(0);
    }
  }, [eventPage, eventPageSize, tenantFilter]);

  useEffect(() => {
    if (pageTab === 'rules') fetchRules();
    if (pageTab === 'channels') fetchChannels();
    if (pageTab === 'events') fetchEvents();
  }, [fetchChannels, fetchEvents, fetchRules, pageTab]);

  useEffect(() => {
    fetchChannels();
  }, [fetchChannels]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await workspaceAPI.list({ page: 1, page_size: 200 });
        setWorkspaces(res.data?.items || []);
      } catch {
        /* optional */
      }
    })();
  }, []);

  const tenantName = (id: string) => workspaces.find((t) => t.id === id)?.workspace_name || id.slice(0, 8);
  const channelNameById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const ch of channels) map[ch.id] = ch.channel_name;
    return map;
  }, [channels]);

  const openCreateRule = () => {
    setEditingRule(null);
    setRuleForm({
      workspace_id: tenantFilter || '',
      rule_name: '',
      rule_type: 'metrics',
      query: '',
      level: 'warning',
      annotations: '',
      enabled: true,
      channel_ids: [],
    });
    setRuleDialogOpen(true);
  };

  const openEditRule = (rule: AlertRule) => {
    setEditingRule(rule);
    setRuleForm({
      workspace_id: rule.workspace_id,
      rule_name: rule.rule_name,
      rule_type: rule.rule_type,
      query: rule.query,
      level: rule.level,
      annotations: rule.annotations,
      enabled: rule.enabled,
      channel_ids: parseChannelIds(rule.channels),
    });
    setRuleDialogOpen(true);
  };

  const openCreateChannel = () => {
    setEditingChannel(null);
    setChannelForm({
      workspace_id: tenantFilter || '',
      channel_name: '',
      channel_type: 'webhook',
      config: '{}',
      enabled: true,
    });
    setChannelDialogOpen(true);
  };

  const openEditChannel = (ch: NotificationChannel) => {
    setEditingChannel(ch);
    setChannelForm({
      workspace_id: channelWorkspaceId(ch),
      channel_name: ch.channel_name,
      channel_type: ch.channel_type,
      config: ch.config || '{}',
      enabled: ch.enabled,
    });
    setChannelDialogOpen(true);
  };

  const handleSaveRule = async () => {
    setSaving(true);
    try {
      const payload = {
        workspace_id: ruleForm.workspace_id,
        rule_name: ruleForm.rule_name,
        rule_type: ruleForm.rule_type,
        query: ruleForm.query,
        level: ruleForm.level,
        annotations: ruleForm.annotations,
        enabled: ruleForm.enabled,
        channels: JSON.stringify(ruleForm.channel_ids),
      };
      if (editingRule) {
        await alertAPI.updateRule(editingRule.id, payload);
        enqueueSnackbar('告警规则已更新', { variant: 'success' });
      } else {
        await alertAPI.createRule(payload);
        enqueueSnackbar('VMRule 告警规则已创建', { variant: 'success' });
      }
      setRuleDialogOpen(false);
      fetchRules();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingRule ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleSaveChannel = async () => {
    setSaving(true);
    try {
      if (editingChannel) {
        await alertAPI.updateChannel(editingChannel.id, channelForm);
        enqueueSnackbar('通知渠道已更新', { variant: 'success' });
      } else {
        await alertAPI.createChannel(channelForm);
        enqueueSnackbar('通知渠道已创建', { variant: 'success' });
      }
      setChannelDialogOpen(false);
      fetchChannels();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingChannel ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    try {
      if (deleteDialog.type === 'rule' && deleteDialog.rule) {
        await alertAPI.deleteRule(deleteDialog.rule.id);
        enqueueSnackbar('告警规则已删除', { variant: 'success' });
        fetchRules();
      }
      if (deleteDialog.type === 'channel' && deleteDialog.channel) {
        await alertAPI.deleteChannel(deleteDialog.channel.id);
        enqueueSnackbar('通知渠道已删除', { variant: 'success' });
        fetchChannels();
      }
      setDeleteDialog({ open: false, type: 'rule' });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const handleAckEvent = async (event: AlertEvent) => {
    try {
      await alertAPI.ackEvent(event.id);
      enqueueSnackbar('告警已确认', { variant: 'success' });
      fetchEvents();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '确认失败'), { variant: 'error' });
    }
  };

  const filteredChannelsForRule = channels.filter((ch) => !ruleForm.workspace_id || channelWorkspaceId(ch) === ruleForm.workspace_id);

  if (loading && pageTab === 'rules' && rules.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="告警引擎"
        subtitle="VictoriaMetrics VMRule / vmalert / Alertmanager 告警与通知渠道管理"
        actionLabel={pageTab === 'rules' ? '新建规则' : pageTab === 'channels' && isAdmin ? '新建渠道' : undefined}
        onAction={pageTab === 'rules' ? openCreateRule : pageTab === 'channels' && isAdmin ? openCreateChannel : undefined}
      />

      {instanceId && (
        <Alert severity="success" sx={{ mb: 2 }}>
          当前上下文：实例 {instanceName || instanceId}。可在 PromQL 中加入实例标签来限定告警范围。
        </Alert>
      )}

      <Tabs value={pageTab} onChange={(_, v) => setPageTab(v)} sx={{ mb: 2 }}>
        <Tab value="rules" label="告警规则" />
        <Tab value="channels" label="通知渠道" />
        <Tab value="events" label="告警事件" />
      </Tabs>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ p: 2 }}>
            <Stack direction="row" spacing={1.5} alignItems="center">
              <NotificationsActiveOutlinedIcon color="primary" />
              <Box>
                <Typography variant="caption" color="text.secondary">VMRule 规则数</Typography>
                <Typography variant="h6">{pageTab === 'rules' ? total : '-'}</Typography>
              </Box>
            </Stack>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 8 }}>
          <Card sx={{ p: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel>工作空间</InputLabel>
              <Select value={tenantFilter} label="工作空间" onChange={(e) => { setWorkspaceFilter(e.target.value); setPage(0); setEventPage(0); }}>
                <MenuItem value="">全部工作空间</MenuItem>
                {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
              </Select>
            </FormControl>
            {pageTab === 'rules' && (
              <FormControl size="small" sx={{ minWidth: 160 }}>
                <InputLabel>级别</InputLabel>
                <Select value={levelFilter} label="级别" onChange={(e) => { setLevelFilter(e.target.value); setPage(0); }}>
                  <MenuItem value="">全部级别</MenuItem>
                  <MenuItem value="critical">严重</MenuItem>
                  <MenuItem value="warning">警告</MenuItem>
                  <MenuItem value="info">信息</MenuItem>
                </Select>
              </FormControl>
            )}
          </Card>
        </Grid>
      </Grid>

      {pageTab === 'rules' && (
        <Card>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>规则</TableCell>
                  <TableCell>工作空间</TableCell>
                  <TableCell>PromQL</TableCell>
                  <TableCell>级别</TableCell>
                  <TableCell>通知渠道</TableCell>
                  <TableCell>VMRule</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {rules.length === 0 ? (
                  <TableRow><TableCell colSpan={8}><EmptyState title="暂无告警规则" description="创建第一条 VMRule 告警规则" /></TableCell></TableRow>
                ) : rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell sx={{ fontWeight: 600 }}>{rule.rule_name}</TableCell>
                    <TableCell>{tenantName(rule.workspace_id)}</TableCell>
                    <TableCell>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>{rule.query}</Typography>
                    </TableCell>
                    <TableCell>
                      <Chip size="small" label={levelMeta[rule.level]?.label || rule.level} color={levelMeta[rule.level]?.color || 'default'} />
                    </TableCell>
                    <TableCell>
                      {parseChannelIds(rule.channels).map((id) => (
                        <Chip key={id} size="small" label={channelNameById[id] || id.slice(0, 8)} variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
                      ))}
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                        {rule.vm_namespace || '-'} / {rule.vm_rule_name || '-'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip size="small" label={rule.enabled ? '启用' : '停用'} color={rule.enabled ? 'success' : 'default'} variant="outlined" />
                    </TableCell>
                    <TableCell align="right">
                      {isAdmin && (
                        <Tooltip title="编辑">
                          <IconButton size="small" onClick={() => openEditRule(rule)}>
                            <EditOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                      {isAdmin && (
                        <Tooltip title="删除">
                          <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, type: 'rule', rule })}>
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          {total > 0 && (
            <TablePagination
              component="div"
              count={total}
              page={page}
              rowsPerPage={pageSize}
              onPageChange={(_, next) => setPage(next)}
              onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }}
              rowsPerPageOptions={[10, 20, 50]}
              labelRowsPerPage="每页行数"
            />
          )}
        </Card>
      )}

      {pageTab === 'channels' && (
        <Card>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>名称</TableCell>
                  <TableCell>工作空间</TableCell>
                  <TableCell>类型</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {channels.length === 0 ? (
                  <TableRow><TableCell colSpan={5}><EmptyState title="暂无通知渠道" description="创建通知渠道供告警规则绑定" /></TableCell></TableRow>
                ) : channels.map((ch) => (
                  <TableRow key={ch.id}>
                    <TableCell sx={{ fontWeight: 600 }}>{ch.channel_name}</TableCell>
                    <TableCell>{tenantName(channelWorkspaceId(ch))}</TableCell>
                    <TableCell>{channelTypeLabels[ch.channel_type] || ch.channel_type}</TableCell>
                    <TableCell>
                      <Chip size="small" label={ch.enabled ? '启用' : '停用'} color={ch.enabled ? 'success' : 'default'} variant="outlined" />
                    </TableCell>
                    <TableCell align="right">
                      {isAdmin && (
                        <>
                          <Tooltip title="编辑">
                            <IconButton size="small" onClick={() => openEditChannel(ch)}>
                              <EditOutlinedIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="删除">
                            <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, type: 'channel', channel: ch })}>
                              <DeleteOutlinedIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Card>
      )}

      {pageTab === 'events' && (
        <Card>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>规则</TableCell>
                  <TableCell>级别</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>开始时间</TableCell>
                  <TableCell>确认</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {events.length === 0 ? (
                  <TableRow><TableCell colSpan={6}><EmptyState title="暂无事件" description="vmalert 触发后的事件会在这里展示" /></TableCell></TableRow>
                ) : events.map((event) => (
                  <TableRow key={event.id}>
                    <TableCell sx={{ fontWeight: 500 }}>{event.rule_name}</TableCell>
                    <TableCell>
                      <Chip size="small" label={levelMeta[event.level]?.label || event.level} color={levelMeta[event.level]?.color || 'default'} />
                    </TableCell>
                    <TableCell><Chip size="small" label={event.status} /></TableCell>
                    <TableCell>{new Date(event.start_time).toLocaleString()}</TableCell>
                    <TableCell>
                      {event.acked_at ? (
                        <Typography variant="caption" color="text.secondary">{new Date(event.acked_at).toLocaleString()}</Typography>
                      ) : '-'}
                    </TableCell>
                    <TableCell align="right">
                      {!event.acked_at && event.status === 'firing' && (
                        <Tooltip title="确认告警">
                          <IconButton size="small" color="primary" onClick={() => handleAckEvent(event)}>
                            <CheckCircleOutlineIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          {eventTotal > 0 && (
            <TablePagination
              component="div"
              count={eventTotal}
              page={eventPage}
              rowsPerPage={eventPageSize}
              onPageChange={(_, next) => setEventPage(next)}
              onRowsPerPageChange={(e) => { setEventPageSize(parseInt(e.target.value, 10)); setEventPage(0); }}
              rowsPerPageOptions={[10, 20, 50]}
              labelRowsPerPage="每页行数"
            />
          )}
        </Card>
      )}

      <Dialog open={ruleDialogOpen} onClose={() => setRuleDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingRule ? '编辑告警规则' : '新建 VMRule 告警规则'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>工作空间</InputLabel>
                <Select value={ruleForm.workspace_id} label="工作空间" onChange={(e) => setRuleForm({ ...ruleForm, workspace_id: e.target.value, channel_ids: [] })} disabled={!!editingRule}>
                  {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField fullWidth size="small" label="规则名称" value={ruleForm.rule_name} onChange={(e) => setRuleForm({ ...ruleForm, rule_name: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>级别</InputLabel>
                <Select value={ruleForm.level} label="级别" onChange={(e) => setRuleForm({ ...ruleForm, level: e.target.value })}>
                  <MenuItem value="critical">严重</MenuItem>
                  <MenuItem value="warning">警告</MenuItem>
                  <MenuItem value="info">信息</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>通知渠道</InputLabel>
                <Select
                  multiple
                  value={ruleForm.channel_ids}
                  label="通知渠道"
                  input={<OutlinedInput label="通知渠道" />}
                  renderValue={(selected) => selected.map((id) => channelNameById[id] || id.slice(0, 8)).join(', ')}
                  onChange={(e) => setRuleForm({ ...ruleForm, channel_ids: typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value })}
                >
                  {filteredChannelsForRule.map((ch) => (
                    <MenuItem key={ch.id} value={ch.id}>
                      <Checkbox checked={ruleForm.channel_ids.includes(ch.id)} />
                      <ListItemText primary={ch.channel_name} secondary={channelTypeLabels[ch.channel_type] || ch.channel_type} />
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth multiline minRows={3} label="PromQL" value={ruleForm.query} onChange={(e) => setRuleForm({ ...ruleForm, query: e.target.value })} placeholder='sum(rate(http_requests_total{status=~"5.."}[5m])) > 0' />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth multiline minRows={2} label="注解" value={ruleForm.annotations} onChange={(e) => setRuleForm({ ...ruleForm, annotations: e.target.value })} placeholder="告警说明、处理建议或 Runbook 链接" />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setRuleDialogOpen(false)}>取消</Button>
          <Button startIcon={<AddIcon />} variant="contained" disabled={saving || !ruleForm.workspace_id || !ruleForm.rule_name || !ruleForm.query} onClick={handleSaveRule}>
            {saving ? '保存中...' : editingRule ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={channelDialogOpen} onClose={() => setChannelDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingChannel ? '编辑通知渠道' : '新建通知渠道'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>工作空间</InputLabel>
            <Select value={channelForm.workspace_id} label="工作空间" onChange={(e) => setChannelForm({ ...channelForm, workspace_id: e.target.value })} disabled={!!editingChannel}>
              {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
            </Select>
          </FormControl>
          <TextField fullWidth size="small" label="渠道名称" value={channelForm.channel_name} onChange={(e) => setChannelForm({ ...channelForm, channel_name: e.target.value })} sx={{ mb: 2 }} />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>类型</InputLabel>
            <Select value={channelForm.channel_type} label="类型" onChange={(e) => setChannelForm({ ...channelForm, channel_type: e.target.value })}>
              <MenuItem value="webhook">Webhook</MenuItem>
              <MenuItem value="email">邮件</MenuItem>
              <MenuItem value="dingtalk">钉钉</MenuItem>
              <MenuItem value="slack">Slack</MenuItem>
              <MenuItem value="sms">短信</MenuItem>
            </Select>
          </FormControl>
          <TextField fullWidth multiline minRows={4} size="small" label="配置 (JSON)" value={channelForm.config} onChange={(e) => setChannelForm({ ...channelForm, config: e.target.value })} sx={{ fontFamily: 'monospace' }} />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setChannelDialogOpen(false)}>取消</Button>
          <Button variant="contained" disabled={saving || !channelForm.workspace_id || !channelForm.channel_name} onClick={handleSaveChannel}>
            {saving ? '保存中...' : editingChannel ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title={deleteDialog.type === 'rule' ? '删除告警规则' : '删除通知渠道'}
        message={deleteDialog.type === 'rule'
          ? `确定要删除规则「${deleteDialog.rule?.rule_name}」吗？`
          : `确定要删除渠道「${deleteDialog.channel?.channel_name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false, type: 'rule' })}
      />
    </Box>
  );
}
