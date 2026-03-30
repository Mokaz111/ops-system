import { Box, Card, CardContent, Typography, Divider, TextField, Button, Grid } from '@mui/material';
import PageHeader from '../../components/common/PageHeader';
import { useAuthStore } from '../../stores/useAuthStore';

export default function SettingsPage() {
  const { user } = useAuthStore();

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
              <TextField fullWidth label="显示名" value={user?.display_name || ''} sx={{ mb: 2 }} />
              <TextField fullWidth label="邮箱" value={user?.email || ''} sx={{ mb: 2 }} />
              <TextField fullWidth label="手机号" value={user?.phone || ''} sx={{ mb: 2 }} />
              <Button variant="contained">保存修改</Button>
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
      </Grid>
    </Box>
  );
}
