import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  ListItemText,
  MenuItem,
  OutlinedInput,
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
import AddIcon from '@mui/icons-material/Add';
import UploadFileOutlinedIcon from '@mui/icons-material/UploadFileOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import SendOutlinedIcon from '@mui/icons-material/SendOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import FilterToolbar from '../../components/common/FilterToolbar';
import { alertAPI } from '../../api/alert';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import type { AlertRule, ImportRulesResult, NotificationChannel } from '../../types/api';
import { levelMeta, parseChannelIds, useWorkspaceOptions, WorkspaceFilterSelect } from './shared';

export default function AlertRulesPage() {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [searchParams] = useSearchParams();
  const instanceId = searchParams.get('instance_id');
  const instanceName = searchParams.get('instance_name');

  const { workspaces, tenantName } = useWorkspaceOptions();
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  // 默认继承 TopBar 的全局工作空间上下文，页内可临时覆盖。
  const globalWorkspaceId = useWorkspaceStore((s) => s.currentId);
  const [tenantFilter, setTenantFilter] = useState(globalWorkspaceId);
  useEffect(() => { setTenantFilter(globalWorkspaceId); setPage(0); }, [globalWorkspaceId]);
  const [levelFilter, setLevelFilter] = useState('');

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; rule?: AlertRule }>({ open: false });
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    workspace_id: '',
    rule_name: '',
    rule_type: 'metrics',
    query: '',
    level: 'warning',
    annotations: '',
    enabled: true,
    channel_ids: [] as string[],
  });

  // 批量导入 Prometheus 风格规则
  const [importOpen, setImportOpen] = useState(false);
  const [importTenantId, setImportTenantId] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<ImportRulesResult | null>(null);

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
      const { data: res } = await alertAPI.listChannels({ page: 1, page_size: 100, workspace_id: tenantFilter || undefined });
      setChannels(res.data?.items || []);
    } catch {
      /* 渠道列表用于名称映射，失败不阻塞 */
    }
  }, [tenantFilter]);

  useEffect(() => { fetchRules(); }, [fetchRules]);
  useEffect(() => { fetchChannels(); }, [fetchChannels]);

  const channelNameById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const ch of channels) map[ch.id] = ch.channel_name;
    return map;
  }, [channels]);

  const filteredChannelsForForm = channels.filter((ch) => !form.workspace_id || ch.tenant_id === form.workspace_id);

  const openCreate = () => {
    setEditingRule(null);
    setForm({
      workspace_id: tenantFilter || '',
      rule_name: '',
      rule_type: 'metrics',
      query: '',
      level: 'warning',
      annotations: '',
      enabled: true,
      channel_ids: [],
    });
    setDialogOpen(true);
  };

  const openEdit = (rule: AlertRule) => {
    setEditingRule(rule);
    setForm({
      workspace_id: rule.tenant_id,
      rule_name: rule.rule_name,
      rule_type: rule.rule_type,
      query: rule.query,
      level: rule.level,
      annotations: rule.annotations,
      enabled: rule.enabled,
      channel_ids: parseChannelIds(rule.channels),
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      // 后端 createAlertRuleBody 使用 tenant_id（必填）。
      const payload = {
        tenant_id: form.workspace_id,
        rule_name: form.rule_name,
        rule_type: form.rule_type,
        query: form.query,
        level: form.level,
        annotations: form.annotations,
        enabled: form.enabled,
        channels: JSON.stringify(form.channel_ids),
      };
      if (editingRule) {
        await alertAPI.updateRule(editingRule.id, payload);
        enqueueSnackbar('告警规则已更新', { variant: 'success' });
      } else {
        await alertAPI.createRule(payload);
        enqueueSnackbar('VMRule 告警规则已创建', { variant: 'success' });
      }
      setDialogOpen(false);
      fetchRules();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, editingRule ? '更新失败' : '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteDialog.rule) return;
    try {
      await alertAPI.deleteRule(deleteDialog.rule.id);
      enqueueSnackbar('告警规则已删除', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchRules();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const openImport = () => {
    setImportTenantId(tenantFilter || '');
    setImportFile(null);
    setImportResult(null);
    setImportOpen(true);
  };

  const handleImport = async () => {
    if (!importTenantId || !importFile) return;
    setImporting(true);
    try {
      const { data: res } = await alertAPI.importRules(importTenantId, importFile);
      const result = res.data;
      setImportResult(result);
      if (result.created > 0) {
        enqueueSnackbar(`成功导入 ${result.created} 条规则`, { variant: 'success' });
        fetchRules();
      } else {
        enqueueSnackbar('没有导入任何规则，请查看导入结果详情', { variant: 'warning' });
      }
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '导入失败'), { variant: 'error' });
    } finally {
      setImporting(false);
    }
  };

  if (loading && rules.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="告警规则"
        subtitle="VMRule / vmalert 规则定义，触发后经 Alertmanager 分发"
        actionLabel={isAdmin ? '新建规则' : undefined}
        onAction={isAdmin ? openCreate : undefined}
      />

      {instanceId && (
        <Alert severity="success" sx={{ mb: 2 }}>
          当前上下文：实例 {instanceName || instanceId}。可在 PromQL 中加入实例标签来限定告警范围。
        </Alert>
      )}

      <FilterToolbar>
        <WorkspaceFilterSelect
          value={tenantFilter}
          onChange={(v) => { setTenantFilter(v); setPage(0); }}
          workspaces={workspaces}
        />
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel>级别</InputLabel>
          <Select value={levelFilter} label="级别" onChange={(e) => { setLevelFilter(e.target.value); setPage(0); }}>
            <MenuItem value="">全部级别</MenuItem>
            <MenuItem value="critical">严重</MenuItem>
            <MenuItem value="warning">警告</MenuItem>
            <MenuItem value="info">信息</MenuItem>
          </Select>
        </FormControl>
        <Box sx={{ flex: 1 }} />
        <Button startIcon={<SendOutlinedIcon />} onClick={() => navigate('/alerts/channels')}>
          通知渠道
        </Button>
        {isAdmin && (
          <Button variant="outlined" startIcon={<UploadFileOutlinedIcon />} onClick={openImport}>
            导入 Prometheus 规则
          </Button>
        )}
      </FilterToolbar>

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
                <TableRow><TableCell colSpan={8}><EmptyState title="暂无告警规则" description="创建第一条 VMRule 告警规则，或从 Prometheus YAML 批量导入" /></TableCell></TableRow>
              ) : rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell sx={{ fontWeight: 600 }}>{rule.rule_name}</TableCell>
                  <TableCell>{tenantName(rule.tenant_id)}</TableCell>
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
                      <>
                        <Tooltip title="编辑">
                          <IconButton size="small" onClick={() => openEdit(rule)}>
                            <EditOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="删除">
                          <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, rule })}>
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

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingRule ? '编辑告警规则' : '新建告警规则'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>工作空间</InputLabel>
                <Select
                  value={form.workspace_id}
                  label="工作空间"
                  onChange={(e) => setForm({ ...form, workspace_id: e.target.value, channel_ids: [] })}
                  disabled={!!editingRule}
                >
                  {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField fullWidth size="small" label="规则名称" value={form.rule_name} onChange={(e) => setForm({ ...form, rule_name: e.target.value })} />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>级别</InputLabel>
                <Select value={form.level} label="级别" onChange={(e) => setForm({ ...form, level: e.target.value })}>
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
                  value={form.channel_ids}
                  label="通知渠道"
                  input={<OutlinedInput label="通知渠道" />}
                  renderValue={(selected) => selected.map((id) => channelNameById[id] || id.slice(0, 8)).join(', ')}
                  onChange={(e) => setForm({ ...form, channel_ids: typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value })}
                >
                  {filteredChannelsForForm.map((ch) => (
                    <MenuItem key={ch.id} value={ch.id}>
                      <Checkbox checked={form.channel_ids.includes(ch.id)} />
                      <ListItemText primary={ch.channel_name} />
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth multiline minRows={3} label="PromQL" value={form.query} onChange={(e) => setForm({ ...form, query: e.target.value })} placeholder='sum(rate(http_requests_total{status=~"5.."}[5m])) > 0' />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField fullWidth multiline minRows={2} label="注解" value={form.annotations} onChange={(e) => setForm({ ...form, annotations: e.target.value })} placeholder="告警说明、处理建议或 Runbook 链接" />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button startIcon={<AddIcon />} variant="contained" disabled={saving || !form.workspace_id || !form.rule_name || !form.query} onClick={handleSave}>
            {saving ? '保存中...' : editingRule ? '更新' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={importOpen} onClose={() => !importing && setImportOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>导入 Prometheus 告警规则</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            支持标准 Prometheus rule file（含 groups）格式的 .yaml/.yml 文件，或打包多个规则文件的 .zip/.tar.gz 压缩包。
            recording rules 与已存在的同名规则会自动跳过；severity 标签将映射为平台告警级别。
          </Alert>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>目标工作空间</InputLabel>
            <Select value={importTenantId} label="目标工作空间" onChange={(e) => setImportTenantId(e.target.value)}>
              {workspaces.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.workspace_name}</MenuItem>)}
            </Select>
          </FormControl>
          <Button component="label" variant="outlined" startIcon={<UploadFileOutlinedIcon />} sx={{ mb: 1 }}>
            {importFile ? importFile.name : '选择文件（.yaml / .yml / .zip / .tar.gz）'}
            <input
              hidden
              type="file"
              accept=".yaml,.yml,.zip,.tar.gz,.tgz,application/x-yaml,application/zip,application/gzip"
              onChange={(e) => {
                setImportFile(e.target.files?.[0] || null);
                setImportResult(null);
                e.target.value = '';
              }}
            />
          </Button>
          {importFile && (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              {(importFile.size / 1024).toFixed(1)} KB（上限 20MB）
            </Typography>
          )}

          {importResult && (
            <Box sx={{ mt: 2 }}>
              <Alert severity={importResult.errors.length > 0 ? 'warning' : 'success'} sx={{ mb: 1 }}>
                共解析 {importResult.total} 条告警规则：成功导入 {importResult.created} 条，
                跳过重名 {importResult.skipped_duplicate} 条，跳过 recording rules {importResult.skipped_recording} 条，
                失败 {importResult.errors.length} 条。
              </Alert>
              {importResult.errors.length > 0 && (
                <TableContainer sx={{ maxHeight: 220, border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                  <Table size="small" stickyHeader>
                    <TableHead>
                      <TableRow>
                        <TableCell>文件</TableCell>
                        <TableCell>规则</TableCell>
                        <TableCell>原因</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {importResult.errors.map((e, idx) => (
                        <TableRow key={idx}>
                          <TableCell sx={{ fontSize: 12 }}>{e.file}{e.group ? ` / ${e.group}` : ''}</TableCell>
                          <TableCell sx={{ fontSize: 12 }}>{e.alert || '-'}</TableCell>
                          <TableCell sx={{ fontSize: 12, color: 'error.main' }}>{e.reason}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setImportOpen(false)} disabled={importing}>
            {importResult ? '关闭' : '取消'}
          </Button>
          <Button
            variant="contained"
            startIcon={<UploadFileOutlinedIcon />}
            disabled={importing || !importTenantId || !importFile}
            onClick={handleImport}
          >
            {importing ? '导入中...' : '开始导入'}
          </Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除告警规则"
        message={`确定要删除规则「${deleteDialog.rule?.rule_name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
