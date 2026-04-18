import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Drawer,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Tab,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import UploadFileIcon from '@mui/icons-material/UploadFile';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import HistoryIcon from '@mui/icons-material/History';
import { useSearchParams } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { useAuthStore } from '../../stores/useAuthStore';
import TemplateWizard from './TemplateWizard';
import AddVersionDialog from './AddVersionDialog';
import EditTemplateDialog from './EditTemplateDialog';
import VersionManagerDrawer from './VersionManagerDrawer';
import {
  integrationAPI,
  type AppliedRef,
  type IntegrationCategory,
  type IntegrationTemplate,
  type IntegrationTemplateVersion,
  type PreflightIssue,
  type RenderedResource,
} from '../../api/integration';
import { instanceAPI } from '../../api/instance';
import { clusterAPI, type Cluster } from '../../api/cluster';
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { extractApiError } from '../../api';
import { isAbortError, makeAbortController } from '../../api/client';
import type { Instance } from '../../types/api';

interface VariableDef {
  name: string;
  label?: string;
  type?: string;
  default?: string;
  required?: boolean;
  help?: string;
  options?: string[];
}

function parseVariables(raw: string): VariableDef[] {
  if (!raw) return [];
  try {
    const data = JSON.parse(raw);
    if (Array.isArray(data)) return data as VariableDef[];
    if (Array.isArray(data?.variables)) return data.variables as VariableDef[];
  } catch {
    /* ignore */
  }
  return [];
}

export default function IntegrationPage() {
  const [searchParams] = useSearchParams();
  const defaultInstanceId = searchParams.get('instance_id') || '';
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';

  const [categories, setCategories] = useState<IntegrationCategory[]>([]);
  const [templates, setTemplates] = useState<IntegrationTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [category, setCategory] = useState('');
  const [keyword, setKeyword] = useState('');
  const [reloadTick, setReloadTick] = useState(0);

  const [wizardOpen, setWizardOpen] = useState(false);
  const [addVersionFor, setAddVersionFor] = useState<IntegrationTemplate | null>(null);
  const [editingTemplate, setEditingTemplate] = useState<IntegrationTemplate | null>(null);
  const [versionManageFor, setVersionManageFor] = useState<IntegrationTemplate | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; template?: IntegrationTemplate }>({ open: false });

  const [selected, setSelected] = useState<IntegrationTemplate | null>(null);
  const [versions, setVersions] = useState<IntegrationTemplateVersion[]>([]);
  const [currentVersion, setCurrentVersion] = useState<IntegrationTemplateVersion | null>(null);

  const [instances, setInstances] = useState<Instance[]>([]);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaHost[]>([]);
  const [instanceId, setInstanceId] = useState<string>(defaultInstanceId);
  const [grafanaHostId, setGrafanaHostId] = useState<string>('');
  const [values, setValues] = useState<Record<string, string>>({});
  const [tab, setTab] = useState<'form' | 'preview' | 'applied'>('form');
  const [rendered, setRendered] = useState<RenderedResource[]>([]);
  const [applied, setApplied] = useState<AppliedRef[]>([]);
  const [preflight, setPreflight] = useState<PreflightIssue[]>([]);
  const [installStatus, setInstallStatus] = useState<string>('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    // 列表过滤参数变化（category/keyword）会触发频繁重发，
    // 用 AbortController 把上一轮在飞的请求取消，避免后到的旧响应覆盖新结果。
    const ctl = makeAbortController();
    let alive = true;
    (async () => {
      try {
        const [cats, list, inst, cls, ghs] = await Promise.all([
          integrationAPI.listCategories({ signal: ctl.signal }),
          integrationAPI.listTemplates({ page: 1, page_size: 50, category, keyword }, { signal: ctl.signal }),
          instanceAPI.list({ page: 1, page_size: 100 }, { signal: ctl.signal }),
          clusterAPI.list({ page: 1, page_size: 100 }, { signal: ctl.signal }).catch(() => null),
          grafanaHostAPI.list({ page: 1, page_size: 100 }, { signal: ctl.signal }).catch(() => null),
        ]);
        if (!alive) return;
        setCategories(cats.data.data || []);
        setTemplates(list.data.data?.items || []);
        setInstances(inst.data.data?.items || []);
        setClusters(cls?.data.data?.items || []);
        setGrafanaHosts(ghs?.data.data?.items || []);
      } catch (err) {
        if (isAbortError(err)) return;
        if (alive) enqueueSnackbar(extractApiError(err, '加载接入中心数据失败'), { variant: 'error' });
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
      ctl.abort();
    };
  }, [category, keyword, reloadTick, enqueueSnackbar]);

  const reloadTemplates = () => setReloadTick((v) => v + 1);

  const handleDeleteTemplate = async () => {
    if (!deleteDialog.template) return;
    try {
      await integrationAPI.deleteTemplate(deleteDialog.template.id);
      enqueueSnackbar('模板删除成功', { variant: 'success' });
      setDeleteDialog({ open: false });
      reloadTemplates();
    } catch (err) {
      // 后端 stage-5 后会返回 409 ErrIntegrationTemplateInUse / ErrIntegrationVersionInUse
      // 等带语义的中文消息，extractApiError 会原样透出来。
      enqueueSnackbar(extractApiError(err, '模板删除失败'), { variant: 'error' });
    }
  };

  const openDrawer = useCallback(async (t: IntegrationTemplate) => {
    setSelected(t);
    setTab('form');
    setRendered([]);
    setApplied([]);
    setPreflight([]);
    setInstallStatus('');
    try {
      const { data: res } = await integrationAPI.listVersions(t.id);
      const vs = res.data || [];
      setVersions(vs);
      const latest = vs.find((v) => v.version === t.latest_version) || vs[0] || null;
      setCurrentVersion(latest);
      if (latest) {
        const defs = parseVariables(latest.variables);
        const init: Record<string, string> = {};
        defs.forEach((d) => {
          if (d.default !== undefined) init[d.name] = d.default;
        });
        setValues(init);
      } else {
        setValues({});
      }
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '加载版本失败'), { variant: 'error' });
    }
  }, [enqueueSnackbar]);

  const closeDrawer = () => {
    setSelected(null);
    setCurrentVersion(null);
    setVersions([]);
    setRendered([]);
    setApplied([]);
    setPreflight([]);
    setInstallStatus('');
    setValues({});
    setGrafanaHostId('');
  };

  const variableDefs = useMemo(
    () => parseVariables(currentVersion?.variables || ''),
    [currentVersion],
  );

  const targetInstance = useMemo(
    () => instances.find((i) => i.id === instanceId) || null,
    [instances, instanceId],
  );

  const targetCluster = useMemo(() => {
    if (!targetInstance?.cluster_id) return null;
    return clusters.find((c) => c.id === targetInstance.cluster_id) || null;
  }, [clusters, targetInstance]);

  const applicableGrafanaHosts = useMemo(() => {
    if (!targetInstance) return grafanaHosts;
    return grafanaHosts.filter(
      (h) => h.scope === 'platform' || (h.scope === 'tenant' && h.tenant_id === targetInstance.tenant_id),
    );
  }, [grafanaHosts, targetInstance]);

  const preview = async () => {
    if (!selected || !currentVersion || !targetInstance) {
      enqueueSnackbar('请先选择目标监控实例', { variant: 'warning' });
      return;
    }
    setBusy(true);
    try {
      const { data: res } = await integrationAPI.installPlan({
        template_id: selected.id,
        template_version: currentVersion.version,
        instance_id: targetInstance.id,
        tenant_id: targetInstance.tenant_id,
        grafana_host_id: grafanaHostId || undefined,
        values,
      });
      setRendered(res.data?.rendered || []);
      setPreflight(res.data?.preflight || []);
      setTab('preview');
      if ((res.data?.preflight || []).length > 0) {
        enqueueSnackbar('预检发现问题，请先查看"渲染预览"顶部提示', { variant: 'warning' });
      }
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '渲染失败，请检查变量填写'), { variant: 'error' });
    } finally {
      setBusy(false);
    }
  };

  const doInstall = async (force: boolean) => {
    if (!selected || !currentVersion || !targetInstance) return;
    setBusy(true);
    try {
      const { data: res } = await integrationAPI.install({
        template_id: selected.id,
        template_version: currentVersion.version,
        instance_id: targetInstance.id,
        tenant_id: targetInstance.tenant_id,
        grafana_host_id: grafanaHostId || undefined,
        values,
        force,
      });
      const data = res.data;
      setRendered(data?.rendered || []);
      setApplied(data?.applied || []);
      setPreflight(data?.preflight || []);
      const status = data?.status || data?.installation?.status || '';
      setInstallStatus(status);
      const variant =
        status === 'success'
          ? 'success'
          : status === 'partial' || status === 'preflight_failed'
          ? 'warning'
          : status === 'failed'
          ? 'error'
          : 'info';
      const msg =
        status === 'success'
          ? '安装成功：已下发到 K8s / Grafana'
          : status === 'partial'
          ? '部分资源下发失败，请查看详情'
          : status === 'failed'
          ? '下发失败，请查看详情'
          : status === 'rendered'
          ? '已登记安装记录（仅渲染，未执行 Apply）'
          : status === 'preflight_failed'
          ? '预检未通过：点击"忽略预检强制安装"可继续'
          : `已登记安装记录（${status}）`;
      enqueueSnackbar(msg, { variant });
      setTab('applied');
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '安装失败'), { variant: 'error' });
    } finally {
      setBusy(false);
    }
  };

  const install = () => doInstall(false);
  const installForce = () => doInstall(true);

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="接入中心"
        subtitle="选择模版一键下发采集 / 告警 / 大盘三件套"
        actionLabel={isAdmin ? '上传模板' : undefined}
        actionIcon={<UploadFileIcon />}
        onAction={isAdmin ? () => setWizardOpen(true) : undefined}
      />

      {defaultInstanceId && (
        <Alert severity="info" sx={{ mb: 2 }}>
          当前上下文：监控实例 {defaultInstanceId}。安装时默认作用于该实例。
        </Alert>
      )}

      <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel>分类</InputLabel>
          <Select label="分类" value={category} onChange={(e) => setCategory(e.target.value)}>
            <MenuItem value="">全部分类</MenuItem>
            {categories.map((c) => (
              <MenuItem key={c.key} value={c.key}>
                {c.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <TextField
          size="small"
          placeholder="搜索模版名称"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          sx={{ minWidth: 240 }}
        />
      </Stack>

      {templates.length === 0 ? (
        <EmptyState title="暂无模版" description="Seeder 失败？请检查后端日志。" />
      ) : (
        <Grid container spacing={2}>
          {templates.map((t) => (
            <Grid key={t.id} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
              <Card sx={{ height: '100%', position: 'relative' }}>
                <CardActionArea onClick={() => openDrawer(t)} sx={{ height: '100%' }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                      <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                        {t.display_name || t.name}
                      </Typography>
                      <Chip size="small" label={t.latest_version || 'v0'} />
                    </Box>
                    <Typography variant="caption" color="text.secondary">
                      {t.component || '-'} · {t.category || '-'}
                    </Typography>
                    <Typography
                      variant="body2"
                      color="text.secondary"
                      sx={{
                        mt: 1,
                        minHeight: 60,
                        display: '-webkit-box',
                        WebkitLineClamp: 3,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                      }}
                    >
                      {t.description || '—'}
                    </Typography>
                  </CardContent>
                </CardActionArea>
                {isAdmin && (
                  <Box sx={{ position: 'absolute', top: 4, right: 4, display: 'flex', gap: 0.5 }}>
                    <Tooltip title="编辑元数据">
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingTemplate(t);
                        }}
                      >
                        <EditOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="版本管理">
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation();
                          setVersionManageFor(t);
                        }}
                      >
                        <HistoryIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="追加版本">
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation();
                          setAddVersionFor(t);
                        }}
                      >
                        <AddIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="删除模板">
                      <IconButton
                        size="small"
                        color="error"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteDialog({ open: true, template: t });
                        }}
                      >
                        <DeleteOutlinedIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </Box>
                )}
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      <Drawer
        anchor="right"
        open={!!selected}
        onClose={closeDrawer}
        PaperProps={{ sx: { width: { xs: '100%', md: 720 } } }}
      >
        {selected && (
          <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <Box sx={{ flex: 1 }}>
                <Typography variant="h6">{selected.display_name || selected.name}</Typography>
                <Typography variant="caption" color="text.secondary">
                  {selected.component} · {selected.category}
                </Typography>
              </Box>
              <IconButton onClick={closeDrawer}>
                <CloseIcon />
              </IconButton>
            </Box>

            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {selected.description}
            </Typography>

            <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
              <FormControl size="small" sx={{ minWidth: 160 }}>
                <InputLabel>版本</InputLabel>
                <Select
                  label="版本"
                  value={currentVersion?.version || ''}
                  onChange={(e) => {
                    const v = versions.find((x) => x.version === e.target.value) || null;
                    setCurrentVersion(v);
                    if (v) {
                      const defs = parseVariables(v.variables);
                      const init: Record<string, string> = {};
                      defs.forEach((d) => {
                        if (d.default !== undefined) init[d.name] = d.default;
                      });
                      setValues(init);
                    }
                  }}
                >
                  {versions.map((v) => (
                    <MenuItem key={v.id} value={v.version}>
                      {v.version}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControl size="small" sx={{ flex: 1 }}>
                <InputLabel>目标监控实例</InputLabel>
                <Select
                  label="目标监控实例"
                  value={instanceId}
                  onChange={(e) => setInstanceId(e.target.value)}
                >
                  <MenuItem value="">请选择</MenuItem>
                  {instances.map((i) => (
                    <MenuItem key={i.id} value={i.id}>
                      {i.instance_name} ({i.instance_type})
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Stack>

            <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
              <FormControl size="small" sx={{ flex: 1 }}>
                <InputLabel>Grafana 主机（可选）</InputLabel>
                <Select
                  label="Grafana 主机（可选）"
                  value={grafanaHostId}
                  onChange={(e) => setGrafanaHostId(e.target.value)}
                >
                  <MenuItem value="">使用平台默认 Grafana</MenuItem>
                  {applicableGrafanaHosts.map((h) => (
                    <MenuItem key={h.id} value={h.id}>
                      [{h.scope === 'platform' ? '平台' : '租户'}] {h.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <Box sx={{ flex: 1, display: 'flex', alignItems: 'center' }}>
                <Typography variant="caption" color="text.secondary">
                  目标集群：
                  <b style={{ marginLeft: 4 }}>
                    {targetInstance
                      ? targetCluster
                        ? targetCluster.display_name || targetCluster.name
                        : '平台默认集群'
                      : '—'}
                  </b>
                  {targetInstance && !targetCluster && (
                    <>（该实例未绑定集群，可在实例管理页指定）</>
                  )}
                </Typography>
              </Box>
            </Stack>

            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
              <Tab value="form" label="变量配置" />
              <Tab value="preview" label={`渲染预览${rendered.length ? ` (${rendered.length})` : ''}`} />
              <Tab value="applied" label={`下发结果${applied.length ? ` (${applied.length})` : ''}`} />
            </Tabs>

            <Box sx={{ flex: 1, overflow: 'auto', mb: 2 }}>
              {tab === 'form' && (
                <>
                  {currentVersion?.changelog && (
                    <Alert severity="info" sx={{ mb: 2 }}>
                      {currentVersion.changelog}
                    </Alert>
                  )}
                  {variableDefs.length === 0 ? (
                    <Typography variant="body2" color="text.secondary">
                      该模版未定义变量。
                    </Typography>
                  ) : (
                    <Grid container spacing={2}>
                      {variableDefs.map((v) => (
                        <Grid key={v.name} size={{ xs: 12, md: 6 }}>
                          <TextField
                            fullWidth
                            size="small"
                            label={`${v.label || v.name}${v.required ? ' *' : ''}`}
                            helperText={v.help || v.name}
                            value={values[v.name] ?? ''}
                            onChange={(e) => setValues((prev) => ({ ...prev, [v.name]: e.target.value }))}
                          />
                        </Grid>
                      ))}
                    </Grid>
                  )}
                </>
              )}

              {tab === 'preview' && (
                <>
                  {preflight.length > 0 && (
                    <Alert severity="warning" sx={{ mb: 2 }}>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>
                        预检发现 {preflight.length} 项问题：
                      </Typography>
                      <Stack spacing={0.5} sx={{ mt: 0.5 }}>
                        {preflight.map((p, i) => (
                          <Typography key={i} variant="caption" sx={{ wordBreak: 'break-all' }}>
                            · [{p.reason}] {p.part}
                            {p.apiVersion ? ` ${p.apiVersion}` : ''}
                            {p.kind ? `/${p.kind}` : ''}
                            {p.name ? ` ${p.name}` : ''}
                            {p.error ? ` — ${p.error}` : ''}
                          </Typography>
                        ))}
                      </Stack>
                    </Alert>
                  )}
                  {rendered.length === 0 ? (
                    <Alert severity="info">点击下方"预览"按钮以生成渲染结果。</Alert>
                  ) : (
                    <Stack spacing={2}>
                      {rendered.map((r, idx) => (
                        <Paper key={idx} variant="outlined" sx={{ p: 1.5 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <Chip size="small" label={r.part} color="primary" />
                            <Typography variant="subtitle2">
                              {r.kind} / {r.name}
                            </Typography>
                          </Box>
                          <Box
                            component="pre"
                            sx={{
                              m: 0,
                              p: 1.5,
                              bgcolor: 'background.default',
                              borderRadius: 1,
                              fontSize: 11.5,
                              overflow: 'auto',
                              maxHeight: 300,
                            }}
                          >
                            {r.yaml || r.dashboard}
                          </Box>
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </>
              )}

              {tab === 'applied' && (
                <>
                  {installStatus && (
                    <Alert
                      severity={
                        installStatus === 'success'
                          ? 'success'
                          : installStatus === 'partial' || installStatus === 'preflight_failed'
                          ? 'warning'
                          : installStatus === 'failed'
                          ? 'error'
                          : 'info'
                      }
                      sx={{ mb: 2 }}
                    >
                      安装状态：<b>{installStatus}</b>
                      {installStatus === 'rendered' && '（当前后端未启用 K8s/Grafana 客户端，仅登记渲染结果）'}
                      {installStatus === 'preflight_failed' && '（预检未通过，已中止下发；见下方 preflight 详情）'}
                    </Alert>
                  )}
                  {installStatus === 'preflight_failed' && preflight.length > 0 && (
                    <Stack spacing={1} sx={{ mb: 2 }}>
                      {preflight.map((p, i) => (
                        <Alert key={i} severity="warning" variant="outlined">
                          <Typography variant="body2" sx={{ wordBreak: 'break-all' }}>
                            [{p.reason}] {p.part}
                            {p.apiVersion ? ` ${p.apiVersion}` : ''}
                            {p.kind ? `/${p.kind}` : ''}
                            {p.name ? ` ${p.name}` : ''}
                          </Typography>
                          {p.error && (
                            <Typography variant="caption" color="text.secondary" sx={{ wordBreak: 'break-all' }}>
                              {p.error}
                            </Typography>
                          )}
                        </Alert>
                      ))}
                    </Stack>
                  )}
                  {applied.length === 0 && installStatus !== 'preflight_failed' ? (
                    <Alert severity="info">点击下方"安装"按钮以真实下发到 K8s / Grafana。</Alert>
                  ) : applied.length === 0 ? null : (
                    <Stack spacing={1.5}>
                      {applied.map((a, idx) => (
                        <Paper key={idx} variant="outlined" sx={{ p: 1.5 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5, flexWrap: 'wrap' }}>
                            <Chip size="small" label={a.part} color="primary" />
                            <Chip size="small" label={a.target} variant="outlined" />
                            <Chip
                              size="small"
                              color={a.status === 'success' ? 'success' : a.status === 'failed' ? 'error' : 'default'}
                              label={a.status || '-'}
                            />
                            {a.action && <Chip size="small" variant="outlined" label={a.action} />}
                          </Box>
                          <Typography variant="body2" sx={{ wordBreak: 'break-all' }}>
                            {a.target === 'k8s'
                              ? `${a.apiVersion || ''} ${a.kind || ''} ${a.namespace ? a.namespace + '/' : ''}${a.name || ''}`
                              : `Dashboard ${a.name || ''}${a.uid ? ` (uid=${a.uid})` : ''} org=${a.grafana_org || ''}`}
                          </Typography>
                          {a.error && (
                            <Typography variant="caption" color="error" sx={{ wordBreak: 'break-all' }}>
                              {a.error}
                            </Typography>
                          )}
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </>
              )}
            </Box>

            <Stack direction="row" spacing={1} justifyContent="flex-end">
              <Tooltip title="渲染并预览 YAML（不写库）">
                <span>
                  <Button
                    startIcon={<VisibilityOutlinedIcon />}
                    onClick={preview}
                    disabled={busy || !instanceId}
                  >
                    预览
                  </Button>
                </span>
              </Tooltip>
              {installStatus === 'preflight_failed' && (
                <Tooltip title="忽略预检问题强制下发（可能部分失败）">
                  <span>
                    <Button
                      color="warning"
                      variant="outlined"
                      onClick={installForce}
                      disabled={busy || !instanceId}
                    >
                      忽略预检强制安装
                    </Button>
                  </span>
                </Tooltip>
              )}
              <Button
                variant="contained"
                startIcon={<CheckCircleOutlineIcon />}
                onClick={install}
                disabled={busy || !instanceId}
              >
                安装
              </Button>
            </Stack>
          </Box>
        )}
      </Drawer>

      <TemplateWizard
        open={wizardOpen}
        categories={categories}
        onClose={() => setWizardOpen(false)}
        onSuccess={reloadTemplates}
      />

      <AddVersionDialog
        open={!!addVersionFor}
        template={addVersionFor}
        onClose={() => setAddVersionFor(null)}
        onSuccess={reloadTemplates}
      />

      <EditTemplateDialog
        open={!!editingTemplate}
        template={editingTemplate}
        categories={categories}
        onClose={() => setEditingTemplate(null)}
        onSuccess={reloadTemplates}
      />

      <VersionManagerDrawer
        open={!!versionManageFor}
        template={versionManageFor}
        onClose={() => setVersionManageFor(null)}
        onSuccess={reloadTemplates}
      />

      <ConfirmDialog
        open={deleteDialog.open}
        title="删除接入模板"
        message={
          // 后端 stage-5 行为：模板删除会一并软删所有版本；若仍有"活跃安装"会被 409 拒绝。
          // 这里描述要与实际语义一致，避免误导用户以为"会顺带卸载在跑的安装"。
          `确定要删除模板「${deleteDialog.template?.display_name || deleteDialog.template?.name}」吗？` +
          `\n所有历史版本将一并下架（软删除）；若该模板仍存在活跃的安装记录，删除会被拒绝，请先到对应实例处卸载。`
        }
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDeleteTemplate}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
