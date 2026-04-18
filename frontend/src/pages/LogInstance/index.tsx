import { useEffect, useState } from 'react';
import {
  Box,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
} from '@mui/material';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import StatusChip from '../../components/common/StatusChip';
import { logAPI, type LogInstance } from '../../api/logs';

export default function LogInstancePage() {
  const [items, setItems] = useState<LogInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { data: res } = await logAPI.list({ page: 1, page_size: 50, keyword });
        if (!alive) return;
        setItems(res.data?.items || []);
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [keyword]);

  if (loading) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader title="日志实例" subtitle="基于 VictoriaLogs 的日志存储实例管理" />

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <TextField
          size="small"
          placeholder="搜索实例名称"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          sx={{ minWidth: 260 }}
        />
      </Stack>

      {items.length === 0 ? (
        <EmptyState
          title="暂无日志实例"
          description="M4 将开放创建入口（Helm vm/victoria-logs-single/cluster），此处仅做元数据占位。"
        />
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>命名空间</TableCell>
                <TableCell>Release</TableCell>
                <TableCell>保留天数</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((i) => (
                <TableRow key={i.id} hover>
                  <TableCell>{i.instance_name}</TableCell>
                  <TableCell>{i.namespace || '-'}</TableCell>
                  <TableCell>{i.release_name || '-'}</TableCell>
                  <TableCell>{i.retention_days || '-'}</TableCell>
                  <TableCell>
                    <StatusChip status={i.status} />
                  </TableCell>
                  <TableCell>{new Date(i.created_at).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
}
