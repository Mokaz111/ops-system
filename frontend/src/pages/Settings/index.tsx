import { useEffect, useState } from 'react';
import { Box, Card, CardContent, Typography, Divider, TextField, Button, Grid, Alert } from '@mui/material';
import { useSnackbar } from 'notistack';
import PageHeader from '../../components/common/PageHeader';
import ConfirmDialog from '../../components/common/ConfirmDialog';
import { useAuthStore } from '../../stores/useAuthStore';
import { userAPI } from '../../api/user';
import { platformAPI } from '../../api/platform';
import type { PlatformInitSharedClusterPlan } from '../../types/api';

export default function SettingsPage() {
  const { enqueueSnackbar } = useSnackbar();
  const { user, setUser } = useAuthStore();
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [saving, setSaving] = useState(false);
  const [initLoading, setInitLoading] = useState(false);
  const [initApplyConfirmOpen, setInitApplyConfirmOpen] = useState(false);
  const [initPlan, setInitPlan] = useState<PlatformInitSharedClusterPlan | null>(null);
  const [initForm, setInitForm] = useState({
    namespace: 'monitoring',
    release_name: 'vm-shared-stack',
  });

  useEffect(() => {
    setEmail(user?.email || '');
    setPhone(user?.phone || '');
  }, [user]);

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
    } catch {
      enqueueSnackbar('保存失败，请稍后重试', { variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleInitDryRun = async () => {
    setInitLoading(true);
    try {
      const { data: res } = await platformAPI.initSharedCluster({
        ...initForm,
        dry_run: true,
      });
      setInitPlan(res.data);
      enqueueSnackbar('共享集群初始化 dry-run 成功', { variant: 'success' });
    } catch {
      enqueueSnackbar('共享集群初始化 dry-run 失败', { variant: 'error' });
    } finally {
      setInitLoading(false);
    }
  };

  const handleInitApply = async () => {
    setInitLoading(true);
    try {
      const { data: res } = await platformAPI.initSharedCluster({
        ...initForm,
        dry_run: false,
      });
      setInitPlan(res.data);
      enqueueSnackbar('共享集群初始化已提交', { variant: 'success' });
      setInitApplyConfirmOpen(false);
    } catch {
      enqueueSnackbar('共享集群初始化提交失败', { variant: 'error' });
    } finally {
      setInitLoading(false);
    }
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
              <Box sx={{ mb: 1.5 }}>
                <Typography variant="body2" color="text.secondary">Grafana</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{import.meta.env.VITE_GRAFANA_URL || '未配置'}</Typography>
              </Box>
              <Box>
                <Typography variant="body2" color="text.secondary">夜莺 (N9E)</Typography>
                <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>{import.meta.env.VITE_N9E_URL || '未配置'}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle1" sx={{ mb: 2 }}>共享监控集群初始化（仅管理员）</Typography>
              <Divider sx={{ mb: 2 }} />
              {user?.role !== 'admin' ? (
                <Alert severity="info">当前账号不是 admin，无法执行共享集群初始化。</Alert>
              ) : (
                <>
                  <Alert severity="info" sx={{ mb: 2 }}>
                    将使用 Helm Chart `vm/victoria-metrics-k8s-stack` 初始化或升级全局共享监控集群，并启用内置 Grafana。
                  </Alert>
                  <Grid container spacing={2}>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Namespace"
                        value={initForm.namespace}
                        onChange={(e) => setInitForm((prev) => ({ ...prev, namespace: e.target.value }))}
                      />
                    </Grid>
                    <Grid size={{ xs: 12, md: 6 }}>
                      <TextField
                        fullWidth
                        size="small"
                        label="Release Name"
                        value={initForm.release_name}
                        onChange={(e) => setInitForm((prev) => ({ ...prev, release_name: e.target.value }))}
                      />
                    </Grid>
                  </Grid>
                  <Box sx={{ mt: 2 }}>
                    <Button variant="contained" onClick={handleInitDryRun} disabled={initLoading}>
                      {initLoading ? '执行中...' : '生成初始化 Dry-run'}
                    </Button>
                    <Button
                      variant="outlined"
                      sx={{ ml: 1 }}
                      onClick={() => setInitApplyConfirmOpen(true)}
                      disabled={initLoading || !initPlan}
                    >
                      应用初始化
                    </Button>
                  </Box>
                  <Box
                    component="pre"
                    sx={{
                      p: 2,
                      borderRadius: 1,
                      backgroundColor: '#f8f9fa',
                      fontSize: 12,
                      overflowX: 'auto',
                      m: 0,
                      mt: 2,
                    }}
                  >
                    {initPlan ? JSON.stringify(initPlan, null, 2) : '暂无初始化预览，请先执行 dry-run。'}
                  </Box>
                </>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <ConfirmDialog
        open={initApplyConfirmOpen}
        title="确认初始化共享监控集群"
        message="将对共享监控集群执行 helm install/upgrade。建议先执行 dry-run 并核对预览内容。是否继续？"
        confirmLabel="确认初始化"
        severity="warning"
        loading={initLoading}
        onConfirm={handleInitApply}
        onCancel={() => setInitApplyConfirmOpen(false)}
      />
    </Box>
  );
}
