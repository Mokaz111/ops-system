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
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
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
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import AddIcon from '@mui/icons-material/Add';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import EmptyState from '../../components/common/EmptyState';
import { useAuthStore } from '../../stores/useAuthStore';
import { userAPI } from '../../api/user';
import { apiTokenAPI } from '../../api/apiToken';
import { extractApiError } from '../../api';
import type { APIToken } from '../../types/api';

const scopeLabels: Record<string, string> = {
  read: '只读',
  read_write: '读写',
};

export default function SettingsPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user, setUser } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [saving, setSaving] = useState(false);

  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [tokensLoading, setTokensLoading] = useState(false);
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false);
  const [tokenForm, setTokenForm] = useState({ name: '', scope: 'read_write' });
  const [creatingToken, setCreatingToken] = useState(false);
  const [newTokenPlain, setNewTokenPlain] = useState<string | null>(null);
  const [revokeDialog, setRevokeDialog] = useState<{ open: boolean; token?: APIToken }>({ open: false });

  useEffect(() => {
    setEmail(user?.email || '');
    setPhone(user?.phone || '');
  }, [user]);

  const fetchTokens = useCallback(async () => {
    setTokensLoading(true);
    try {
      const { data: res } = await apiTokenAPI.list({ page: 1, page_size: 50 });
      setTokens(res.data?.items || []);
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '获取 API Token 失败'), { variant: 'error' });
    } finally {
      setTokensLoading(false);
    }
  }, [enqueueSnackbar]);

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  const handleSave = async () => {
    if (!user?.id) {
      enqueueSnackbar('当前用户信息无效', { variant: 'error' });
      return;
    }
    setSaving(true);
    try {
      const { data: res } = await userAPI.update(user.id, { email, phone });
      setUser(res.data);
      enqueueSnackbar('个人信息已更新', { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '保存失败，请稍后重试'), { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleCreateToken = async () => {
    if (!tokenForm.name.trim()) {
      enqueueSnackbar('请输入 Token 名称', { variant: 'warning' });
      return;
    }
    setCreatingToken(true);
    try {
      const { data: res } = await apiTokenAPI.create({
        name: tokenForm.name.trim(),
        scope: tokenForm.scope as 'read' | 'read_write',
      });
      setNewTokenPlain(res.data.token);
      enqueueSnackbar('API Token 已创建', { variant: 'success' });
      fetchTokens();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '创建 Token 失败'), { variant: 'error' });
    } finally {
      setCreatingToken(false);
    }
  };

  const handleRevokeToken = async () => {
    if (!revokeDialog.token) return;
    try {
      await apiTokenAPI.revoke(revokeDialog.token.id);
      enqueueSnackbar('Token 已吊销', { variant: 'success' });
      setRevokeDialog({ open: false });
      fetchTokens();
    } catch (err) {
      enqueueSnackbar(extractApiError(err, '吊销失败'), { variant: 'error' });
    }
  };

  const copyToken = async (text: string) => {
    await navigator.clipboard.writeText(text);
    enqueueSnackbar('已复制到剪贴板', { variant: 'success' });
  };

  return (
    <Box>
      <PageHeader title="系统设置" subtitle="平台配置和个人信息管理" />

      <Grid container spacing={2.5}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>个人信息</Typography>
              <Divider sx={{ mb: 2 }} />
              <TextField fullWidth label="用户名" value={user?.username || ''} disabled sx={{ mb: 2 }} />
              <TextField fullWidth label="邮箱" value={email} onChange={(e) => setEmail(e.target.value)} sx={{ mb: 2 }} />
              <TextField fullWidth label="手机号" value={phone} onChange={(e) => setPhone(e.target.value)} sx={{ mb: 2 }} />
              <Button variant="contained" onClick={handleSave} disabled={saving}>
                {saving ? '保存中...' : '保存修改'}
              </Button>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>平台信息</Typography>
              <Divider sx={{ mb: 2 }} />
              <Box sx={{ mb: 1.5 }}>
                <Typography variant="body2" color="text.secondary">版本</Typography>
                <Typography variant="body1">v0.3.0</Typography>
              </Box>
              <Box sx={{ mb: 1.5 }}>
                <Typography variant="body2" color="text.secondary">API 地址</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{import.meta.env.VITE_API_BASE_URL || '/api/v1'}</Typography>
              </Box>
              <Box>
                <Typography variant="body2" color="text.secondary">Grafana</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{import.meta.env.VITE_GRAFANA_URL || '未配置'}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Box>
                  <Typography variant="subtitle1">API Token</Typography>
                  <Typography variant="body2" color="text.secondary">
                    用于程序化访问平台 API。明文 Token 仅在创建时显示一次。
                  </Typography>
                </Box>
                <Button startIcon={<AddIcon />} variant="contained" onClick={() => { setTokenForm({ name: '', scope: 'read_write' }); setNewTokenPlain(null); setTokenDialogOpen(true); }}>
                  创建 Token
                </Button>
              </Box>
              <Divider sx={{ mb: 2 }} />
              {tokensLoading ? (
                <Typography variant="body2" color="text.secondary">加载中...</Typography>
              ) : tokens.length === 0 ? (
                <EmptyState title="暂无 API Token" description="创建 Token 以通过脚本或 CI 调用平台 API" />
              ) : (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>名称</TableCell>
                        <TableCell>前缀</TableCell>
                        <TableCell>权限</TableCell>
                        <TableCell>最后使用</TableCell>
                        <TableCell>创建时间</TableCell>
                        <TableCell align="right">操作</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {tokens.map((t) => (
                        <TableRow key={t.id}>
                          <TableCell sx={{ fontWeight: 500 }}>{t.name}</TableCell>
                          <TableCell>
                            <Chip size="small" label={t.token_prefix} variant="outlined" sx={{ fontFamily: 'monospace' }} />
                          </TableCell>
                          <TableCell>{scopeLabels[t.scope] || t.scope}</TableCell>
                          <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                            {t.last_used_at ? new Date(t.last_used_at).toLocaleString() : '从未使用'}
                          </TableCell>
                          <TableCell sx={{ color: 'text.secondary', fontSize: '0.8125rem' }}>
                            {new Date(t.created_at).toLocaleDateString()}
                          </TableCell>
                          <TableCell align="right">
                            <Tooltip title="吊销">
                              <IconButton size="small" color="error" onClick={() => setRevokeDialog({ open: true, token: t })}>
                                <DeleteOutlinedIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </CardContent>
          </Card>
        </Grid>

        {isAdmin && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="info">
              共享 VM 集群初始化（含 Grafana + VMAuth）已移至可用区页面操作。
              请前往「可用区管理」→ 点击对应可用区的 <strong>初始化共享 VMCluster</strong> 按钮。
            </Alert>
          </Grid>
        )}
      </Grid>

      <Dialog open={tokenDialogOpen} onClose={() => setTokenDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{newTokenPlain ? 'Token 已创建' : '创建 API Token'}</DialogTitle>
        <DialogContent sx={{ pt: '16px !important' }}>
          {newTokenPlain ? (
            <>
              <Alert severity="warning" sx={{ mb: 2 }}>
                请立即复制并妥善保存 Token，关闭后将无法再次查看明文。
              </Alert>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <TextField fullWidth value={newTokenPlain} InputProps={{ readOnly: true, sx: { fontFamily: 'monospace', fontSize: '0.8125rem' } }} />
                <IconButton onClick={() => copyToken(newTokenPlain)}><ContentCopyIcon /></IconButton>
              </Box>
            </>
          ) : (
            <>
              <TextField fullWidth label="名称" value={tokenForm.name} onChange={(e) => setTokenForm({ ...tokenForm, name: e.target.value })} sx={{ mb: 2 }} required />
              <FormControl fullWidth size="small">
                <InputLabel>权限范围</InputLabel>
                <Select value={tokenForm.scope} label="权限范围" onChange={(e) => setTokenForm({ ...tokenForm, scope: e.target.value })}>
                  <MenuItem value="read">只读</MenuItem>
                  <MenuItem value="read_write">读写</MenuItem>
                </Select>
              </FormControl>
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setTokenDialogOpen(false)}>{newTokenPlain ? '关闭' : '取消'}</Button>
          {!newTokenPlain && (
            <Button variant="contained" onClick={handleCreateToken} disabled={creatingToken || !tokenForm.name.trim()}>
              {creatingToken ? '创建中...' : '创建'}
            </Button>
          )}
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={revokeDialog.open}
        title="吊销 API Token"
        message={`确定要吊销 Token「${revokeDialog.token?.name}」吗？使用该 Token 的请求将立即失效。`}
        severity="error"
        confirmLabel="吊销"
        onConfirm={handleRevokeToken}
        onCancel={() => setRevokeDialog({ open: false })}
      />
    </Box>
  );
}
