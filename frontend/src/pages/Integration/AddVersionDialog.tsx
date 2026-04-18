import { useState, useEffect } from 'react';
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  TextField,
  Typography,
} from '@mui/material';
import { useSnackbar } from 'notistack';
import {
  integrationAPI,
  type IntegrationTemplate,
  type IntegrationTemplateVersion,
} from '../../api/integration';

interface Props {
  open: boolean;
  template: IntegrationTemplate | null;
  onClose: () => void;
  onSuccess: () => void;
}

function validateJSON(raw: string): string | null {
  if (!raw.trim()) return null;
  try {
    JSON.parse(raw);
    return null;
  } catch (e) {
    return (e as Error).message;
  }
}

export default function AddVersionDialog({ open, template, onClose, onSuccess }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const [version, setVersion] = useState('');
  const [collector, setCollector] = useState('');
  const [alertSpec, setAlertSpec] = useState('');
  const [dashboard, setDashboard] = useState('');
  const [variables, setVariables] = useState('[]');
  const [changelog, setChangelog] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [baseVersion, setBaseVersion] = useState<IntegrationTemplateVersion | null>(null);

  useEffect(() => {
    if (!open || !template) {
      setBaseVersion(null);
      return;
    }
    setVersion('');
    setChangelog('');
    let alive = true;
    (async () => {
      try {
        const { data: res } = await integrationAPI.listVersions(template.id);
        if (!alive) return;
        const list = res.data || [];
        const latest = list.find((v) => v.version === template.latest_version) || list[0] || null;
        setBaseVersion(latest);
        setCollector(latest?.collector_spec || '');
        setAlertSpec(latest?.alert_spec || '');
        setDashboard(latest?.dashboard_spec || '');
        setVariables(latest?.variables || '[]');
      } catch {
        if (!alive) return;
        setBaseVersion(null);
        setCollector('');
        setAlertSpec('');
        setDashboard('');
        setVariables('[]');
      }
    })();
    return () => {
      alive = false;
    };
  }, [open, template]);

  const submit = async () => {
    if (!template) return;
    if (!version.trim()) {
      enqueueSnackbar('版本号必填', { variant: 'warning' });
      return;
    }
    for (const [raw, label] of [
      [alertSpec, 'alert_spec'],
      [dashboard, 'dashboard_spec'],
      [variables, 'variables'],
    ] as const) {
      const err = validateJSON(raw);
      if (err) {
        enqueueSnackbar(`${label} 必须是合法 JSON：${err}`, { variant: 'warning' });
        return;
      }
    }
    setSubmitting(true);
    try {
      await integrationAPI.createVersion(template.id, {
        version: version.trim(),
        collector_spec: collector,
        alert_spec: alertSpec,
        dashboard_spec: dashboard,
        variables,
        changelog,
      });
      enqueueSnackbar('新版本已登记', { variant: 'success' });
      onClose();
      onSuccess();
    } catch (e) {
      enqueueSnackbar(`提交失败：${(e as Error).message}`, { variant: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={() => !submitting && onClose()} maxWidth="md" fullWidth>
      <DialogTitle>追加版本 · {template?.display_name || template?.name}</DialogTitle>
      <DialogContent sx={{ pt: '16px !important' }}>
        {baseVersion && (
          <Alert severity="info" sx={{ mb: 2 }}>
            以当前版本 <b>{baseVersion.version}</b> 为基底；只需修改变化部分。提交成功后 latest_version 会自动切换到新版本。
          </Alert>
        )}
        <Grid container spacing={2}>
          <Grid size={{ xs: 4 }}>
            <TextField
              fullWidth
              size="small"
              label="新版本号 *"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              helperText="建议 SemVer，如 v1.1.0"
            />
          </Grid>
          <Grid size={{ xs: 8 }}>
            <TextField
              fullWidth
              size="small"
              label="变更说明"
              value={changelog}
              onChange={(e) => setChangelog(e.target.value)}
            />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>collector_spec (YAML)</Typography>
            <TextField
              fullWidth
              multiline
              minRows={5}
              value={collector}
              onChange={(e) => setCollector(e.target.value)}
              InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
            />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>alert_spec (JSON)</Typography>
            <TextField
              fullWidth
              multiline
              minRows={4}
              value={alertSpec}
              onChange={(e) => setAlertSpec(e.target.value)}
              InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
            />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>dashboard_spec (JSON)</Typography>
            <TextField
              fullWidth
              multiline
              minRows={4}
              value={dashboard}
              onChange={(e) => setDashboard(e.target.value)}
              InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
            />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>variables (JSON)</Typography>
            <TextField
              fullWidth
              multiline
              minRows={4}
              value={variables}
              onChange={(e) => setVariables(e.target.value)}
              InputProps={{ sx: { fontFamily: 'monospace', fontSize: 12.5 } }}
            />
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={onClose} disabled={submitting}>取消</Button>
        <Button variant="contained" onClick={submit} disabled={submitting || !version.trim()}>
          {submitting ? '提交中...' : '提交'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
