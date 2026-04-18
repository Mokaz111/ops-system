import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  MenuItem,
  Select,
  TextField,
  Typography,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PageHeader from '../../components/common/PageHeader';
import { logAPI, type LogInstance } from '../../api/logs';

export default function LogQueryPage() {
  const [instances, setInstances] = useState<LogInstance[]>([]);
  const [instanceId, setInstanceId] = useState('');
  const [query, setQuery] = useState('*');
  const [result, setResult] = useState<string>('');
  const [running, setRunning] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      const { data: res } = await logAPI.list({ page: 1, page_size: 100 });
      if (!alive) return;
      const items = res.data?.items || [];
      setInstances(items);
      if (items.length > 0) setInstanceId(items[0].id);
    })();
    return () => {
      alive = false;
    };
  }, []);

  const run = async () => {
    if (!instanceId) return;
    setRunning(true);
    try {
      const { data: res } = await logAPI.query(instanceId, { query });
      setResult(JSON.stringify(res.data, null, 2));
    } catch (e) {
      setResult(String(e));
    } finally {
      setRunning(false);
    }
  };

  return (
    <Box>
      <PageHeader title="日志查询" subtitle="LogsQL 查询台（M4 接入真实 VictoriaLogs）" />

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
            <Select
              size="small"
              displayEmpty
              value={instanceId}
              onChange={(e) => setInstanceId(e.target.value)}
              sx={{ minWidth: 240 }}
            >
              <MenuItem value="">选择日志实例</MenuItem>
              {instances.map((i) => (
                <MenuItem key={i.id} value={i.id}>
                  {i.instance_name}
                </MenuItem>
              ))}
            </Select>
            <TextField
              size="small"
              placeholder="LogsQL"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              sx={{ flex: 1 }}
            />
            <Button
              variant="contained"
              startIcon={<PlayArrowIcon />}
              onClick={run}
              disabled={!instanceId || running}
            >
              查询
            </Button>
          </Box>

          {result ? (
            <Box
              component="pre"
              sx={{
                bgcolor: 'background.default',
                p: 2,
                borderRadius: 1,
                maxHeight: 480,
                overflow: 'auto',
                fontSize: 12,
              }}
            >
              {result}
            </Box>
          ) : (
            <Alert severity="info">
              M1 占位：后端会返回 "LogsQL query endpoint placeholder"，M4 起接入真实 VictoriaLogs。
            </Alert>
          )}

          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            支持的语法示例：<code>* | error</code>、<code>{'{service="api"} | status:500'}</code>
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}
