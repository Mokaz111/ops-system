import { Box, Button, Card, CardContent, Typography, Alert } from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import NotificationsActiveOutlinedIcon from '@mui/icons-material/NotificationsActiveOutlined';
import PageHeader from '../../components/common/PageHeader';

export default function AlertPage() {
  const n9eUrl = import.meta.env.VITE_N9E_URL || 'http://n9e.example.com';

  return (
    <Box>
      <PageHeader title="告警引擎" subtitle="告警管理由夜莺 (Nightingale / N9E) 独立提供" />

      <Alert severity="info" sx={{ mb: 3 }}>
        告警引擎已独立部署为 N9E (夜莺) 系统。告警规则配置、告警事件查看、通知渠道管理等功能请前往 N9E 控制台操作。
      </Alert>

      <Card sx={{ maxWidth: 600, mx: 'auto', textAlign: 'center' }}>
        <CardContent sx={{ py: 6 }}>
          <Box sx={{ mb: 3 }}>
            <Box
              sx={{
                width: 80,
                height: 80,
                borderRadius: '20px',
                background: 'linear-gradient(135deg, #e8f0fe, #d2e3fc)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                mx: 'auto',
                mb: 2,
              }}
            >
              <NotificationsActiveOutlinedIcon sx={{ fontSize: 40, color: 'primary.main' }} />
            </Box>
            <Typography variant="h6" gutterBottom>
              夜莺监控告警平台
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 400, mx: 'auto', mb: 3 }}>
              N9E 通过 n9e-edge 连接各租户的 VictoriaMetrics 实例，提供完整的告警规则管理、事件聚合、通知分发等能力。
            </Typography>
          </Box>

          <Button
            variant="contained"
            size="large"
            startIcon={<OpenInNewIcon />}
            onClick={() => window.open(n9eUrl, '_blank')}
            sx={{ px: 4, py: 1.5 }}
          >
            打开夜莺控制台
          </Button>

          <Box sx={{ mt: 4, display: 'flex', justifyContent: 'center', gap: 4 }}>
            <Box>
              <Typography variant="h6" color="primary.main">规则管理</Typography>
              <Typography variant="caption" color="text.secondary">告警规则 CRUD</Typography>
            </Box>
            <Box>
              <Typography variant="h6" color="primary.main">事件中心</Typography>
              <Typography variant="caption" color="text.secondary">告警事件与历史</Typography>
            </Box>
            <Box>
              <Typography variant="h6" color="primary.main">通知渠道</Typography>
              <Typography variant="caption" color="text.secondary">钉钉/邮件/Webhook</Typography>
            </Box>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}
