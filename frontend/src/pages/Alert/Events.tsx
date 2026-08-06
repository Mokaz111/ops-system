import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Card,
  Chip,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import FilterToolbar from '../../components/common/FilterToolbar';
import { alertAPI } from '../../api/alert';
import { extractApiError } from '../../api';
import { useWorkspaceStore } from '../../stores/useWorkspaceStore';
import type { AlertEvent } from '../../types/api';
import { eventStatusMeta, levelMeta, useWorkspaceOptions, WorkspaceFilterSelect } from './shared';

export default function AlertEventsPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { workspaces } = useWorkspaceOptions();
  const [events, setEvents] = useState<AlertEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const globalWorkspaceId = useWorkspaceStore((s) => s.currentId);
  const [tenantFilter, setTenantFilter] = useState(globalWorkspaceId);
  useEffect(() => { setTenantFilter(globalWorkspaceId); setPage(0); }, [globalWorkspaceId]);
  const [statusFilter, setStatusFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState('');

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await alertAPI.listEvents({
        page: page + 1,
        page_size: pageSize,
        workspace_id: tenantFilter || undefined,
        status: statusFilter || undefined,
        level: levelFilter || undefined,
      });
      setEvents(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取告警事件失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [enqueueSnackbar, levelFilter, page, pageSize, statusFilter, tenantFilter]);

  useEffect(() => { fetchEvents(); }, [fetchEvents]);

  const handleAck = async (event: AlertEvent) => {
    try {
      await alertAPI.ackEvent(event.id);
      enqueueSnackbar('告警已确认', { variant: 'success' });
      fetchEvents();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '确认失败'), { variant: 'error' });
    }
  };

  if (loading && events.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="告警事件" subtitle="vmalert 触发、经 Alertmanager 分发的事件流" />

      <FilterToolbar>
        <WorkspaceFilterSelect
          value={tenantFilter}
          onChange={(v) => { setTenantFilter(v); setPage(0); }}
          workspaces={workspaces}
        />
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>状态</InputLabel>
          <Select value={statusFilter} label="状态" onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}>
            <MenuItem value="">全部状态</MenuItem>
            <MenuItem value="firing">告警中</MenuItem>
            <MenuItem value="acknowledged">已确认</MenuItem>
            <MenuItem value="resolved">已恢复</MenuItem>
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>级别</InputLabel>
          <Select value={levelFilter} label="级别" onChange={(e) => { setLevelFilter(e.target.value); setPage(0); }}>
            <MenuItem value="">全部级别</MenuItem>
            <MenuItem value="critical">严重</MenuItem>
            <MenuItem value="warning">警告</MenuItem>
            <MenuItem value="info">信息</MenuItem>
          </Select>
        </FormControl>
      </FilterToolbar>

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>规则</TableCell>
                <TableCell>级别</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>开始时间</TableCell>
                <TableCell>确认</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {events.length === 0 ? (
                <TableRow><TableCell colSpan={6}><EmptyState title="暂无事件" description="vmalert 触发后的事件会在这里展示" /></TableCell></TableRow>
              ) : events.map((event) => (
                <TableRow key={event.id}>
                  <TableCell sx={{ fontWeight: 500 }}>{event.rule_name}</TableCell>
                  <TableCell>
                    <Chip size="small" label={levelMeta[event.level]?.label || event.level} color={levelMeta[event.level]?.color || 'default'} />
                  </TableCell>
                  <TableCell>
                    <Chip size="small" label={eventStatusMeta[event.status]?.label || event.status} color={eventStatusMeta[event.status]?.color || 'default'} variant="outlined" />
                  </TableCell>
                  <TableCell>{new Date(event.start_time).toLocaleString()}</TableCell>
                  <TableCell>
                    {event.acked_at ? (
                      <Typography variant="caption" color="text.secondary">{new Date(event.acked_at).toLocaleString()}</Typography>
                    ) : '-'}
                  </TableCell>
                  <TableCell align="right">
                    {!event.acked_at && event.status === 'firing' && (
                      <Tooltip title="确认告警">
                        <IconButton size="small" color="primary" onClick={() => handleAck(event)}>
                          <CheckCircleOutlineIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        {total > 0 && (
          <TablePagination
            component="div"
            count={total}
            page={page}
            rowsPerPage={pageSize}
            onPageChange={(_, next) => setPage(next)}
            onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }}
            rowsPerPageOptions={[10, 20, 50]}
            labelRowsPerPage="每页行数"
          />
        )}
      </Card>
    </Box>
  );
}
