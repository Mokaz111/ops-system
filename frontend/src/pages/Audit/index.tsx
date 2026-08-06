import { useCallback, useEffect, useState } from 'react';
import {
  Box,
  Card,
  Chip,
  FormControl,
  InputAdornment,
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
  TextField,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import { auditAPI } from '../../api/audit';
import { extractApiError } from '../../api';
import type { AuditLog } from '../../types/api';

const statusColors: Record<string, 'success' | 'error' | 'default'> = {
  success: 'success',
  failed: 'error',
};

export default function AuditPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [items, setItems] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');

  const fetchAudits = useCallback(async () => {
    setLoading(true);
    try {
      const { data: res } = await auditAPI.list({
        page: page + 1,
        page_size: pageSize,
        search: search || undefined,
        status: statusFilter || undefined,
        action: actionFilter || undefined,
      });
      setItems(res.data?.items || []);
      setTotal(res.data?.total || 0);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取审计日志失败'), { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [actionFilter, enqueueSnackbar, page, pageSize, search, statusFilter]);

  useEffect(() => {
    fetchAudits();
  }, [fetchAudits]);

  if (loading && items.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="审计日志" subtitle="平台操作审计记录，仅管理员可见" />

      <Card sx={{ mb: 2, p: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
        <TextField
          placeholder="搜索操作、资源..."
          size="small"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(0); }}
          InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon sx={{ color: 'text.disabled' }} /></InputAdornment> }}
          sx={{ width: 280 }}
        />
        <FormControl size="small" sx={{ minWidth: 140 }}>
          <InputLabel>状态</InputLabel>
          <Select value={statusFilter} label="状态" onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}>
            <MenuItem value="">全部</MenuItem>
            <MenuItem value="success">成功</MenuItem>
            <MenuItem value="failed">失败</MenuItem>
          </Select>
        </FormControl>
        <TextField
          size="small"
          label="操作类型"
          value={actionFilter}
          onChange={(e) => { setActionFilter(e.target.value); setPage(0); }}
          sx={{ minWidth: 180 }}
        />
      </Card>

      <Card>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>时间</TableCell>
                <TableCell>操作</TableCell>
                <TableCell>资源</TableCell>
                <TableCell>资源 ID</TableCell>
                <TableCell>操作者</TableCell>
                <TableCell>IP</TableCell>
                <TableCell>状态</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <EmptyState title="暂无审计记录" />
                  </TableCell>
                </TableRow>
              ) : items.map((log) => (
                <TableRow key={log.id}>
                  <TableCell sx={{ fontSize: '0.8125rem', color: 'text.secondary', whiteSpace: 'nowrap' }}>
                    {new Date(log.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell sx={{ fontWeight: 500 }}>{log.action}</TableCell>
                  <TableCell>{log.resource}</TableCell>
                  <TableCell>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                      {log.resource_id || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                      {log.actor_id ? log.actor_id.slice(0, 8) : log.actor_type || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell sx={{ fontSize: '0.8125rem' }}>{log.ip || '-'}</TableCell>
                  <TableCell>
                    <Chip size="small" label={log.status} color={statusColors[log.status] || 'default'} variant="outlined" />
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
            onPageChange={(_, p) => setPage(p)}
            rowsPerPage={pageSize}
            onRowsPerPageChange={(e) => { setPageSize(parseInt(e.target.value, 10)); setPage(0); }}
            rowsPerPageOptions={[10, 20, 50, 100]}
            labelRowsPerPage="每页行数"
          />
        )}
      </Card>
    </Box>
  );
}
