import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Drawer,
  Grid,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import SyncIcon from '@mui/icons-material/Sync';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { metricAPI, type Metric, type MetricTemplateMapping } from '../../api/metric';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { useSnackbar } from 'notistack';

interface PanelRef {
  dashboard_uid: string;
  panel_id: number;
  title: string;
  expr: string;
}

function parsePanels(raw: string): PanelRef[] {
  if (!raw) return [];
  try {
    const v = JSON.parse(raw);
    if (Array.isArray(v)) return v as PanelRef[];
  } catch {
    /* ignore */
  }
  return [];
}

export default function MetricPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [items, setItems] = useState<Metric[]>([]);
  const [loading, setLoading] = useState(true);
  const [component, setComponent] = useState('');
  const [keyword, setKeyword] = useState('');
  const [reparseTemplateId, setReparseTemplateId] = useState('');
  const [reparsing, setReparsing] = useState(false);

  const [selected, setSelected] = useState<Metric | null>(null);
  const [related, setRelated] = useState<MetricTemplateMapping[]>([]);
  const [editOpen, setEditOpen] = useState(false);
  const [editForm, setEditForm] = useState({ description_cn: '', description_en: '', unit: '', tags: '' });
  const [saving, setSaving] = useState(false);

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await metricAPI.list({ page: 1, page_size: 100, component, keyword });
      setItems(res.data?.items || []);
    } finally {
      setLoading(false);
    }
  }, [component, keyword]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const openDetail = useCallback(async (m: Metric) => {
    setSelected(m);
    setRelated([]);
    try {
      const { data: res } = await metricAPI.related(m.id);
      setRelated(res.data || []);
    } catch {
      /* ignore */
    }
  }, []);

  const openEdit = (m: Metric) => {
    setSelected(m);
    setEditForm({
      description_cn: m.description_cn || '',
      description_en: m.description_en || '',
      unit: m.unit || '',
      tags: m.tags || '',
    });
    setEditOpen(true);
  };

  const handleSaveEdit = async () => {
    if (!selected) return;
    setSaving(true);
    try {
      await metricAPI.update(selected.id, editForm);
      enqueueSnackbar('指标已更新', { variant: 'success' });
      setEditOpen(false);
      fetchList();
      const updated = { ...selected, ...editForm };
      setSelected(updated);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '更新失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleReparse = async () => {
    if (!reparseTemplateId.trim()) {
      enqueueSnackbar('请输入模版 ID', { variant: 'warning' });
      return;
    }
    setReparsing(true);
    try {
      await metricAPI.reparse(reparseTemplateId.trim());
      enqueueSnackbar('指标重解析已触发', { variant: 'success' });
      fetchList();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '重解析失败'), { variant: 'error' });
    } finally {
      setReparsing(false);
    }
  };

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="指标库" subtitle="统一管理指标含义、单位、标签与来源模版" />

      <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap' }}>
        <TextField
          size="small"
          placeholder="按组件筛选（node / mysql / redis ...）"
          value={component}
          onChange={(e) => setComponent(e.target.value)}
          sx={{ minWidth: 260 }}
        />
        <TextField
          size="small"
          placeholder="搜索名称或描述"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          sx={{ minWidth: 260 }}
        />
        {isAdmin && (
          <>
            <TextField
              size="small"
              placeholder="模版 ID（重解析）"
              value={reparseTemplateId}
              onChange={(e) => setReparseTemplateId(e.target.value)}
              sx={{ minWidth: 280 }}
            />
            <Button startIcon={<SyncIcon />} variant="outlined" onClick={handleReparse} disabled={reparsing}>
              {reparsing ? '解析中...' : '重解析模版指标'}
            </Button>
          </>
        )}
      </Stack>

      {items.length === 0 ? (
        <EmptyState
          title="指标库为空"
          description="启动后 Seeder 会自动填充 node/mysql/redis 的指标。若为空，请检查后端日志。"
        />
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>指标</TableCell>
                <TableCell>组件</TableCell>
                <TableCell>类型</TableCell>
                <TableCell>单位</TableCell>
                <TableCell>描述</TableCell>
                <TableCell>来源</TableCell>
                {isAdmin && <TableCell align="right">操作</TableCell>}
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((m) => (
                <TableRow key={m.id} hover sx={{ cursor: 'pointer' }} onClick={() => openDetail(m)}>
                  <TableCell sx={{ fontFamily: 'monospace' }}>{m.name}</TableCell>
                  <TableCell>{m.component || '-'}</TableCell>
                  <TableCell>
                    <Chip size="small" label={m.metric_type || '-'} />
                  </TableCell>
                  <TableCell>{m.unit || '-'}</TableCell>
                  <TableCell>
                    <Typography variant="caption" color="text.secondary">
                      {m.description_cn || m.description_en || '—'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {m.manual_override ? (
                      <Chip size="small" color="warning" label="手工" />
                    ) : (
                      <Chip size="small" color="primary" label={m.source_template_version || '模版'} />
                    )}
                  </TableCell>
                  {isAdmin && (
                    <TableCell align="right" onClick={(e) => e.stopPropagation()}>
                      <IconButton size="small" onClick={() => openEdit(m)}>
                        <EditOutlinedIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Drawer
        anchor="right"
        open={!!selected && !editOpen}
        onClose={() => setSelected(null)}
        PaperProps={{ sx: { width: { xs: '100%', md: 640 } } }}
      >
        {selected && (
          <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <Box sx={{ flex: 1 }}>
                <Typography variant="h6" sx={{ fontFamily: 'monospace' }}>
                  {selected.name}
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 0.5 }}>
                  <Chip size="small" label={selected.metric_type || '-'} />
                  {selected.unit && <Chip size="small" label={selected.unit} variant="outlined" />}
                  {selected.component && <Chip size="small" label={selected.component} color="primary" />}
                </Stack>
              </Box>
              {isAdmin && (
                <IconButton onClick={() => openEdit(selected)} sx={{ mr: 1 }}>
                  <EditOutlinedIcon />
                </IconButton>
              )}
              <IconButton onClick={() => setSelected(null)}>
                <CloseIcon />
              </IconButton>
            </Box>

            <Grid container spacing={2} sx={{ mb: 2 }}>
              <Grid size={{ xs: 12 }}>
                <Typography variant="caption" color="text.secondary">中文描述</Typography>
                <Typography variant="body2">{selected.description_cn || '—'}</Typography>
              </Grid>
              <Grid size={{ xs: 12 }}>
                <Typography variant="caption" color="text.secondary">英文描述</Typography>
                <Typography variant="body2">{selected.description_en || '—'}</Typography>
              </Grid>
              {selected.source_template_id && (
                <Grid size={{ xs: 12 }}>
                  <Typography variant="caption" color="text.secondary">来源模版</Typography>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                    {selected.source_template_id.slice(0, 8)} · {selected.source_template_version}
                  </Typography>
                </Grid>
              )}
            </Grid>

            <Divider sx={{ my: 2 }} />

            <Typography variant="subtitle2" sx={{ mb: 1 }}>关联模版与面板</Typography>
            {related.length === 0 ? (
              <Alert severity="info">暂无关联记录。</Alert>
            ) : (
              <Stack spacing={1.5}>
                {related.map((r) => {
                  const panels = parsePanels(r.dashboard_panels);
                  return (
                    <Paper key={r.id} variant="outlined" sx={{ p: 1.5 }}>
                      <Stack direction="row" spacing={1} sx={{ mb: 1, alignItems: 'center' }}>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                          {r.template_id.slice(0, 8)} · {r.template_version}
                        </Typography>
                        {r.appears_in_collector && <Chip size="small" label="采集" color="primary" />}
                        {r.appears_in_alert && <Chip size="small" label="告警" color="warning" />}
                        {r.appears_in_dashboard && <Chip size="small" label="大盘" color="success" />}
                      </Stack>
                      {panels.length > 0 && (
                        <Stack spacing={0.5}>
                          {panels.map((p, idx) => (
                            <Box key={idx}>
                              <Typography variant="caption" color="text.secondary">
                                {p.dashboard_uid} · #{p.panel_id} {p.title}
                              </Typography>
                              <Typography
                                variant="caption"
                                sx={{
                                  display: 'block',
                                  fontFamily: 'monospace',
                                  bgcolor: 'background.default',
                                  p: 0.5,
                                  borderRadius: 0.5,
                                  wordBreak: 'break-all',
                                }}
                              >
                                {p.expr}
                              </Typography>
                            </Box>
                          ))}
                        </Stack>
                      )}
                    </Paper>
                  );
                })}
              </Stack>
            )}
          </Box>
        )}
      </Drawer>

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>编辑指标 — {selected?.name}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <TextField fullWidth label="中文描述" value={editForm.description_cn} onChange={(e) => setEditForm({ ...editForm, description_cn: e.target.value })} sx={{ mb: 2 }} multiline minRows={2} />
          <TextField fullWidth label="英文描述" value={editForm.description_en} onChange={(e) => setEditForm({ ...editForm, description_en: e.target.value })} sx={{ mb: 2 }} multiline minRows={2} />
          <TextField fullWidth label="单位" value={editForm.unit} onChange={(e) => setEditForm({ ...editForm, unit: e.target.value })} sx={{ mb: 2 }} />
          <TextField fullWidth label="标签 (JSON)" value={editForm.tags} onChange={(e) => setEditForm({ ...editForm, tags: e.target.value })} multiline minRows={2} />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setEditOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleSaveEdit} disabled={saving}>
            {saving ? '保存中...' : '保存'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
