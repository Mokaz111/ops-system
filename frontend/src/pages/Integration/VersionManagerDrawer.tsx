import { useState, useEffect, useMemo } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Drawer,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
import { useSnackbar } from 'notistack';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import {
  integrationAPI,
  type IntegrationTemplate,
  type IntegrationTemplateVersion,
} from '../../api/integration';
import { extractApiError } from '../../api';

interface Props {
  open: boolean;
  template: IntegrationTemplate | null;
  onClose: () => void;
  onSuccess: () => void;
}

type DiffLine = { kind: 'eq' | 'add' | 'del'; text: string };

// 极简行 diff：仅用于展示（不追求最优对齐）。
function simpleDiff(left: string, right: string): DiffLine[] {
  const a = (left || '').split('\n');
  const b = (right || '').split('\n');
  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      out.push({ kind: 'eq', text: a[i] });
      i++;
      j++;
      continue;
    }
    const aNext = b.indexOf(a[i], j);
    const bNext = a.indexOf(b[j], i);
    if (aNext !== -1 && (bNext === -1 || aNext - j <= bNext - i)) {
      while (j < aNext) {
        out.push({ kind: 'add', text: b[j] });
        j++;
      }
    } else if (bNext !== -1) {
      while (i < bNext) {
        out.push({ kind: 'del', text: a[i] });
        i++;
      }
    } else {
      out.push({ kind: 'del', text: a[i] });
      out.push({ kind: 'add', text: b[j] });
      i++;
      j++;
    }
  }
  while (i < a.length) out.push({ kind: 'del', text: a[i++] });
  while (j < b.length) out.push({ kind: 'add', text: b[j++] });
  return out;
}

function DiffBlock({ title, left, right }: { title: string; left: string; right: string }) {
  const lines = useMemo(() => simpleDiff(left || '', right || ''), [left, right]);
  const hasChange = lines.some((l) => l.kind !== 'eq');
  return (
    <Paper variant="outlined" sx={{ mb: 2 }}>
      <Box sx={{ px: 2, py: 1, borderBottom: '1px solid', borderColor: 'divider', display: 'flex', justifyContent: 'space-between' }}>
        <Typography variant="subtitle2">{title}</Typography>
        <Chip size="small" label={hasChange ? '有变更' : '无变更'} color={hasChange ? 'warning' : 'default'} />
      </Box>
      <Box
        sx={{
          maxHeight: 240,
          overflow: 'auto',
          fontFamily: 'monospace',
          fontSize: 12,
          whiteSpace: 'pre',
          p: 1,
        }}
      >
        {lines.length === 0 ? (
          <Typography variant="caption" color="text.secondary">（两侧均为空）</Typography>
        ) : (
          lines.map((l, idx) => {
            const color =
              l.kind === 'add' ? 'rgba(76,175,80,0.14)' : l.kind === 'del' ? 'rgba(244,67,54,0.14)' : 'transparent';
            const marker = l.kind === 'add' ? '+' : l.kind === 'del' ? '-' : ' ';
            return (
              <Box key={idx} sx={{ backgroundColor: color, px: 0.5 }}>
                <span style={{ opacity: 0.5 }}>{marker} </span>
                {l.text || '\u200B'}
              </Box>
            );
          })
        )}
      </Box>
    </Paper>
  );
}

export default function VersionManagerDrawer({ open, template, onClose, onSuccess }: Props) {
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(false);
  const [versions, setVersions] = useState<IntegrationTemplateVersion[]>([]);
  const [leftVer, setLeftVer] = useState('');
  const [rightVer, setRightVer] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<IntegrationTemplateVersion | null>(null);
  const [reloadTick, setReloadTick] = useState(0);

  useEffect(() => {
    if (!open || !template) return;
    let alive = true;
    (async () => {
      setLoading(true);
      try {
        const { data: res } = await integrationAPI.listVersions(template.id);
        if (!alive) return;
        const list = res.data || [];
        setVersions(list);
        if (list.length >= 2) {
          setLeftVer(list[1].version);
          setRightVer(list[0].version);
        } else if (list.length === 1) {
          setLeftVer(list[0].version);
          setRightVer(list[0].version);
        } else {
          setLeftVer('');
          setRightVer('');
        }
      } catch (err) {
        if (alive) {
          setVersions([]);
          enqueueSnackbar(extractApiError(err, '加载版本失败'), { variant: 'error' });
        }
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [open, template, reloadTick, enqueueSnackbar]);

  const left = useMemo(() => versions.find((v) => v.version === leftVer) || null, [versions, leftVer]);
  const right = useMemo(() => versions.find((v) => v.version === rightVer) || null, [versions, rightVer]);

  const handleSwap = () => {
    setLeftVer(rightVer);
    setRightVer(leftVer);
  };

  const confirmDelete = async () => {
    if (!template || !deleteTarget) return;
    try {
      await integrationAPI.deleteVersion(template.id, deleteTarget.version);
      enqueueSnackbar(`版本 ${deleteTarget.version} 已下架`, { variant: 'success' });
      setDeleteTarget(null);
      setReloadTick((v) => v + 1);
      onSuccess();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '下架失败'), { variant: 'error' });
    }
  };

  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: 880 } } }}>
      <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
        <Stack direction="row" alignItems="center" sx={{ mb: 2 }}>
          <Typography variant="h6" sx={{ flex: 1 }}>
            版本管理 · {template?.display_name || template?.name}
          </Typography>
          <IconButton onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </Stack>

        <Alert severity="info" sx={{ mb: 2 }}>
          当前 latest_version：<b>{template?.latest_version || '-'}</b>。下架会从数据库删除该版本；
          若仍被活跃安装记录引用 / 或是唯一版本，后端会拒绝。删除 latest 时会自动切换到剩余最新创建的版本。
        </Alert>

        {loading ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CircularProgress size={20} /> 正在加载...
          </Box>
        ) : versions.length === 0 ? (
          <Alert severity="warning">该模板尚无任何版本。</Alert>
        ) : (
          <>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>版本列表</Typography>
            <Table size="small" sx={{ mb: 3 }}>
              <TableHead>
                <TableRow>
                  <TableCell>版本</TableCell>
                  <TableCell>创建时间</TableCell>
                  <TableCell>变更说明</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {versions.map((v) => {
                  const isLatest = v.version === template?.latest_version;
                  return (
                    <TableRow key={v.id} hover>
                      <TableCell>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                            {v.version}
                          </Typography>
                          {isLatest && <Chip size="small" color="primary" label="latest" />}
                        </Stack>
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption" color="text.secondary">
                          {new Date(v.created_at).toLocaleString()}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption">{v.changelog || '—'}</Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="作为左侧（基线）对比">
                          <Button size="small" onClick={() => setLeftVer(v.version)} disabled={leftVer === v.version}>
                            设为左
                          </Button>
                        </Tooltip>
                        <Tooltip title="作为右侧（新版）对比">
                          <Button size="small" onClick={() => setRightVer(v.version)} disabled={rightVer === v.version}>
                            设为右
                          </Button>
                        </Tooltip>
                        <Tooltip title="下架此版本">
                          <IconButton size="small" color="error" onClick={() => setDeleteTarget(v)}>
                            <DeleteOutlinedIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>

            <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
              <FormControl size="small" sx={{ minWidth: 160 }}>
                <InputLabel>左（基线）</InputLabel>
                <Select label="左（基线）" value={leftVer} onChange={(e) => setLeftVer(e.target.value)}>
                  {versions.map((v) => (
                    <MenuItem key={v.id} value={v.version}>
                      {v.version}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <IconButton onClick={handleSwap} size="small">
                <CompareArrowsIcon />
              </IconButton>
              <FormControl size="small" sx={{ minWidth: 160 }}>
                <InputLabel>右（新版）</InputLabel>
                <Select label="右（新版）" value={rightVer} onChange={(e) => setRightVer(e.target.value)}>
                  {versions.map((v) => (
                    <MenuItem key={v.id} value={v.version}>
                      {v.version}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <Typography variant="caption" color="text.secondary">
                选取两个版本即可查看各 spec 字段的 diff。
              </Typography>
            </Stack>

            {left && right && (
              <>
                <DiffBlock title="collector_spec" left={left.collector_spec} right={right.collector_spec} />
                <DiffBlock title="alert_spec" left={left.alert_spec} right={right.alert_spec} />
                <DiffBlock title="dashboard_spec" left={left.dashboard_spec} right={right.dashboard_spec} />
                <DiffBlock title="variables" left={left.variables} right={right.variables} />
              </>
            )}
          </>
        )}

        <ConfirmDialog
          open={!!deleteTarget}
          title="下架模板版本"
          message={`确定要下架版本「${deleteTarget?.version}」吗？仍被活跃安装引用或该版本为唯一版本时会被服务端拒绝。`}
          severity="error"
          confirmLabel="下架"
          onConfirm={confirmDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      </Box>
    </Drawer>
  );
}
