import { useCallback, useEffect, useState } from 'react';
import {
  Alert, Box, Button, Card, CardContent, Chip, FormControl, Grid, IconButton, InputLabel,
  MenuItem, Select, Tab, Tabs, TextField, Tooltip, Typography,
} from '@mui/material';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { umodelAPI, type Entity, type LogSet, type MetricSet } from '../../api/umodel';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import { isPlatformAdmin } from '../../utils/membership';

type TabKey = 'entity' | 'metric-set' | 'log-set';

export default function UModelPage() {
  const { enqueueSnackbar } = useSnackbar();
  const user = useAuthStore((s) => s.user);
  const admin = isPlatformAdmin(user);
  const globalWorkspaceId = useWorkspaceStore((s) => s.currentId);
  // 后端 UModel 接口对平台管理员强制要求 workspace_id。
  const workspaceId = admin ? globalWorkspaceId : undefined;
  const ready = !admin || !!workspaceId;

  const [tab, setTab] = useState<TabKey>('entity');
  const [entities, setEntities] = useState<Entity[]>([]);
  const [metricSets, setMetricSets] = useState<MetricSet[]>([]);
  const [logSets, setLogSets] = useState<LogSet[]>([]);
  const [entityForm, setEntityForm] = useState({ entity_type: 'service', name: '', display_name: '' });
  const [metricSetForm, setMetricSetForm] = useState({ name: '', component: '', display_name: '' });
  const [logSetForm, setLogSetForm] = useState({ name: '', component: '', display_name: '' });
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; kind: TabKey; id?: string; name?: string }>({ open: false, kind: 'entity' });

  const load = useCallback(async () => {
    if (!ready) return;
    const params = { page: 1, page_size: 100, workspace_id: workspaceId };
    const [eRes, mRes, lRes] = await Promise.allSettled([
      umodelAPI.listEntities(params),
      umodelAPI.listMetricSets(params),
      umodelAPI.listLogSets(params),
    ]);
    if (eRes.status === 'fulfilled') setEntities(eRes.value.data.data?.items || []);
    if (mRes.status === 'fulfilled') setMetricSets(mRes.value.data.data?.items || []);
    if (lRes.status === 'fulfilled') setLogSets(lRes.value.data.data?.items || []);
  }, [ready, workspaceId]);

  // 用微任务调度首次加载，避免 effect 内同步 setState（react-hooks/set-state-in-effect）。
  useEffect(() => { queueMicrotask(load); }, [load]);

  const createEntity = async () => {
    try {
      await umodelAPI.createEntity(entityForm, workspaceId);
      enqueueSnackbar('Entity 已创建', { variant: 'success' });
      setEntityForm({ entity_type: 'service', name: '', display_name: '' });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    }
  };

  const createMetricSet = async () => {
    try {
      await umodelAPI.createMetricSet(metricSetForm, workspaceId);
      enqueueSnackbar('MetricSet 已创建', { variant: 'success' });
      setMetricSetForm({ name: '', component: '', display_name: '' });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    }
  };

  const createLogSet = async () => {
    try {
      await umodelAPI.createLogSet(logSetForm, workspaceId);
      enqueueSnackbar('LogSet 已创建', { variant: 'success' });
      setLogSetForm({ name: '', component: '', display_name: '' });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    }
  };

  const handleDelete = async () => {
    const { kind, id } = deleteDialog;
    if (!id) return;
    try {
      if (kind === 'entity') await umodelAPI.deleteEntity(id, workspaceId);
      if (kind === 'metric-set') await umodelAPI.deleteMetricSet(id, workspaceId);
      if (kind === 'log-set') await umodelAPI.deleteLogSet(id, workspaceId);
      enqueueSnackbar('已删除', { variant: 'success' });
      setDeleteDialog({ open: false, kind });
      load();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  const renderList = <T extends { id: string; name: string; display_name?: string }>(
    items: T[],
    kind: TabKey,
    extra?: (item: T) => React.ReactNode,
  ) => (
    <Card>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title="暂无数据" description="使用左侧表单创建第一条记录" />
        ) : items.map((item) => (
          <Box key={item.id} sx={{ display: 'flex', gap: 1, alignItems: 'center', py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
            {extra?.(item)}
            <Typography sx={{ fontWeight: 500 }}>{item.display_name || item.name}</Typography>
            <Typography variant="caption" color="text.secondary">{item.name}</Typography>
            <Box sx={{ flex: 1 }} />
            <Tooltip title="删除">
              <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, kind, id: item.id, name: item.display_name || item.name })}>
                <DeleteOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        ))}
      </CardContent>
    </Card>
  );

  return (
    <Box>
      <PageHeader title="UModel 元数据" subtitle="Entity / MetricSet / LogSet — 指标、日志与拓扑关联的统一模型" />

      {!ready && (
        <Alert severity="info" sx={{ mb: 2 }}>
          请先在顶部选择一个工作空间（UModel 元数据按工作空间隔离）。
        </Alert>
      )}

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
        <Tab value="entity" label="Entity" />
        <Tab value="metric-set" label="MetricSet" />
        <Tab value="log-set" label="LogSet" />
      </Tabs>

      {ready && tab === 'entity' && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>新建 Entity</Typography>
                <FormControl fullWidth size="small" sx={{ mb: 2 }}>
                  <InputLabel>类型</InputLabel>
                  <Select value={entityForm.entity_type} label="类型" onChange={(e) => setEntityForm({ ...entityForm, entity_type: e.target.value })}>
                    <MenuItem value="service">service</MenuItem>
                    <MenuItem value="k8s_cluster">k8s_cluster</MenuItem>
                    <MenuItem value="namespace">namespace</MenuItem>
                    <MenuItem value="workload">workload</MenuItem>
                  </Select>
                </FormControl>
                <TextField fullWidth size="small" label="名称" sx={{ mb: 2 }} value={entityForm.name} onChange={(e) => setEntityForm({ ...entityForm, name: e.target.value })} />
                <TextField fullWidth size="small" label="显示名" sx={{ mb: 2 }} value={entityForm.display_name} onChange={(e) => setEntityForm({ ...entityForm, display_name: e.target.value })} />
                <Button variant="contained" onClick={createEntity} disabled={!entityForm.name.trim()}>创建</Button>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 8 }}>
            {renderList(entities, 'entity', (e) => <Chip size="small" label={e.entity_type} />)}
          </Grid>
        </Grid>
      )}

      {ready && tab === 'metric-set' && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>新建 MetricSet</Typography>
                <TextField fullWidth size="small" label="名称" sx={{ mb: 2 }} value={metricSetForm.name} onChange={(e) => setMetricSetForm({ ...metricSetForm, name: e.target.value })} />
                <TextField fullWidth size="small" label="显示名" sx={{ mb: 2 }} value={metricSetForm.display_name} onChange={(e) => setMetricSetForm({ ...metricSetForm, display_name: e.target.value })} />
                <TextField fullWidth size="small" label="组件" sx={{ mb: 2 }} value={metricSetForm.component} onChange={(e) => setMetricSetForm({ ...metricSetForm, component: e.target.value })} />
                <Button variant="contained" onClick={createMetricSet} disabled={!metricSetForm.name.trim()}>创建</Button>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 8 }}>
            {renderList(metricSets, 'metric-set', (m) => m.component ? <Chip size="small" label={m.component} variant="outlined" /> : null)}
          </Grid>
        </Grid>
      )}

      {ready && tab === 'log-set' && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 2 }}>新建 LogSet</Typography>
                <TextField fullWidth size="small" label="名称" sx={{ mb: 2 }} value={logSetForm.name} onChange={(e) => setLogSetForm({ ...logSetForm, name: e.target.value })} />
                <TextField fullWidth size="small" label="显示名" sx={{ mb: 2 }} value={logSetForm.display_name} onChange={(e) => setLogSetForm({ ...logSetForm, display_name: e.target.value })} />
                <TextField fullWidth size="small" label="组件" sx={{ mb: 2 }} value={logSetForm.component} onChange={(e) => setLogSetForm({ ...logSetForm, component: e.target.value })} />
                <Button variant="contained" onClick={createLogSet} disabled={!logSetForm.name.trim()}>创建</Button>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 8 }}>
            {renderList(logSets, 'log-set', (l) => l.component ? <Chip size="small" label={l.component} variant="outlined" /> : null)}
          </Grid>
        </Grid>
      )}

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除记录"
        message={`确定要删除「${deleteDialog.name}」吗？`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false, kind: deleteDialog.kind })}
      />
    </Box>
  );
}
