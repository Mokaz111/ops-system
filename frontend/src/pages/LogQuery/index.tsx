import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Collapse,
  FormControlLabel,
  IconButton,
  MenuItem,
  Select,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import DownloadIcon from '@mui/icons-material/Download';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { useSearchParams } from 'react-router-dom';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip as ChartTooltip, XAxis, YAxis } from 'recharts';
import PageHeader from '../../components/common/PageHeader';
import { logAPI, type LogEntry, type LogInstance } from '../../api/logs';
import { extractApiError } from '../../api';

const timeRanges = [
  { label: '近 15 分钟', minutes: 15 },
  { label: '近 1 小时', minutes: 60 },
  { label: '近 6 小时', minutes: 360 },
  { label: '近 24 小时', minutes: 1440 },
  { label: '近 3 天', minutes: 4320 },
];

// 级别快捷过滤：点击追加/移除对应 LogsQL 片段。
const levelFilters = [
  { key: 'error', label: 'ERROR', color: '#d32f2f', logsql: 'level:~"(?i)^(error|fatal|critical)$"' },
  { key: 'warn', label: 'WARN', color: '#ed6c02', logsql: 'level:~"(?i)^warn(ing)?$"' },
  { key: 'info', label: 'INFO', color: '#0288d1', logsql: 'level:~"(?i)^info$"' },
  { key: 'signal', label: '关键信号', color: '#7b1fa2', logsql: 'ops_signal:~"^(error|critical)$"' },
] as const;

const OPS_FIELD_PREFIX = 'ops_';
const HIDDEN_FIELDS = new Set(['_stream', '_stream_id']);

function levelOf(entry: LogEntry): string {
  const raw = (entry.fields?.level || entry.fields?.severity || entry.fields?.ops_signal || '').toLowerCase();
  if (['error', 'fatal', 'critical'].includes(raw)) return 'error';
  if (raw.startsWith('warn')) return 'warn';
  if (raw === 'debug' || raw === 'trace') return 'debug';
  if (raw === 'info') return 'info';
  return '';
}

const levelColors: Record<string, string> = {
  error: '#d32f2f',
  warn: '#ed6c02',
  info: '#0288d1',
  debug: '#9e9e9e',
};

function LogRow({ entry, onFieldFilter }: { entry: LogEntry; onFieldFilter: (k: string, v: string) => void }) {
  const [open, setOpen] = useState(false);
  const level = levelOf(entry);
  const allFields = Object.entries(entry.fields || {}).filter(([k]) => !HIDDEN_FIELDS.has(k));
  const bizFields = allFields.filter(([k]) => !k.startsWith(OPS_FIELD_PREFIX));
  const opsFields = allFields.filter(([k]) => k.startsWith(OPS_FIELD_PREFIX));

  return (
    <>
      <TableRow
        hover
        sx={{ cursor: 'pointer', '& td': { borderLeft: level ? `3px solid ${levelColors[level]}` : '3px solid transparent' } }}
        onClick={() => setOpen(!open)}
      >
        <TableCell sx={{ whiteSpace: 'nowrap', fontSize: 12, color: 'text.secondary', width: 170 }}>
          <Tooltip title={entry.time || ''}>
            <span>{entry.time ? new Date(entry.time).toLocaleString() : '-'}</span>
          </Tooltip>
        </TableCell>
        <TableCell sx={{ width: 70 }}>
          {level && (
            <Typography variant="caption" sx={{ color: levelColors[level], fontWeight: 700 }}>
              {level.toUpperCase()}
            </Typography>
          )}
        </TableCell>
        <TableCell
          sx={{
            fontFamily: 'monospace',
            fontSize: 12,
            maxWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: open ? 'pre-wrap' : 'nowrap',
            wordBreak: 'break-all',
          }}
        >
          {entry.message || '-'}
        </TableCell>
        <TableCell sx={{ width: 40, p: 0.5 }}>
          {open ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </TableCell>
      </TableRow>
      <TableRow>
        <TableCell colSpan={4} sx={{ py: 0, border: 0 }}>
          <Collapse in={open} unmountOnExit>
            <Box sx={{ p: 1.5, bgcolor: 'background.default', borderRadius: 1, mb: 1 }}>
              {bizFields.length > 0 && (
                <Box sx={{ mb: opsFields.length ? 1 : 0 }}>
                  {bizFields.map(([k, v]) => (
                    <Tooltip key={k} title="点击追加为过滤条件">
                      <Chip
                        size="small"
                        icon={<FilterAltIcon sx={{ fontSize: 14 }} />}
                        label={`${k}: ${v.length > 80 ? v.slice(0, 80) + '…' : v}`}
                        variant="outlined"
                        onClick={(e) => {
                          e.stopPropagation();
                          onFieldFilter(k, v);
                        }}
                        sx={{ mr: 0.5, mb: 0.5, maxWidth: '100%' }}
                      />
                    </Tooltip>
                  ))}
                </Box>
              )}
              {opsFields.length > 0 && (
                <Box>
                  {opsFields.map(([k, v]) => (
                    <Chip
                      key={k}
                      size="small"
                      label={`${k}: ${v}`}
                      sx={{ mr: 0.5, mb: 0.5, bgcolor: 'action.hover', fontSize: 11 }}
                    />
                  ))}
                </Box>
              )}
              {allFields.length === 0 && (
                <Typography variant="caption" color="text.secondary">
                  无附加字段
                </Typography>
              )}
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
}

export default function LogQueryPage() {
  const [searchParams] = useSearchParams();
  const presetInstanceId = searchParams.get('instance_id');
  const [instances, setInstances] = useState<LogInstance[]>([]);
  const [instanceId, setInstanceId] = useState('');
  const [query, setQuery] = useState('');
  const [activeLevels, setActiveLevels] = useState<string[]>([]);
  const [rangeMinutes, setRangeMinutes] = useState(15);
  const [limit, setLimit] = useState(100);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [stats, setStats] = useState<{ returned: number; limit: number } | null>(null);
  const [elapsedMs, setElapsedMs] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [running, setRunning] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [queried, setQueried] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { data: res } = await logAPI.list({ page: 1, page_size: 100 });
        if (!alive) return;
        const items = res.data?.items || [];
        setInstances(items);
        // 支持从「日志实例」页带 instance_id 跳转直达。
        const preset = presetInstanceId && items.some((i) => i.id === presetInstanceId) ? presetInstanceId : '';
        if (preset) setInstanceId(preset);
        else if (items.length > 0) setInstanceId(items[0].id);
      } catch {
        /* 列表加载失败时保持空态 */
      }
    })();
    return () => {
      alive = false;
    };
  }, [presetInstanceId]);

  // 组合最终 LogsQL：级别快捷过滤 AND 用户输入。
  const effectiveQuery = useMemo(() => {
    const parts: string[] = [];
    const levels = levelFilters.filter((f) => activeLevels.includes(f.key)).map((f) => f.logsql);
    if (levels.length === 1) parts.push(levels[0]);
    if (levels.length > 1) parts.push('(' + levels.join(' OR ') + ')');
    const raw = query.trim();
    if (raw && raw !== '*') parts.push(raw);
    return parts.join(' AND ') || '*';
  }, [query, activeLevels]);

  const run = useCallback(async () => {
    if (!instanceId) return;
    setRunning(true);
    setError('');
    const started = performance.now();
    try {
      const end = new Date();
      const start = new Date(end.getTime() - rangeMinutes * 60 * 1000);
      const { data: res } = await logAPI.query(instanceId, {
        query: effectiveQuery,
        start: start.toISOString(),
        end: end.toISOString(),
        limit,
      });
      setEntries(res.data?.entries || []);
      setStats(res.data?.stats || null);
      setElapsedMs(Math.round(performance.now() - started));
      setQueried(true);
    } catch (e) {
      setError(extractApiError(e, '查询失败'));
      setEntries([]);
      setStats(null);
      setElapsedMs(null);
    } finally {
      setRunning(false);
    }
  }, [instanceId, effectiveQuery, rangeMinutes, limit]);

  // 自动刷新：10s 一次；切实例/关开关时重置。
  useEffect(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    if (autoRefresh && instanceId) {
      timerRef.current = setInterval(run, 10_000);
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [autoRefresh, instanceId, run]);

  const toggleLevel = (key: string) => {
    setActiveLevels((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  const appendFieldFilter = (k: string, v: string) => {
    const fragment = `${k}:"${v.replaceAll('\\', '\\\\').replaceAll('"', '\\"')}"`;
    setQuery((prev) => {
      const trimmed = prev.trim();
      if (trimmed.includes(fragment)) return prev;
      return trimmed && trimmed !== '*' ? `${trimmed} AND ${fragment}` : fragment;
    });
  };

  // 日志量直方图：把已返回的日志按时间分桶、按级别堆叠。
  // 注意这是基于当前结果集（受 limit 截断）的分布，不是服务端全量统计。
  const histogram = useMemo(() => {
    if (entries.length === 0) return [];
    const times = entries.map((e) => new Date(e.time).getTime()).filter((t) => !Number.isNaN(t));
    if (times.length === 0) return [];
    const min = Math.min(...times);
    const max = Math.max(...times);
    const bucketCount = 30;
    const span = Math.max(max - min, 1);
    const bucketMs = span / bucketCount;
    const buckets = Array.from({ length: bucketCount }, (_, i) => ({
      time: new Date(min + i * bucketMs).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
      error: 0,
      warn: 0,
      info: 0,
      other: 0,
    }));
    for (const entry of entries) {
      const t = new Date(entry.time).getTime();
      if (Number.isNaN(t)) continue;
      const idx = Math.min(Math.floor((t - min) / bucketMs), bucketCount - 1);
      const level = levelOf(entry);
      if (level === 'error') buckets[idx].error += 1;
      else if (level === 'warn') buckets[idx].warn += 1;
      else if (level === 'info') buckets[idx].info += 1;
      else buckets[idx].other += 1;
    }
    return buckets;
  }, [entries]);

  const exportJSON = () => {
    const blob = new Blob([JSON.stringify(entries, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs-${new Date().toISOString().replace(/[:.]/g, '-')}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Box>
      <PageHeader title="日志查询" subtitle="LogsQL 查询控制台（租户过滤由后端强制附加，无需手写）" />

      <Card sx={{ mb: 2 }}>
        <CardContent>
          {/* 第一行：实例 / 时间 / limit / 执行 */}
          <Box sx={{ display: 'flex', gap: 1.5, mb: 1.5, alignItems: 'center', flexWrap: 'wrap' }}>
            <Select
              size="small"
              displayEmpty
              value={instanceId}
              onChange={(e) => setInstanceId(e.target.value)}
              sx={{ minWidth: 220 }}
            >
              <MenuItem value="">选择日志实例</MenuItem>
              {instances.map((i) => (
                <MenuItem key={i.id} value={i.id}>
                  {i.instance_name}
                </MenuItem>
              ))}
            </Select>
            <Select size="small" value={rangeMinutes} onChange={(e) => setRangeMinutes(Number(e.target.value))} sx={{ minWidth: 130 }}>
              {timeRanges.map((r) => (
                <MenuItem key={r.minutes} value={r.minutes}>
                  {r.label}
                </MenuItem>
              ))}
            </Select>
            <TextField
              size="small"
              type="number"
              label="Limit"
              value={limit}
              onChange={(e) => setLimit(Math.min(1000, Math.max(1, Number(e.target.value) || 100)))}
              sx={{ width: 90 }}
            />
            <FormControlLabel
              control={<Switch size="small" checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} />}
              label={<Typography variant="body2">自动刷新(10s)</Typography>}
              sx={{ ml: 0 }}
            />
            <Box sx={{ flex: 1 }} />
            <Button variant="contained" startIcon={<PlayArrowIcon />} onClick={run} disabled={!instanceId || running}>
              {running ? '查询中…' : '查询'}
            </Button>
          </Box>

          {/* 第二行：级别快捷过滤 + 查询输入 */}
          <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'center', flexWrap: 'wrap' }}>
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {levelFilters.map((f) => (
                <Chip
                  key={f.key}
                  size="small"
                  label={f.label}
                  clickable
                  onClick={() => toggleLevel(f.key)}
                  sx={{
                    fontWeight: 600,
                    ...(activeLevels.includes(f.key)
                      ? { bgcolor: f.color, color: '#fff', '&:hover': { bgcolor: f.color } }
                      : { color: f.color, borderColor: f.color }),
                  }}
                  variant={activeLevels.includes(f.key) ? 'filled' : 'outlined'}
                />
              ))}
            </Box>
            <TextField
              size="small"
              placeholder='LogsQL 过滤，如 _msg:~"timeout" 或 kubernetes.pod_name:"api-*"（Enter 执行）'
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') run();
              }}
              sx={{ flex: 1, minWidth: 260 }}
            />
          </Box>

          {(activeLevels.length > 0 || query.trim()) && (
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block', fontFamily: 'monospace' }}>
              生效查询：{effectiveQuery}
            </Typography>
          )}
        </CardContent>
      </Card>

      {histogram.length > 0 && (
        <Card sx={{ mb: 2 }}>
          <CardContent sx={{ pb: '12px !important' }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
              日志量分布（当前结果集，按级别堆叠）
            </Typography>
            <ResponsiveContainer width="100%" height={120}>
              <BarChart data={histogram} margin={{ top: 4, right: 8, bottom: 0, left: -24 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis allowDecimals={false} tick={{ fontSize: 10 }} />
                <ChartTooltip />
                <Bar dataKey="error" stackId="l" fill={levelColors.error} name="ERROR" />
                <Bar dataKey="warn" stackId="l" fill={levelColors.warn} name="WARN" />
                <Bar dataKey="info" stackId="l" fill={levelColors.info} name="INFO" />
                <Bar dataKey="other" stackId="l" fill={levelColors.debug} name="其他" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardContent>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {stats && (
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
              <Typography variant="caption" color="text.secondary">
                返回 {stats.returned} 条（limit={stats.limit}）
                {elapsedMs !== null ? ` · 耗时 ${elapsedMs}ms` : ''}
                {stats.returned >= stats.limit ? ' · 已达上限，可缩小时间范围或加过滤' : ''}
              </Typography>
              <Box sx={{ flex: 1 }} />
              {entries.length > 0 && (
                <Tooltip title="导出当前结果为 JSON">
                  <IconButton size="small" onClick={exportJSON}>
                    <DownloadIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              )}
            </Box>
          )}

          {entries.length > 0 ? (
            <TableContainer sx={{ maxHeight: 560 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ width: 170 }}>时间</TableCell>
                    <TableCell sx={{ width: 70 }}>级别</TableCell>
                    <TableCell>消息（点击行展开字段）</TableCell>
                    <TableCell sx={{ width: 40 }} />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {entries.map((e, idx) => (
                    <LogRow key={`${e.time}-${idx}`} entry={e} onFieldFilter={appendFieldFilter} />
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            !error && (
              <Alert severity="info">
                {queried
                  ? '该时间范围内无匹配日志，可扩大时间范围或调整过滤条件。'
                  : instances.length === 0
                    ? '暂无日志实例：请先在「日志实例」页创建，并确认业务集群已启用日志采集。'
                    : '选择日志实例后点击查询。级别 Chip 可快捷过滤，展开日志行可点字段追加过滤。'}
              </Alert>
            )
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
