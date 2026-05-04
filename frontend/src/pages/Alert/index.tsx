import { useCallback, useEffect, useState } from 'react';
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
  Stack,
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
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import { useSearchParams } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { alertAPI } from '../../api/alert';
import { tenantAPI } from '../../api/tenant';
import { extractApiError } from '../../api';
import type { AlertEvent, AlertRule, Tenant } from '../../types/api';

const levelMeta: Record<string, { label: string; color: 'error' | 'warning' | 'info' | 'default' }> = {
  critical: { label: '严重', color: 'error' },
  warning: { label: '警告', color: 'warning' },
  info: { label: '信息', color: 'info' },
};

export default function AlertPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [searchParams] = useSearchParams();
  const instanceId = searchParams.get('instance_id');
  const instanceName = searchParams.get('instance_name');
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [events, setEvents] = useState<AlertEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [tenantFilter, setTenantFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; rule?: AlertRule }>({ open: false });
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    tenant_id: '',
    rule_name: '',
    rule_type: 'metrics',
    query: '',
    level: 'warning',
    annotations: '',
    enabled: true,
  });

  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await alertAPI.listRules({
        page: page + 1,
        page_size: pageSize,
        tenant_id: tenantFilter || undefined,
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

  const fetchEvents = useCallback(async () => {
    try {
      const { data: res } = await alertAPI.listEvents({
        page: 1,
        page_size: 5,
        tenant_id: tenantFilter || undefined,
      });
      setEvents(res.data?.items || []);
    } catch {
      /* event stream may be empty or unavailable */
    }
  }, [tenantFilter]);

  useEffect(() => {
    fetchRules();
    fetchEvents();
  }, [fetchEvents, fetchRules]);

  useEffect(() => {
    (async () => {
      try {
        const { data: res } = await tenantAPI.list({ page: 1, page_size: 200 });
        setTenants(res.data?.items || []);
      } catch {
        /* tenant filter optional */
      }
    })();
  }, []);

  const tenantName = (id: string) => tenants.find((t) => t.id === id)?.tenant_name || id.slice(0, 8);

  const handleCreate = async () => {
    setSaving(true);
    try {
      await alertAPI.createRule({
        tenant_id: form.tenant_id,
        rule_name: form.rule_name,
        rule_type: form.rule_type,
        query: form.query,
        level: form.level,
        annotations: form.annotations,
        enabled: form.enabled,
      });
      enqueueSnackbar('VMRule 告警规则已创建', { variant: 'success' });
      setDialogOpen(false);
      setForm({ tenant_id: '', rule_name: '', rule_type: 'metrics', query: '', level: 'warning', annotations: '', enabled: true });
      fetchRules();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建告警规则失败'), { variant: 'error' });
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
      enqueueSnackbar(extractApiError(err, '删除告警规则失败'), { variant: 'error' });
    }
  };

  if (loading && rules.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="告警引擎"
        subtitle="以 VictoriaMetrics VMRule / vmalert / Alertmanager 为主链路管理租户告警"
        actionLabel="新建 VMRule"
        onAction={() => setDialogOpen(true)}
      />

      {instanceId && (
        <Alert severity="success" sx={{ mb: 2 }}>
          当前上下文：实例 {instanceName || instanceId}。可在 PromQL 中加入实例标签来限定告警范围。
        </Alert>
      )}

      <Alert severity="info" sx={{ mb: 3 }}>
        后端会把指标类规则落库后渲染为租户命名空间下的 VMRule，由 vmalert 计算并发送到 Alertmanager；N9E 仅作为兼容同步路径。
      </Alert>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ p: 2 }}>
            <Stack direction="row" spacing={1.5} alignItems="center">
              <NotificationsActiveOutlinedIcon color="primary" />
              <Box>
                <Typography variant="caption" color="text.secondary">VMRule 规则数</Typography>
                <Typography variant="h6">{total}</Typography>
              </Box>
            </Stack>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 8 }}>
          <Card sx={{ p: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel>租户</InputLabel>
              <Select value={tenantFilter} label="租户" onChange={(e) => { setTenantFilter(e.target.value); setPage(0); }}>
                <MenuItem value="">全部租户</MenuItem>
                {tenants.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.tenant_name}</MenuItem>)}
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ minWidth: 160 }}>
              <InputLabel>级别</InputLabel>
              <Select value={levelFilter} label="级别" onChange={(e) => { setLevelFilter(e.target.value); setPage(0); }}>
                <MenuItem value="">全部级别</MenuItem>
                <MenuItem value="critical">严重</MenuItem>
                <MenuItem value="warning">警告</MenuItem>
                <MenuItem value="info">信息</MenuItem>
              </Select>
            </FormControl>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>规则</TableCell>
                <TableCell>租户</TableCell>
                <TableCell>PromQL</TableCell>
                <TableCell>级别</TableCell>
                <TableCell>VMRule</TableCell>
                <TableCell>状态</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.length === 0 ? (
                <TableRow><TableCell colSpan={7}><EmptyState title="暂无告警规则" description="创建第一条 VMRule 告警规则" /></TableCell></TableRow>
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
                    <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                      {rule.vm_namespace || '-'} / {rule.vm_rule_name || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip size="small" label={rule.enabled ? '启用' : '停用'} color={rule.enabled ? 'success' : 'default'} variant="outlined" />
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="删除">
                      <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, rule })}>
                        <DeleteOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
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

      <Card sx={{ mt: 2, p: 2 }}>
        <Typography variant="subtitle2" sx={{ mb: 1.5 }}>最近告警事件</Typography>
        {events.length === 0 ? (
          <EmptyState title="暂无事件" description="vmalert 触发后的事件会在这里展示。" />
        ) : (
          <Table size="small">
            <TableBody>
              {events.map((event) => (
                <TableRow key={event.id}>
                  <TableCell>{event.rule_name}</TableCell>
                  <TableCell><Chip size="small" label={event.status} /></TableCell>
                  <TableCell><Chip size="small" label={levelMeta[event.level]?.label || event.level} color={levelMeta[event.level]?.color || 'default'} /></TableCell>
                  <TableCell>{new Date(event.start_time).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>新建 VMRule 告警规则</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>租户</InputLabel>
                <Select value={form.tenant_id} label="租户" onChange={(e) => setForm({ ...form, tenant_id: e.target.value })}>
                  {tenants.map((tenant) => <MenuItem key={tenant.id} value={tenant.id}>{tenant.tenant_name}</MenuItem>)}
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
                <InputLabel>规则类型</InputLabel>
                <Select value={form.rule_type} label="规则类型" onChange={(e) => setForm({ ...form, rule_type: e.target.value })}>
                  <MenuItem value="metrics">Metrics / PromQL</MenuItem>
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
          <Button startIcon={<AddIcon />} variant="contained" disabled={saving || !form.tenant_id || !form.rule_name || !form.query} onClick={handleCreate}>
            {saving ? '创建中...' : '创建'}
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
