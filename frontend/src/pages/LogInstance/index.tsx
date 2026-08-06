import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
} from '@mui/material';
import ManageSearchOutlinedIcon from '@mui/icons-material/ManageSearchOutlined';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import EmptyState from '../../components/common/EmptyState';
import LoadingScreen from '../../components/common/LoadingScreen';
import StatusChip from '../../components/common/StatusChip';
import { logAPI, type LogInstance } from '../../api/logs';
import { zoneAPI, type Zone } from '../../api/zone';
import { workspaceAPI } from '../../api/workspace';
import type { Workspace } from '../../types/api';
import { extractApiError } from '../../api';
import { useAuthStore } from '../../stores/useAuthStore';
import { isPlatformAdmin, getPrimaryWorkspaceId } from '../../utils/membership';

export default function LogInstancePage() {
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isPlatformAdminUser = isPlatformAdmin(user);
  // 日志实例的写接口在后端为平台管理员专属（RequireRole admin）。
  const isAdmin = isPlatformAdminUser;

  const [items, setItems] = useState<LogInstance[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    tenant_id: '',
    zone_id: '',
    instance_name: '',
    retention_days: 7,
  });

  const fetch = async () => {
    setLoading(true);
    try {
      const { data: res } = await logAPI.list({ page: 1, page_size: 50, keyword });
      setItems(res.data?.items || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetch();
  }, [keyword]);

  useEffect(() => {
    if (!isAdmin) return;
    zoneAPI.list({ page: 1, page_size: 100, status: 'active' }).then(({ data: res }) => {
      const list = res.data?.items || [];
      setZones(list);
      if (list.length > 0) setForm((f) => ({ ...f, zone_id: f.zone_id || list[0].id }));
    });
    // 平台管理员没有归属工作空间，创建时需显式选择目标租户。
    if (isPlatformAdminUser) {
      workspaceAPI.list({ page: 1, page_size: 100 }).then(({ data: res }) => {
        const list = res.data?.items || [];
        setWorkspaces(list);
        if (list.length > 0) setForm((f) => ({ ...f, tenant_id: f.tenant_id || list[0].id }));
      });
    }
  }, [isAdmin, isPlatformAdminUser]);

  const handleCreate = async () => {
    if (!form.instance_name || !form.zone_id) {
      enqueueSnackbar('名称和可用区为必填项', { variant: 'warning' });
      return;
    }
    const tenantId = isPlatformAdminUser ? form.tenant_id : getPrimaryWorkspaceId(user);
    if (!tenantId) {
      enqueueSnackbar(isPlatformAdminUser ? '请选择目标工作空间' : '当前用户未加入任何工作空间', { variant: 'error' });
      return;
    }
    setSaving(true);
    try {
      await logAPI.create({
        tenant_id: tenantId,
        zone_id: form.zone_id,
        instance_name: form.instance_name,
        retention_days: form.retention_days,
      });
      enqueueSnackbar('日志实例创建成功', { variant: 'success' });
      setDialogOpen(false);
      fetch();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建失败'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading && items.length === 0) return <LoadingScreen />;

  return (
    <Box>
      <PageHeader
        title="日志实例"
        subtitle="VictoriaLogs 日志数据面：绑定 Zone 日志管道（Vector → Kafka → Aggregator → VictoriaLogs）"
        actionLabel={isAdmin ? '创建日志实例' : undefined}
        onAction={isAdmin ? () => setDialogOpen(true) : undefined}
      />

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <TextField
          size="small"
          placeholder="搜索日志实例名称"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          sx={{ minWidth: 260 }}
        />
      </Stack>

      {items.length === 0 ? (
        <EmptyState
          title="暂无日志实例"
          description={isAdmin ? '请先对 Zone 执行 init-logs，再创建日志实例' : '当前工作空间下暂无日志实例'}
        />
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>名称</TableCell>
                <TableCell>Zone</TableCell>
                <TableCell>后端</TableCell>
                <TableCell>保留天数</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>创建时间</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((i) => (
                <TableRow key={i.id} hover>
                  <TableCell>{i.instance_name}</TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{i.zone_id?.substring(0, 8) || '-'}</TableCell>
                  <TableCell>{i.backend_type || 'victorialogs'}</TableCell>
                  <TableCell>{i.retention_days || '-'}</TableCell>
                  <TableCell>
                    <StatusChip status={i.status} />
                  </TableCell>
                  <TableCell>{new Date(i.created_at).toLocaleString()}</TableCell>
                  <TableCell align="right">
                    <Button
                      size="small"
                      startIcon={<ManageSearchOutlinedIcon />}
                      onClick={() => navigate(`/logs/query?instance_id=${i.id}`)}
                    >
                      查询日志
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>创建日志实例</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            需先在目标 Zone 完成 init-logs（VictoriaLogs + Kafka + Vector Aggregator）。
          </Alert>
          {isPlatformAdminUser && (
            <FormControl fullWidth size="small" sx={{ mb: 2 }}>
              <InputLabel>目标工作空间</InputLabel>
              <Select
                value={form.tenant_id}
                label="目标工作空间"
                onChange={(e) => setForm({ ...form, tenant_id: e.target.value })}
              >
                {workspaces.map((w) => (
                  <MenuItem key={w.id} value={w.id}>
                    {w.workspace_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>可用区</InputLabel>
            <Select
              value={form.zone_id}
              label="可用区"
              onChange={(e) => setForm({ ...form, zone_id: e.target.value })}
            >
              {zones.map((z) => (
                <MenuItem key={z.id} value={z.id}>
                  {z.display_name || z.slug}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <TextField
            fullWidth
            size="small"
            label="日志实例名称"
            value={form.instance_name}
            onChange={(e) => setForm({ ...form, instance_name: e.target.value })}
            sx={{ mb: 2 }}
            required
          />
          <TextField
            fullWidth
            size="small"
            type="number"
            label="保留天数"
            value={form.retention_days}
            onChange={(e) => setForm({ ...form, retention_days: Number(e.target.value) || 7 })}
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={handleCreate} disabled={saving}>
            {saving ? '创建中...' : '创建'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
