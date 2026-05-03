import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
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
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import UploadOutlinedIcon from '@mui/icons-material/UploadOutlined';
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { grafanaAPI } from '../../api/grafana';
import { grafanaHostAPI, type GrafanaHost } from '../../api/grafanaHost';
import { extractApiError } from '../../api';
import type { GrafanaDashboard, GrafanaOrg } from '../../types/api';

export default function DashboardMgmtPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(true);
  const [orgs, setOrgs] = useState<GrafanaOrg[]>([]);
  const [selectedOrgId, setSelectedOrgId] = useState<number | ''>('');
  const [dashboards, setDashboards] = useState<GrafanaDashboard[]>([]);
  const [dashLoading, setDashLoading] = useState(false);

  const [importOpen, setImportOpen] = useState(false);
  const [importJson, setImportJson] = useState('');
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; db?: GrafanaDashboard }>({ open: false });
  const [saving, setSaving] = useState(false);

  const [grafanaHosts, setGrafanaHosts] = useState<GrafanaHost[]>([]);

  const fetchHosts = useCallback(async () => {
    try {
      const { data: res } = await grafanaHostAPI.list({ page: 1, page_size: 100 });
      setGrafanaHosts(res.data?.items || []);
    } catch { /* optional */ }
  }, []);

  useEffect(() => { fetchHosts(); }, [fetchHosts]);

  const hostId = grafanaHosts.find((h) => h.scope === 'platform')?.id || '';

  const grafanaUrl = import.meta.env.VITE_GRAFANA_URL || '/grafana';

  useEffect(() => {
    if (!hostId) { setLoading(false); return; }
    (async () => {
      try {
        const { data: res } = await grafanaAPI.listOrgs(hostId);
        setOrgs(res.data || []);
      } catch {
        // orgs optional
      } finally {
        setLoading(false);
      }
    })();
  }, [hostId]);

  const fetchDashboards = useCallback(async () => {
    if (selectedOrgId === '' || !hostId) {
      setDashboards([]);
      return;
    }
    setDashLoading(true);
    try {
      const { data: res } = await grafanaAPI.listDashboards(hostId, selectedOrgId as number);
      setDashboards(res.data || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取 Dashboard 列表失败'), { variant: 'error' });
    } finally {
      setDashLoading(false);
    }
  }, [selectedOrgId, hostId, enqueueSnackbar]);

  useEffect(() => {
    fetchDashboards();
  }, [fetchDashboards]);

  const handleImport = async () => {
    if (selectedOrgId === '' || !importJson.trim() || !hostId) return;
    setSaving(true);
    try {
      const dashboard = JSON.parse(importJson);
      await grafanaAPI.importDashboard(hostId, selectedOrgId as number, { dashboard, overwrite: true });
      enqueueSnackbar('Dashboard 导入成功', { variant: 'success' });
      setImportOpen(false);
      setImportJson('');
      fetchDashboards();
    } catch (err) {
      if (err instanceof SyntaxError) {
        enqueueSnackbar('JSON 格式无效', { variant: 'error' });
      } else {
        enqueueSnackbar(extractApiError(err, '导入失败'), { variant: 'error' });
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (selectedOrgId === '' || !deleteDialog.db || !hostId) return;
    try {
      await grafanaAPI.deleteDashboard(hostId, selectedOrgId as number, deleteDialog.db.uid);
      enqueueSnackbar('Dashboard 已删除', { variant: 'success' });
      setDeleteDialog({ open: false });
      fetchDashboards();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '删除失败'), { variant: 'error' });
    }
  };

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="Dashboard 管理"
        subtitle="管理各 Grafana 组织下的 Dashboard，支持导入、浏览和删除"
        actionLabel="导入 Dashboard"
        onAction={() => selectedOrgId !== '' && setImportOpen(true)}
      />

      <Alert severity="info" sx={{ mb: 2 }}>
        选择一个 Grafana 组织以管理其下 Dashboard。可通过 Grafana 管理页面创建或管理组织。
      </Alert>

      <Card sx={{ mb: 2, p: 2 }}>
        <FormControl size="small" sx={{ minWidth: 280 }}>
          <InputLabel>Grafana 组织</InputLabel>
          <Select
            value={selectedOrgId}
            label="Grafana 组织"
            onChange={(e) => setSelectedOrgId(e.target.value as number | '')}
          >
            <MenuItem value="">请选择组织</MenuItem>
            {orgs.map((org) => (
              <MenuItem key={org.id} value={org.id}>
                {org.name}
                <Chip size="small" label={org.id} variant="outlined" sx={{ ml: 1, height: 18, fontSize: '0.65rem' }} />
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        {selectedOrgId !== '' && (
          <Button
            size="small"
            variant="outlined"
            sx={{ ml: 2 }}
            onClick={() => window.open(`${grafanaUrl}/?orgId=${selectedOrgId}`, '_blank')}
          >
            打开 Grafana
          </Button>
        )}
      </Card>

      {selectedOrgId === '' ? (
        <Card>
          <CardContent>
            <EmptyState title="请选择组织" description="从上方下拉框选择一个 Grafana 组织以查看其 Dashboard" />
          </CardContent>
        </Card>
      ) : dashLoading ? (
        <LoadingScreen />
      ) : dashboards.length === 0 ? (
        <Card>
          <CardContent>
            <EmptyState title="暂无 Dashboard" description="点击右上角「导入 Dashboard」按钮导入 JSON" />
          </CardContent>
        </Card>
      ) : (
        <Grid container spacing={2}>
          {dashboards.map((db) => (
            <Grid key={db.uid} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="subtitle1" sx={{ fontWeight: 500, mb: 0.5 }}>
                        <DashboardOutlinedIcon sx={{ fontSize: 18, mr: 0.5, verticalAlign: 'text-bottom', color: 'text.secondary' }} />
                        {db.title}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                        UID: {db.uid}
                      </Typography>
                      <Box sx={{ mt: 1 }}>
                        {db.tags?.map((tag) => (
                          <Chip key={tag} label={tag} size="small" variant="outlined" sx={{ mr: 0.5, mb: 0.5, height: 20, fontSize: '0.65rem' }} />
                        ))}
                        {db.folder_title && (
                          <Chip label={db.folder_title} size="small" color="primary" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                        )}
                      </Box>
                    </Box>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, ml: 1 }}>
                      <Tooltip title="打开">
                        <IconButton size="small" onClick={() => window.open(`${grafanaUrl}${db.url}`, '_blank')}>
                          <OpenInNewIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="删除">
                        <IconButton size="small" color="error" onClick={() => setDeleteDialog({ open: true, db })}>
                          <DeleteOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Import Dashboard Dialog */}
      <Dialog open={importOpen} onClose={() => setImportOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>导入 Dashboard 到组织</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            粘贴 Grafana Dashboard JSON（可从 Grafana 导出或使用预置模板）
          </Typography>
          <TextField
            fullWidth
            multiline
            minRows={10}
            maxRows={20}
            label="Dashboard JSON"
            value={importJson}
            onChange={(e) => setImportJson(e.target.value)}
            placeholder='{"title": "My Dashboard", "panels": [...], ...}'
            sx={{ '& .MuiInputBase-root': { fontFamily: 'monospace', fontSize: '0.8125rem' } }}
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setImportOpen(false)}>取消</Button>
          <Button variant="contained" startIcon={<UploadOutlinedIcon />} onClick={handleImport} disabled={saving || !importJson.trim()}>
            {saving ? '导入中...' : '导入'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Dashboard Confirmation */}
      <ConfirmDialog
        open={deleteDialog.open}
        title="删除 Dashboard"
        message={`确定要删除 Dashboard「${deleteDialog.db?.title}」吗？此操作不可恢复。`}
        severity="error"
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setDeleteDialog({ open: false })}
      />
    </Box>
  );
}
