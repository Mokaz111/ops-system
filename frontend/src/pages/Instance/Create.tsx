import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Alert, Autocomplete, Box, Button, Card, CardContent, FormControl, FormHelperText,
  Grid, InputLabel, MenuItem, Select, TextField, Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useSnackbar } from 'notistack';
import { instanceAPI } from '../../api/instance';
import { workspaceAPI } from '../../api/workspace';
import { zoneAPI, type Zone } from '../../api/zone';
import { grafanaInstanceAPI, type GrafanaInstance } from '../../api/grafanaInstance';
import { extractApiError } from '../../api';
import type { Workspace } from '../../types/api';
import { useAuthStore } from '../../stores/useAuthStore';

const retentionOptions = [
  { value: 7, label: '7 天' },
  { value: 15, label: '15 天' },
  { value: 30, label: '30 天' },
  { value: 60, label: '60 天' },
  { value: 90, label: '90 天' },
];

export default function InstanceCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();

  useEffect(() => {
    if (user && user.role !== 'admin') {
      enqueueSnackbar('仅管理员可创建指标空间', { variant: 'warning' });
      navigate('/instances', { replace: true });
    }
  }, [user, navigate, enqueueSnackbar]);

  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [selectedWorkspace, setSelectedWorkspace] = useState<Workspace | null>(null);
  const [form, setForm] = useState({
    instance_name: '',
    zone_id: searchParams.get('zone_id') || '',
    grafana_instance_id: '',
    retention: 15,
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      const [tRes, zRes, gRes] = await Promise.allSettled([
        workspaceAPI.list({ page: 1, page_size: 200 }),
        zoneAPI.list({ page: 1, page_size: 100 }),
        grafanaInstanceAPI.list({ page: 1, page_size: 100 }),
      ]);
      if (tRes.status === 'fulfilled') setWorkspaces(tRes.value.data.data?.items || []);
      if (zRes.status === 'fulfilled') setZones(zRes.value.data.data?.items || []);
      if (gRes.status === 'fulfilled') setGrafanaHosts(gRes.value.data.data?.items || []);
      const wsId = searchParams.get('workspace_id');
      if (wsId && tRes.status === 'fulfilled') {
        const found = (tRes.value.data.data?.items || []).find((t: Workspace) => t.id === wsId);
        if (found) setSelectedWorkspace(found);
      }
      setLoading(false);
    };
    load();
  }, [searchParams]);

  const validate = useCallback((): boolean => {
    const e: Record<string, string> = {};
    if (!selectedWorkspace) e.tenant = '请选择工作空间';
    if (!form.instance_name.trim()) e.instance_name = '请输入指标空间名称';
    if (!form.zone_id) e.zone_id = '请选择可用区（需已 InitShared）';
    setErrors(e);
    return Object.keys(e).length === 0;
  }, [selectedWorkspace, form]);

  const handleSubmit = async () => {
    if (!validate()) return;
    setSaving(true);
    try {
      await instanceAPI.create({
        workspace_id: selectedWorkspace!.id,
        zone_id: form.zone_id,
        instance_name: form.instance_name.trim(),
        instance_type: 'metrics',
        template_type: 'shared',
        spec: JSON.stringify({ mode: 'shared', retention: form.retention }),
        grafana_instance_id: form.grafana_instance_id || undefined,
      });
      enqueueSnackbar(`指标空间「${form.instance_name}」创建成功`, { variant: 'success' });
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
          <Typography variant="h5" sx={{ fontWeight: 600 }}>创建指标空间</Typography>
          <Typography variant="body2" color="text.secondary">
            在 Zone 共享 VM 池上绑定 Metric Workspace；VMUser 由工作空间开通时创建
          </Typography>
        </Box>
      </Box>

      {loading ? (
        <Card><CardContent><Typography color="text.secondary">加载中...</Typography></CardContent></Card>
      ) : (
        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 8 }}>
            <Card>
              <CardContent>
                <Autocomplete
                  options={workspaces}
                  value={selectedWorkspace}
                  onChange={(_, v) => { setSelectedWorkspace(v); if (v) setErrors((p) => ({ ...p, tenant: '' })); }}
                  getOptionLabel={(t) => t.workspace_name}
                  isOptionEqualToValue={(a, b) => a.id === b.id}
                  renderInput={(params) => (
                    <TextField {...params} label="所属工作空间" required error={!!errors.tenant} helperText={errors.tenant} sx={{ mb: 2.5 }} />
                  )}
                />
                <TextField
                  fullWidth label="指标空间名称" value={form.instance_name}
                  onChange={(e) => setForm({ ...form, instance_name: e.target.value })}
                  sx={{ mb: 2.5 }} required error={!!errors.instance_name}
                  helperText={errors.instance_name || '如 payment-api-metrics'}
                />
                <FormControl fullWidth sx={{ mb: 2.5 }} error={!!errors.zone_id}>
                  <InputLabel>可用区 *</InputLabel>
                  <Select value={form.zone_id} label="可用区 *" onChange={(e) => setForm({ ...form, zone_id: e.target.value })}>
                    {zones.filter((z) => z.status === 'active').map((z) => (
                      <MenuItem key={z.id} value={z.id}>{z.display_name || z.slug}</MenuItem>
                    ))}
                  </Select>
                  <FormHelperText>{errors.zone_id || '管理员需先对该 Zone 执行 InitShared'}</FormHelperText>
                </FormControl>
                <Alert severity="info" icon={<InfoOutlinedIcon />}>
                  SaaS 共享池模式：复用 Zone VMCluster + vmauth；业务采集请通过「业务集群」接入 VMAgent。
                </Alert>
              </CardContent>
            </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <Card>
              <CardContent>
                <FormControl fullWidth sx={{ mb: 2 }}>
                  <InputLabel>数据保留</InputLabel>
                  <Select value={form.retention} label="数据保留" onChange={(e) => setForm({ ...form, retention: e.target.value as number })}>
                    {retentionOptions.map((o) => <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>)}
                  </Select>
                </FormControl>
                <FormControl fullWidth>
                  <InputLabel>Grafana</InputLabel>
                  <Select value={form.grafana_instance_id} label="Grafana" onChange={(e) => setForm({ ...form, grafana_instance_id: e.target.value })}>
                    <MenuItem value="">继承工作空间默认</MenuItem>
                    {grafanaHosts.map((h) => <MenuItem key={h.id} value={h.id}>{h.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </CardContent>
            </Card>
            <Box sx={{ mt: 2, display: 'flex', gap: 1.5 }}>
              <Button variant="outlined" fullWidth onClick={() => navigate('/instances')}>取消</Button>
              <Button variant="contained" fullWidth onClick={handleSubmit} disabled={saving}>
                {saving ? '创建中...' : '创建'}
              </Button>
            </Box>
          </Grid>
        </Grid>
      )}
    </Box>
  );
}
