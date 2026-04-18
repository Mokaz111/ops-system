import { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Step,
  StepLabel,
  Stepper,
  TextField,
  Typography,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import {
  integrationAPI,
  type IntegrationCategory,
} from '../../api/integration';
import { extractApiError } from '../../api';

interface Props {
  open: boolean;
  categories: IntegrationCategory[];
  onClose: () => void;
  onSuccess: () => void;
}

interface BasicForm {
  name: string;
  display_name: string;
  category: string;
  component: string;
  description: string;
  tags: string;
}

interface VersionForm {
  version: string;
  collector_spec: string;
  alert_spec: string;
  dashboard_spec: string;
  variables: string;
  changelog: string;
}

const defaultBasic: BasicForm = {
  name: '',
  display_name: '',
  category: 'monitor',
  component: '',
  description: '',
  tags: '',
};

const defaultVersion: VersionForm = {
  version: 'v1.0.0',
  collector_spec: '',
  alert_spec: '',
  dashboard_spec: '',
  variables: '[]',
  changelog: '首版',
};

const steps = ['模板基础信息', '首版内容', '确认并提交'];

function validateJSON(raw: string): string | null {
  if (!raw.trim()) return null;
  try {
    JSON.parse(raw);
    return null;
  } catch (e) {
    return (e as Error).message;
  }
}

export default function TemplateWizard({ open, categories, onClose, onSuccess }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const [active, setActive] = useState(0);
  const [basic, setBasic] = useState<BasicForm>(defaultBasic);
  const [version, setVersion] = useState<VersionForm>(defaultVersion);
  const [submitting, setSubmitting] = useState(false);

  const reset = () => {
    setActive(0);
    setBasic(defaultBasic);
    setVersion(defaultVersion);
  };

  const handleClose = () => {
    if (submitting) return;
    reset();
    onClose();
  };

  const validateBasic = (): string | null => {
    if (!basic.name.trim()) return '模板唯一标识 (name) 必填';
    if (!/^[a-z0-9][a-z0-9-]*$/.test(basic.name)) return 'name 需为小写字母、数字或连字符';
    if (!basic.category) return '分类必填';
    if (!basic.component.trim()) return '组件名 (component) 必填';
    return null;
  };

  const validateVersion = (): string | null => {
    if (!version.version.trim()) return '版本号必填';
    for (const [key, label] of [
      ['alert_spec', 'alert_spec'],
      ['dashboard_spec', 'dashboard_spec'],
      ['variables', 'variables'],
    ] as const) {
      const err = validateJSON(version[key]);
      if (err) return `${label} 必须是合法 JSON：${err}`;
    }
    return null;
  };

  const next = () => {
    if (active === 0) {
      const err = validateBasic();
      if (err) return enqueueSnackbar(err, { variant: 'warning' });
    }
    if (active === 1) {
      const err = validateVersion();
      if (err) return enqueueSnackbar(err, { variant: 'warning' });
    }
    setActive((v) => Math.min(2, v + 1));
  };

  const prev = () => setActive((v) => Math.max(0, v - 1));

  const submit = async () => {
    setSubmitting(true);
    try {
      const tagsArr = basic.tags
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const { data: tplRes } = await integrationAPI.createTemplate({
        name: basic.name.trim(),
        display_name: basic.display_name.trim() || undefined,
        category: basic.category,
        component: basic.component.trim(),
        description: basic.description.trim() || undefined,
        tags: tagsArr,
      });
      const tplId = tplRes.data?.id;
      if (!tplId) throw new Error('创建模板失败：无 id 返回');
      await integrationAPI.createVersion(tplId, {
        version: version.version.trim(),
        collector_spec: version.collector_spec,
        alert_spec: version.alert_spec,
        dashboard_spec: version.dashboard_spec,
        variables: version.variables,
        changelog: version.changelog,
      });
      enqueueSnackbar('模板上传成功', { variant: 'success' });
      reset();
      onClose();
      onSuccess();
    } catch (err) {
      // 这里的 409 / 422 通常是"模板名重复"或"版本号冲突"——后端 stage-5 已加 ErrIntegrationTemplateNameExists，
      // 用 extractApiError 把后端文案直透出来比通用"创建失败"友好得多。
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>上传接入模板</DialogTitle>
      <DialogContent sx={{ pt: '16px !important' }}>
        <Stepper activeStep={active} sx={{ mb: 3 }}>
          {steps.map((s) => (
            <Step key={s}>
              <StepLabel>{s}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {active === 0 && (
          <Grid container spacing={2}>
            <Grid size={{ xs: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="模板唯一标识 (name) *"
                value={basic.name}
                onChange={(e) => setBasic({ ...basic, name: e.target.value })}
                helperText="小写字母 / 数字 / 连字符，如 node-exporter"
              />
            </Grid>
            <Grid size={{ xs: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="显示名 (display_name)"
                value={basic.display_name}
                onChange={(e) => setBasic({ ...basic, display_name: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel>分类 *</InputLabel>
                <Select
                  label="分类 *"
                  value={basic.category}
                  onChange={(e) => setBasic({ ...basic, category: e.target.value })}
                >
                  {categories.map((c) => (
                    <MenuItem key={c.key} value={c.key}>
                      {c.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <TextField
                fullWidth
                size="small"
                label="组件 (component) *"
                value={basic.component}
                onChange={(e) => setBasic({ ...basic, component: e.target.value })}
                helperText="如 node, mysql, redis, kafka"
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField
                fullWidth
                size="small"
                multiline
                minRows={2}
                label="描述"
                value={basic.description}
                onChange={(e) => setBasic({ ...basic, description: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField
                fullWidth
                size="small"
                label="标签"
                value={basic.tags}
                onChange={(e) => setBasic({ ...basic, tags: e.target.value })}
                helperText="以英文逗号分隔，如：official,host,linux"
              />
            </Grid>
          </Grid>
        )}

        {active === 1 && (
          <Grid container spacing={2}>
            <Grid size={{ xs: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="版本号 *"
                value={version.version}
                onChange={(e) => setVersion({ ...version, version: e.target.value })}
                helperText="建议 SemVer，如 v1.0.0"
              />
            </Grid>
            <Grid size={{ xs: 8 }}>
              <TextField
                fullWidth
                size="small"
                label="变更说明 (changelog)"
                value={version.changelog}
                onChange={(e) => setVersion({ ...version, changelog: e.target.value })}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                采集配置 (collector_spec)
              </Typography>
              <TextField
                fullWidth
                multiline
                minRows={6}
                placeholder={`# 多段 YAML 以 --- 分隔\napiVersion: operator.victoriametrics.com/v1beta1\nkind: VMPodScrape\nmetadata:\n  name: {{ .Values.component }}-{{ .Ctx.TenantID }}\n  namespace: {{ .Ctx.Namespace }}\nspec:\n  podMetricsEndpoints: []\n`}
                value={version.collector_spec}
                onChange={(e) => setVersion({ ...version, collector_spec: e.target.value })}
                InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                告警配置 (alert_spec, JSON)
              </Typography>
              <TextField
                fullWidth
                multiline
                minRows={4}
                placeholder={`{\n  "vmrule": "...yaml...",\n  "n9e": [],\n  "alert_targets": ["vmrule"]\n}`}
                value={version.alert_spec}
                onChange={(e) => setVersion({ ...version, alert_spec: e.target.value })}
                InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                大盘 (dashboard_spec, JSON 数组)
              </Typography>
              <TextField
                fullWidth
                multiline
                minRows={4}
                placeholder={`[ { "title": "Node Overview", "uid": "node-overview", "panels": [] } ]`}
                value={version.dashboard_spec}
                onChange={(e) => setVersion({ ...version, dashboard_spec: e.target.value })}
                InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
              />
            </Grid>
            <Grid size={{ xs: 12 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                可配置变量 (variables, JSON 数组)
              </Typography>
              <TextField
                fullWidth
                multiline
                minRows={4}
                placeholder={`[ { "name": "component", "label": "组件", "default": "node", "required": true } ]`}
                value={version.variables}
                onChange={(e) => setVersion({ ...version, variables: e.target.value })}
                InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
              />
            </Grid>
          </Grid>
        )}

        {active === 2 && (
          <Box>
            <Alert severity="info" sx={{ mb: 2 }}>
              下方为将提交的快照；确认无误后点击"提交"。模板会以 active 状态登记；可在列表中继续追加版本。
            </Alert>
            <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>模板</Typography>
              <Typography variant="body2">
                {basic.name} · {basic.component} · 分类 {basic.category}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {basic.description || '（无描述）'}
              </Typography>
            </Paper>
            <Paper variant="outlined" sx={{ p: 2 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>首版</Typography>
              <Typography variant="body2">{version.version} · {version.changelog || '—'}</Typography>
              <Typography variant="caption" color="text.secondary">
                collector_spec {version.collector_spec ? `${version.collector_spec.length} 字符` : '未填写'} ·
                alert_spec {version.alert_spec ? `${version.alert_spec.length} 字符` : '未填写'} ·
                dashboard_spec {version.dashboard_spec ? `${version.dashboard_spec.length} 字符` : '未填写'}
              </Typography>
            </Paper>
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={handleClose} disabled={submitting}>取消</Button>
        <Box sx={{ flex: 1 }} />
        {active > 0 && <Button onClick={prev} disabled={submitting}>上一步</Button>}
        {active < 2 ? (
          <Button variant="contained" onClick={next}>下一步</Button>
        ) : (
          <Button variant="contained" onClick={submit} disabled={submitting}>
            {submitting ? '提交中...' : '提交'}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
