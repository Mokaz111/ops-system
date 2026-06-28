import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  FormControl,
  FormHelperText,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useSnackbar } from 'notistack';
import { instanceAPI } from '../../api/instance';
import { workspaceAPI } from '../../api/workspace';
import { clusterAPI, type Cluster } from '../../api/cluster';
import { zoneAPI, type Zone } from '../../api/zone';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import type { Workspace } from '../../types/api';
import { useAuthStore } from '../../stores/useAuthStore';

// ── 预配置规格选项（仅独享集群版使用）──

interface ComponentOption {
  value: string;
  label: string;
  cpu: number;
  memory: number;
  storage?: number;
  guidance: string;
}

const vmStorageOptions: ComponentOption[] = [
  { value: 's', label: '小型 — 2C / 4G / 100Gi', cpu: 2, memory: 4, storage: 100, guidance: '≤ 50万 samples/s 写入，≤ 7天保留' },
  { value: 'm', label: '中型 — 4C / 8G / 200Gi', cpu: 4, memory: 8, storage: 200, guidance: '50万 ~ 200万 samples/s，7~30天保留' },
  { value: 'l', label: '大型 — 8C / 16G / 500Gi', cpu: 8, memory: 16, storage: 500, guidance: '200万 ~ 500万 samples/s，30~60天保留' },
  { value: 'xl', label: '超大型 — 16C / 32G / 1Ti', cpu: 16, memory: 32, storage: 1024, guidance: '≥ 500万 samples/s，≥ 60天保留' },
];

const vmSelectOptions: ComponentOption[] = [
  { value: 's', label: '小型 — 2C / 4G', cpu: 2, memory: 4, guidance: '≤ 50万 samples/s，轻量查询' },
  { value: 'm', label: '中型 — 4C / 8G', cpu: 4, memory: 8, guidance: '50万 ~ 200万 samples/s，中等查询负载' },
  { value: 'l', label: '大型 — 8C / 16G', cpu: 8, memory: 16, guidance: '≥ 200万 samples/s，高并发查询' },
];

const vmInsertOptions: ComponentOption[] = [
  { value: 's', label: '小型 — 2C / 4G', cpu: 2, memory: 4, guidance: '≤ 50万 samples/s 写入吞吐' },
  { value: 'm', label: '中型 — 4C / 8G', cpu: 4, memory: 8, guidance: '50万 ~ 200万 samples/s 写入吞吐' },
  { value: 'l', label: '大型 — 8C / 16G', cpu: 8, memory: 16, guidance: '≥ 200万 samples/s 写入吞吐' },
];

const retentionOptions = [
  { value: 7, label: '7 天', guidance: '开发/测试环境' },
  { value: 15, label: '15 天', guidance: '预发布环境' },
  { value: 30, label: '30 天', guidance: '一般生产环境' },
  { value: 60, label: '60 天', guidance: '核心业务' },
  { value: 90, label: '90 天', guidance: '合规要求' },
];

const replicaOptions = [
  { value: 1, label: '1 副本', guidance: '开发/测试，无高可用要求' },
  { value: 2, label: '2 副本', guidance: '一般生产，单节点故障自愈' },
  { value: 3, label: '3 副本', guidance: '核心生产，跨机架/可用区容灾' },
];

const templateTypeOptions = [
  {
    value: 'shared',
    label: '共享版 — 多工作空间共享 VM 集群',
    desc: '复用可用区已部署的共享 VMCluster + VMAuth。通过 VMUser 实现工作空间隔离，VMAuth 解析 Token 动态路由 select/insert 请求并支持按用户限速。零额外资源开销，秒级开通。',
  },
  {
    value: 'dedicated_single',
    label: '独享单机版 — 独立 VMSingle',
    desc: '创建独立 VMSingle 实例，单节点部署，资源隔离且成本较低，适合中小规模或非高可用场景的工作空间。',
  },
  {
    value: 'dedicated_cluster',
    label: '独享集群版 — 独立 VMCluster CR',
    desc: '创建独立 VMCluster CR → Operator 自动编排 vminsert/vmselect/vmstorage 组件。资源完全隔离，适合大规模或合规工作空间。',
  },
];

export default function InstanceCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();

  useEffect(() => {
    if (user && user.role !== 'admin') {
      enqueueSnackbar('仅管理员可创建实例', { variant: 'warning' });
      navigate('/instances', { replace: true });
    }
  }, [user, navigate, enqueueSnackbar]);

  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [selectedWorkspace, setSelectedWorkspace] = useState<Workspace | null>(null);
  const [form, setForm] = useState({
    instance_name: '',
    template_type: 'shared',
    cluster_id: '',
    zone_id: searchParams.get('zone_id') || '',
    grafana_instance_id: '',
    retention: 15,
    // 独享集群专用
    vm_storage_size: 'm',
    vm_select_size: 's',
    vm_insert_size: 's',
    replicas: 2,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      const [tRes, cRes, zRes, gRes] = await Promise.allSettled([
        workspaceAPI.list({ page: 1, page_size: 200 }),
        clusterAPI.list({ page: 1, page_size: 100 }),
        zoneAPI.list({ page: 1, page_size: 100 }),
        grafanaInstanceAPI.list({ page: 1, page_size: 100 }),
      ]);
      if (tRes.status === 'fulfilled') setWorkspaces(tRes.value.data.data?.items || []);
      if (cRes.status === 'fulfilled') setClusters(cRes.value.data.data?.items || []);
      if (zRes.status === 'fulfilled') setZones(zRes.value.data.data?.items || []);
      if (gRes.status === 'fulfilled') setGrafanaHosts(gRes.value.data.data?.items || []);
      const tenantIdFromQuery = searchParams.get('workspace_id');
      if (tenantIdFromQuery && tRes.status === 'fulfilled') {
        const found = (tRes.value.data.data?.items || []).find((t: Workspace) => t.id === tenantIdFromQuery);
        if (found) setSelectedWorkspace(found);
      }
      setLoading(false);
    };
    load();
  }, [searchParams]);

  useEffect(() => {
    if (selectedWorkspace?.template_type) {
      const mode = selectedWorkspace.template_type === 'dedicated_cluster' ? 'dedicated_cluster' : 'shared';
      setForm((prev) => ({ ...prev, template_type: mode }));
    }
  }, [selectedWorkspace]);

  useEffect(() => {
    const qZone = searchParams.get('zone_id');
    if (qZone && zones.length > 0) {
      setForm((prev) => ({ ...prev, zone_id: qZone }));
    }
  }, [searchParams, zones]);

  const isDedicated = form.template_type === 'dedicated_cluster';

  const buildSpec = useCallback(() => {
    if (isDedicated) {
      const storage = vmStorageOptions.find((o) => o.value === form.vm_storage_size)!;
      const select = vmSelectOptions.find((o) => o.value === form.vm_select_size)!;
      const insert = vmInsertOptions.find((o) => o.value === form.vm_insert_size)!;
      return JSON.stringify({
        mode: 'dedicated_cluster',
        vmstorage: { cpu: storage.cpu, memory: storage.memory, storage: storage.storage },
        vmselect: { cpu: select.cpu, memory: select.memory },
        vminsert: { cpu: insert.cpu, memory: insert.memory },
        retention: form.retention,
        replicas: form.replicas,
      });
    }
    // 共享版：无需组件配置，多工作空间隔离通过 VMUser + VMAuth token 路由实现
    return JSON.stringify({ mode: 'shared', retention: form.retention });
  }, [form, isDedicated]);

  const validate = useCallback((): boolean => {
    const e: Record<string, string> = {};
    if (!selectedWorkspace) e.tenant = '请选择工作空间';
    if (!form.instance_name.trim()) e.instance_name = '请输入实例名称';
    setErrors(e);
    return Object.keys(e).length === 0;
  }, [selectedWorkspace, form]);

  const handleSubmit = async () => {
    if (!validate()) return;
    setSaving(true);
    try {
      await instanceAPI.create({
        workspace_id: selectedWorkspace!.id,
        cluster_id: form.cluster_id || undefined,
        zone_id: form.zone_id || undefined,
        instance_name: form.instance_name.trim(),
        instance_type: 'metrics',
        template_type: form.template_type,
        spec: buildSpec(),
        grafana_instance_id: form.grafana_instance_id || undefined,
      });
      enqueueSnackbar(`实例「${form.instance_name}」创建成功`, { variant: 'success' });
      navigate('/instances');
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box>
      <Box sx={{ mb: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Button startIcon={<ArrowBackIcon />} onClick={() => navigate('/instances')}>返回</Button>
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 600 }}>创建实例</Typography>
          <Typography variant="body2" color="text.secondary">
            为工作空间部署可观测性实例 — 基于 VictoriaMetrics Operator CR 模式
          </Typography>
        </Box>
      </Box>

      {loading ? (
        <Card><CardContent><Typography color="text.secondary">加载参考数据...</Typography></CardContent></Card>
      ) : (
        <Grid container spacing={3}>
          {/* ── 左侧：基本信息 + 规格 ── */}
          <Grid size={{ xs: 12, md: 8 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>基本信息</Typography>

                <Autocomplete
                  options={workspaces}
                  value={selectedWorkspace}
                  onChange={(_, v) => { setSelectedWorkspace(v); if (v) setErrors((prev) => ({ ...prev, tenant: '' })); }}
                  getOptionLabel={(t) => t.workspace_name}
                  isOptionEqualToValue={(a, b) => a.id === b.id}
                  renderInput={(params) => (
                    <TextField {...params} label="所属工作空间" required error={!!errors.tenant} helperText={errors.tenant || '选择该实例所属的工作空间'} />
                  )}
                  renderOption={(props, t) => (
                    <li {...props} key={t.id}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                        <Typography sx={{ fontWeight: 500 }}>{t.workspace_name}</Typography>
                        <Chip size="small" label={t.template_type} variant="outlined" sx={{ ml: 'auto', height: 18, fontSize: '0.65rem' }} />
                        <Chip size="small" label={t.status} variant="outlined" color={t.status === 'active' ? 'success' : 'default'} sx={{ height: 18, fontSize: '0.65rem' }} />
                      </Box>
                    </li>
                  )}
                  sx={{ mb: 2.5 }}
                />

                <TextField
                  fullWidth label="实例名称" value={form.instance_name}
                  onChange={(e) => { setForm({ ...form, instance_name: e.target.value }); if (e.target.value.trim()) setErrors((prev) => ({ ...prev, instance_name: '' })); }}
                  sx={{ mb: 2.5 }} required error={!!errors.instance_name}
                  helperText={errors.instance_name || '字母数字与连字符，如 my-app-metrics'}
                />

                <FormControl fullWidth sx={{ mb: 2 }}>
                  <InputLabel>部署模式</InputLabel>
                  <Select value={form.template_type} label="部署模式" onChange={(e) => setForm({ ...form, template_type: e.target.value })}>
                    {templateTypeOptions.map((opt) => (
                      <MenuItem key={opt.value} value={opt.value}>
                        <Box sx={{ width: '100%', py: 0.5 }}>
                          <Typography sx={{ fontWeight: 500 }}>{opt.label}</Typography>
                          <Typography variant="caption" color="text.secondary">{opt.desc}</Typography>
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                {isDedicated ? (
                  <Alert severity="warning" icon={<InfoOutlinedIcon />} sx={{ mb: 1 }}>
                    独享集群版通过创建 VMCluster CR 部署独立集群，Operator 自动编排组件。
                    确保目标可用区已初始化共享 VMCluster（Operator CRD 已注册）。
                  </Alert>
                ) : (
                  <Alert severity="info" icon={<InfoOutlinedIcon />} sx={{ mb: 1 }}>
                    <strong>共享版无需配置组件规格。</strong>开通后系统创建 VMUser + VMRoute CR，
                    VMAuth 解析工作空间 Token 动态路由 select/insert 请求并支持按用户限速。
                    VMAgent 配置 remote write 指向 VMAuth，由 VMAuth 完成多工作空间隔离。
                  </Alert>
                )}
              </CardContent>
            </Card>

            {isDedicated ? (
              <Card sx={{ mt: 2 }}>
                <CardContent>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>独立集群组件规格</Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
                    以下配置写入 VMCluster CR，Operator 据此创建对应规格组件 Pod。
                  </Typography>

                  <Box sx={{ mb: 2.5 }}>
                    <FormControl fullWidth>
                      <InputLabel>vmstorage 规格</InputLabel>
                      <Select value={form.vm_storage_size} label="vmstorage 规格" onChange={(e) => setForm({ ...form, vm_storage_size: e.target.value })}>
                        {vmStorageOptions.map((opt) => (
                          <MenuItem key={opt.value} value={opt.value}>
                            <Box sx={{ width: '100%', py: 0.5 }}>
                              <Typography sx={{ fontWeight: 500 }}>{opt.label}</Typography>
                              <Typography variant="caption" color="text.secondary">{opt.guidance}</Typography>
                            </Box>
                          </MenuItem>
                        ))}
                      </Select>
                      <FormHelperText>负责长期数据存储与压缩。建议：{vmStorageOptions.find(o => o.value === form.vm_storage_size)?.guidance}</FormHelperText>
                    </FormControl>
                  </Box>

                  <Grid container spacing={2.5}>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <FormControl fullWidth>
                        <InputLabel>vmselect 规格</InputLabel>
                        <Select value={form.vm_select_size} label="vmselect 规格" onChange={(e) => setForm({ ...form, vm_select_size: e.target.value })}>
                          {vmSelectOptions.map((opt) => (
                            <MenuItem key={opt.value} value={opt.value}>
                              <Box sx={{ width: '100%', py: 0.5 }}>
                                <Typography sx={{ fontWeight: 500 }}>{opt.label}</Typography>
                                <Typography variant="caption" color="text.secondary">{opt.guidance}</Typography>
                              </Box>
                            </MenuItem>
                          ))}
                        </Select>
                        <FormHelperText>即时查询与去重。建议：{vmSelectOptions.find(o => o.value === form.vm_select_size)?.guidance}</FormHelperText>
                      </FormControl>
                    </Grid>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <FormControl fullWidth>
                        <InputLabel>vminsert 规格</InputLabel>
                        <Select value={form.vm_insert_size} label="vminsert 规格" onChange={(e) => setForm({ ...form, vm_insert_size: e.target.value })}>
                          {vmInsertOptions.map((opt) => (
                            <MenuItem key={opt.value} value={opt.value}>
                              <Box sx={{ width: '100%', py: 0.5 }}>
                                <Typography sx={{ fontWeight: 500 }}>{opt.label}</Typography>
                                <Typography variant="caption" color="text.secondary">{opt.guidance}</Typography>
                              </Box>
                            </MenuItem>
                          ))}
                        </Select>
                        <FormHelperText>数据写入与分片路由。建议：{vmInsertOptions.find(o => o.value === form.vm_insert_size)?.guidance}</FormHelperText>
                      </FormControl>
                    </Grid>
                  </Grid>

                  <Divider sx={{ my: 2.5 }} />
                  <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1.5 }}>数据保留 & 副本策略</Typography>
                  <Grid container spacing={2.5}>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <FormControl fullWidth>
                        <InputLabel>数据保留时长</InputLabel>
                        <Select value={form.retention} label="数据保留时长" onChange={(e) => setForm({ ...form, retention: e.target.value as number })}>
                          {retentionOptions.map((opt) => (
                            <MenuItem key={opt.value} value={opt.value}>
                              <Box sx={{ width: '100%', py: 0.5 }}><Typography>{opt.label}</Typography><Typography variant="caption" color="text.secondary">{opt.guidance}</Typography></Box>
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    </Grid>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <FormControl fullWidth>
                        <InputLabel>组件副本数</InputLabel>
                        <Select value={form.replicas} label="组件副本数" onChange={(e) => setForm({ ...form, replicas: e.target.value as number })}>
                          {replicaOptions.map((opt) => (
                            <MenuItem key={opt.value} value={opt.value}>
                              <Box sx={{ width: '100%', py: 0.5 }}><Typography>{opt.label}</Typography><Typography variant="caption" color="text.secondary">{opt.guidance}</Typography></Box>
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    </Grid>
                  </Grid>

                  <Divider sx={{ my: 2.5 }} />
                  <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>规格摘要</Typography>
                  <Grid container spacing={1} sx={{ fontSize: '0.8125rem', color: 'text.secondary' }}>
                    <Grid size={4}>vmstorage</Grid>
                    <Grid size={8}>{vmStorageOptions.find(o => o.value === form.vm_storage_size)?.label} × {form.replicas} 副本</Grid>
                    <Grid size={4}>vmselect</Grid>
                    <Grid size={8}>{vmSelectOptions.find(o => o.value === form.vm_select_size)?.label} × {form.replicas} 副本</Grid>
                    <Grid size={4}>vminsert</Grid>
                    <Grid size={8}>{vmInsertOptions.find(o => o.value === form.vm_insert_size)?.label} × {form.replicas} 副本</Grid>
                    <Grid size={4}>数据保留</Grid>
                    <Grid size={8}>{form.retention} 天</Grid>
                  </Grid>
                </CardContent>
              </Card>
            ) : (
              <Card sx={{ mt: 2 }}>
                <CardContent>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>共享集群 — 数据保留</Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    共享版复用可用区已有的 VMCluster + VMAuth 基础设施。组件规格由管理员在可用区初始化时统一配置，
                    工作空间通过 VMUser + VMRoute 实现逻辑隔离。
                  </Typography>
                  <FormControl fullWidth>
                    <InputLabel>数据保留时长</InputLabel>
                    <Select value={form.retention} label="数据保留时长" onChange={(e) => setForm({ ...form, retention: e.target.value as number })}>
                      {retentionOptions.map((opt) => (
                        <MenuItem key={opt.value} value={opt.value}>
                          <Box sx={{ width: '100%', py: 0.5 }}><Typography>{opt.label}</Typography><Typography variant="caption" color="text.secondary">{opt.guidance}</Typography></Box>
                        </MenuItem>
                      ))}
                    </Select>
                    <FormHelperText>该工作空间指标数据的保留天数</FormHelperText>
                  </FormControl>
                </CardContent>
              </Card>
            )}
          </Grid>

          {/* ── 右侧：目标环境 ── */}
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>目标环境</Typography>

                <FormControl fullWidth sx={{ mb: 2.5 }}>
                  <InputLabel>可用区</InputLabel>
                  <Select value={form.zone_id} label="可用区" onChange={(e) => setForm({ ...form, zone_id: e.target.value })}>
                    <MenuItem value="">未指定（平台默认集群）</MenuItem>
                    {zones.filter((z) => z.status === 'active').map((z) => (
                      <MenuItem key={z.id} value={z.id}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                          <Typography sx={{ fontWeight: 500 }}>{z.display_name || z.slug}</Typography>
                          <Chip size="small" label="active" variant="outlined" color="success" sx={{ ml: 'auto', height: 18, fontSize: '0.65rem' }} />
                        </Box>
                      </MenuItem>
                    ))}
                    {zones.filter((z) => z.status !== 'active').map((z) => (
                      <MenuItem key={z.id} value={z.id} disabled>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                          <Typography sx={{ color: 'text.disabled' }}>{z.display_name || z.slug}</Typography>
                          <Chip size="small" label={z.status} variant="outlined" color="default" sx={{ ml: 'auto', height: 18, fontSize: '0.65rem' }} />
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                  <FormHelperText>部署到指定可用区（需管理员先初始化该区共享 VMCluster）</FormHelperText>
                </FormControl>

                <FormControl fullWidth sx={{ mb: 2.5 }}>
                  <InputLabel>目标集群</InputLabel>
                  <Select value={form.cluster_id} label="目标集群" onChange={(e) => setForm({ ...form, cluster_id: e.target.value })}>
                    <MenuItem value="">平台默认集群</MenuItem>
                    {clusters.map((c) => (
                      <MenuItem key={c.id} value={c.id}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                          <Typography>{c.display_name || c.name}</Typography>
                          {c.in_cluster && <Chip size="small" label="in-cluster" variant="outlined" sx={{ ml: 'auto', height: 18, fontSize: '0.65rem' }} />}
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl fullWidth sx={{ mb: 2.5 }}>
                  <InputLabel>关联 Grafana</InputLabel>
                  <Select value={form.grafana_instance_id} label="关联 Grafana" onChange={(e) => setForm({ ...form, grafana_instance_id: e.target.value })}>
                    <MenuItem value="">继承工作空间默认</MenuItem>
                    {grafanaHosts.map((h) => (
                      <MenuItem key={h.id} value={h.id}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                          <Typography>{h.name}</Typography>
                          <Chip size="small" label={h.source === 'platform' ? '平台' : '工作空间'} variant="outlined" sx={{ ml: 'auto', height: 18, fontSize: '0.65rem' }} />
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </CardContent>
            </Card>

            <Box sx={{ mt: 2, display: 'flex', gap: 1.5 }}>
              <Button variant="outlined" fullWidth onClick={() => navigate('/instances')}>取消</Button>
              <Button variant="contained" fullWidth onClick={handleSubmit} disabled={saving}>
                {saving ? '创建中...' : '创建实例'}
              </Button>
            </Box>
          </Grid>
        </Grid>
      )}
    </Box>
  );
}
