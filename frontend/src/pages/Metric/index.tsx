import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Chip,
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
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { metricAPI, type Metric, type MetricTemplateMapping } from '../../api/metric';

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
  const [items, setItems] = useState<Metric[]>([]);
  const [loading, setLoading] = useState(true);
  const [component, setComponent] = useState('');
  const [keyword, setKeyword] = useState('');

  const [selected, setSelected] = useState<Metric | null>(null);
  const [related, setRelated] = useState<MetricTemplateMapping[]>([]);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { data: res } = await metricAPI.list({ page: 1, page_size: 100, component, keyword });
        if (!alive) return;
        setItems(res.data?.items || []);
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [component, keyword]);

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

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="指标库" subtitle="统一管理指标含义、单位、标签与来源模版" />

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
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
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Drawer
        anchor="right"
        open={!!selected}
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
    </Box>
  );
}
