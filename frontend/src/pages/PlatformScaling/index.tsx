import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Button,
  Card,
  CardContent,
  Chip,
  FormControl,
  Grid,
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
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { platformAPI } from '../../api/platform';
import type {
  PlatformInitSharedClusterPlan,
  PlatformScaleAuditItem,
  PlatformScaleTarget,
  PlatformScaleVMClusterPlan,
} from '../../types/api';

export default function PlatformScalingPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(false);
  const [initLoading, setInitLoading] = useState(false);
  const [targets, setTargets] = useState<PlatformScaleTarget[]>([]);
  const [targetsLoading, setTargetsLoading] = useState(false);
  const [applyConfirmOpen, setApplyConfirmOpen] = useState(false);
  const [initApplyConfirmOpen, setInitApplyConfirmOpen] = useState(false);
  const [plan, setPlan] = useState<PlatformScaleVMClusterPlan | null>(null);
  const [initPlan, setInitPlan] = useState<PlatformInitSharedClusterPlan | null>(null);
  const [audits, setAudits] = useState<PlatformScaleAuditItem[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);
  const [detailAudit, setDetailAudit] = useState<PlatformScaleAuditItem | null>(null);
  const [auditFilter, setAuditFilter] = useState({
    target_id: '',
    status: '' as '' | 'success' | 'failed' | 'replayed',
    operator: '',
    start_time: '',
    end_time: '',
  });
  const [form, setForm] = useState({
    target_id: '',
    vmselect_replicas: 2,
    vminsert_replicas: 2,
    vmstorage_replicas: 2,
    storage_size: '200Gi',
  });
  const [initForm, setInitForm] = useState({
    namespace: 'monitoring',
    release_name: 'vm-shared-stack',
  });

  useEffect(() => {
    const fetchTargets = async () => {
      setTargetsLoading(true);
      try {
        const { data: res } = await platformAPI.listVMClusterTargets();
        const rows = res.data || [];
        setTargets(rows);
        if (rows.length > 0) {
          setForm((prev) => ({ ...prev, target_id: prev.target_id || rows[0].id }));
        }
      } catch {
        enqueueSnackbar('加载平台扩容目标失败', { variant: 'error' });
      } finally {
        setTargetsLoading(false);
      }
    };
    fetchTargets();
  }, [enqueueSnackbar]);

  const toRFC3339 = useCallback((localTime: string) => {
    if (!localTime) {
      return undefined;
    }
    const parsed = new Date(localTime);
    if (Number.isNaN(parsed.getTime())) {
      return undefined;
    }
    return parsed.toISOString();
  }, []);

  const prettySpecPatch = (raw: string) => {
    if (!raw) {
      return '{}';
    }
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  };

  const refreshAudits = useCallback(async () => {
    const { data: res } = await platformAPI.listAudits({
      page: 1,
      page_size: 20,
      target_id: auditFilter.target_id || undefined,
      status: auditFilter.status || undefined,
      operator: auditFilter.operator || undefined,
      start_time: toRFC3339(auditFilter.start_time),
      end_time: toRFC3339(auditFilter.end_time),
    });
    setAudits(res.data?.items || []);
  }, [auditFilter, toRFC3339]);

  useEffect(() => {
    const fetchAudits = async () => {
      setAuditLoading(true);
      try {
        await refreshAudits();
      } catch {
        enqueueSnackbar('加载变更历史失败', { variant: 'error' });
      } finally {
        setAuditLoading(false);
      }
    };
    fetchAudits();
  }, [enqueueSnackbar, refreshAudits]);

  const selectedTarget = useMemo(
    () => targets.find((t) => t.id === form.target_id),
    [targets, form.target_id],
  );

  const handleDryRun = async () => {
    setLoading(true);
    try {
      const { data: res } = await platformAPI.scaleVMCluster({
        ...form,
        dry_run: true,
      });
      setPlan(res.data);
      enqueueSnackbar('Dry-run 成功，已生成变更预览', { variant: 'success' });
    } catch {
      enqueueSnackbar('Dry-run 失败，请检查参数', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async () => {
    setLoading(true);
    try {
      const idempotencyKey = `${form.target_id}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
      const { data: res } = await platformAPI.scaleVMCluster({
        ...form,
        dry_run: false,
      }, { idempotencyKey });
      setPlan(res.data);
      await refreshAudits();
      enqueueSnackbar('扩容配置已提交', { variant: 'success' });
      setApplyConfirmOpen(false);
    } catch {
      enqueueSnackbar('提交失败，请稍后重试', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const handleInitDryRun = async () => {
    setInitLoading(true);
    try {
      const { data: res } = await platformAPI.initSharedCluster({
        ...initForm,
        dry_run: true,
      });
      setInitPlan(res.data);
      enqueueSnackbar('共享集群初始化 dry-run 成功', { variant: 'success' });
    } catch {
      enqueueSnackbar('共享集群初始化 dry-run 失败', { variant: 'error' });
    } finally {
      setInitLoading(false);
    }
  };

  const handleInitApply = async () => {
    setInitLoading(true);
    try {
      const { data: res } = await platformAPI.initSharedCluster({
        ...initForm,
        dry_run: false,
      });
      setInitPlan(res.data);
      await refreshAudits();
      enqueueSnackbar('共享集群初始化已提交', { variant: 'success' });
      setInitApplyConfirmOpen(false);
    } catch {
      enqueueSnackbar('共享集群初始化提交失败', { variant: 'error' });
    } finally {
      setInitLoading(false);
    }
  };

  return (
    <Box>
      <PageHeader title="平台扩容" subtitle="Iteration 3: 预览后确认应用（含审计）" />

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Alert severity="info" sx={{ mb: 2 }}>
            仅支持对平台注册的 VMCluster 目标进行扩容预览，避免手工输入 namespace/name 误操作。
          </Alert>
          <Alert severity="warning" sx={{ mb: 2 }}>
            应用变更需要二次确认，且请求会携带 Idempotency-Key 以防重复提交。
          </Alert>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12 }}>
              <FormControl fullWidth size="small">
                <InputLabel>扩容目标</InputLabel>
                <Select
                  value={form.target_id}
                  label="扩容目标"
                  onChange={(e) => setForm({ ...form, target_id: e.target.value })}
                  disabled={targetsLoading || targets.length === 0}
                >
                  {targets.map((t) => (
                    <MenuItem key={t.id} value={t.id}>
                      {t.display_name} ({t.namespace}/{t.name})
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField fullWidth size="small" label="Scope" value={selectedTarget?.scope || '-'} disabled />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField fullWidth size="small" label="Namespace" value={selectedTarget?.namespace || '-'} disabled />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField fullWidth size="small" label="VMCluster 名称" value={selectedTarget?.name || '-'} disabled />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="vmselect 副本"
                type="number"
                value={form.vmselect_replicas}
                onChange={(e) => setForm({ ...form, vmselect_replicas: parseInt(e.target.value, 10) || 1 })}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="vminsert 副本"
                type="number"
                value={form.vminsert_replicas}
                onChange={(e) => setForm({ ...form, vminsert_replicas: parseInt(e.target.value, 10) || 1 })}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="vmstorage 副本"
                type="number"
                value={form.vmstorage_replicas}
                onChange={(e) => setForm({ ...form, vmstorage_replicas: parseInt(e.target.value, 10) || 1 })}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="存储大小"
                value={form.storage_size}
                onChange={(e) => setForm({ ...form, storage_size: e.target.value })}
              />
            </Grid>
          </Grid>
          <Box sx={{ mt: 2 }}>
            <Button variant="contained" onClick={handleDryRun} disabled={loading || !form.target_id}>
              {loading ? '执行中...' : '生成 Dry-run 预览'}
            </Button>
            <Button
              variant="outlined"
              sx={{ ml: 1 }}
              onClick={() => setApplyConfirmOpen(true)}
              disabled={loading || !form.target_id || !plan}
            >
              应用变更
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography variant="subtitle2" sx={{ mb: 1 }}>
            共享监控集群初始化（admin）
          </Typography>
          <Alert severity="info" sx={{ mb: 2 }}>
            将使用 Helm Chart `vm/victoria-metrics-k8s-stack` 初始化或升级全局共享监控集群，并启用内置 Grafana。
          </Alert>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="Namespace"
                value={initForm.namespace}
                onChange={(e) => setInitForm((prev) => ({ ...prev, namespace: e.target.value }))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="Release Name"
                value={initForm.release_name}
                onChange={(e) => setInitForm((prev) => ({ ...prev, release_name: e.target.value }))}
              />
            </Grid>
          </Grid>
          <Box sx={{ mt: 2 }}>
            <Button variant="contained" onClick={handleInitDryRun} disabled={initLoading}>
              {initLoading ? '执行中...' : '生成初始化 Dry-run'}
            </Button>
            <Button
              variant="outlined"
              sx={{ ml: 1 }}
              onClick={() => setInitApplyConfirmOpen(true)}
              disabled={initLoading || !initPlan}
            >
              应用初始化
            </Button>
          </Box>
          <Box
            component="pre"
            sx={{
              p: 2,
              borderRadius: 1,
              backgroundColor: '#f8f9fa',
              fontSize: 12,
              overflowX: 'auto',
              m: 0,
              mt: 2,
            }}
          >
            {initPlan ? JSON.stringify(initPlan, null, 2) : '暂无初始化预览，请先执行 dry-run。'}
          </Box>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="subtitle2" sx={{ mb: 1 }}>
            变更预览
          </Typography>
          <Box
            component="pre"
            sx={{
              p: 2,
              borderRadius: 1,
              backgroundColor: '#f8f9fa',
              fontSize: 12,
              overflowX: 'auto',
              m: 0,
            }}
          >
            {plan ? JSON.stringify(plan, null, 2) : '暂无预览，请先执行 dry-run。'}
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ mt: 2 }}>
        <CardContent>
          <Typography variant="subtitle2" sx={{ mb: 1 }}>
            平台扩容变更历史（仅管理员）
          </Typography>
          <Grid container spacing={2} sx={{ mb: 1 }}>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControl fullWidth size="small">
                <InputLabel>目标筛选</InputLabel>
                <Select
                  value={auditFilter.target_id}
                  label="目标筛选"
                  onChange={(e) => setAuditFilter((prev) => ({ ...prev, target_id: e.target.value }))}
                >
                  <MenuItem value="">全部目标</MenuItem>
                  {targets.map((t) => (
                    <MenuItem key={t.id} value={t.id}>
                      {t.display_name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControl fullWidth size="small">
                <InputLabel>状态筛选</InputLabel>
                <Select
                  value={auditFilter.status}
                  label="状态筛选"
                  onChange={(e) => setAuditFilter((prev) => ({
                    ...prev,
                    status: e.target.value as '' | 'success' | 'failed' | 'replayed',
                  }))}
                >
                  <MenuItem value="">全部状态</MenuItem>
                  <MenuItem value="success">success</MenuItem>
                  <MenuItem value="failed">failed</MenuItem>
                  <MenuItem value="replayed">replayed</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="操作者筛选"
                value={auditFilter.operator}
                onChange={(e) => setAuditFilter((prev) => ({ ...prev, operator: e.target.value }))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                type="datetime-local"
                label="开始时间"
                value={auditFilter.start_time}
                onChange={(e) => setAuditFilter((prev) => ({ ...prev, start_time: e.target.value }))}
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                fullWidth
                size="small"
                type="datetime-local"
                label="结束时间"
                value={auditFilter.end_time}
                onChange={(e) => setAuditFilter((prev) => ({ ...prev, end_time: e.target.value }))}
                slotProps={{ inputLabel: { shrink: true } }}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Button
                variant="text"
                onClick={() =>
                  setAuditFilter({
                    target_id: '',
                    status: '',
                    operator: '',
                    start_time: '',
                    end_time: '',
                  })
                }
              >
                重置筛选
              </Button>
            </Grid>
          </Grid>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>时间</TableCell>
                  <TableCell>操作者</TableCell>
                  <TableCell>目标</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>来源 IP</TableCell>
                  <TableCell>错误信息</TableCell>
                  <TableCell>变更详情</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {auditLoading ? (
                  <TableRow>
                    <TableCell colSpan={7}>加载中...</TableCell>
                  </TableRow>
                ) : audits.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7}>暂无变更历史</TableCell>
                  </TableRow>
                ) : audits.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell>{new Date(a.created_at).toLocaleString()}</TableCell>
                    <TableCell>{a.username || a.user_id || '-'}</TableCell>
                    <TableCell>{a.target_id}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={a.status}
                        color={a.status === 'success' ? 'success' : a.status === 'failed' ? 'error' : 'warning'}
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>{a.client_ip || '-'}</TableCell>
                    <TableCell>{a.error_message || '-'}</TableCell>
                    <TableCell>
                      <Button size="small" onClick={() => setDetailAudit(a)}>
                        查看
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={applyConfirmOpen}
        title="确认应用平台扩容变更"
        message="将把本次规格变更应用到选中的 VMCluster 目标。建议先执行 dry-run 并核对预览内容。是否继续？"
        confirmLabel="确认应用"
        severity="warning"
        loading={loading}
        onConfirm={handleApply}
        onCancel={() => setApplyConfirmOpen(false)}
      />

      <ConfirmDialog
        open={initApplyConfirmOpen}
        title="确认初始化共享监控集群"
        message="将对共享监控集群执行 helm install/upgrade。建议先执行 dry-run 并核对预览内容。是否继续？"
        confirmLabel="确认初始化"
        severity="warning"
        loading={initLoading}
        onConfirm={handleInitApply}
        onCancel={() => setInitApplyConfirmOpen(false)}
      />

      <Dialog open={Boolean(detailAudit)} onClose={() => setDetailAudit(null)} maxWidth="md" fullWidth>
        <DialogTitle>变更详情</DialogTitle>
        <DialogContent dividers>
          <Typography variant="body2" sx={{ mb: 1 }}>
            操作者：{detailAudit?.username || detailAudit?.user_id || '-'} | 目标：{detailAudit?.target_id || '-'} |
            状态：{detailAudit?.status || '-'}
          </Typography>
          <Box
            component="pre"
            sx={{
              p: 2,
              borderRadius: 1,
              backgroundColor: '#f8f9fa',
              fontSize: 12,
              overflowX: 'auto',
              m: 0,
            }}
          >
            {detailAudit ? prettySpecPatch(detailAudit.spec_patch) : '{}'}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailAudit(null)}>关闭</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
