import { Box, Card, CardContent, Chip, Grid, Stack, Typography } from '@mui/material';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import RuleOutlinedIcon from '@mui/icons-material/RuleOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import LabelOutlinedIcon from '@mui/icons-material/LabelOutlined';
import PageHeader from '../../components/common/PageHeader';

const roadmap = [
  {
    icon: <LabelOutlinedIcon />,
    title: '标签匹配静默',
    description: '基于 Alertmanager silence API，按标签匹配器（matchers）静默一批告警。',
  },
  {
    icon: <ScheduleOutlinedIcon />,
    title: '计划性维护窗口',
    description: '为发布、演练等预定义时间窗口，窗口内命中的告警不再通知。',
  },
  {
    icon: <RuleOutlinedIcon />,
    title: '静默审计',
    description: '记录静默的创建人、原因与生效范围，到期自动恢复通知。',
  },
];

export default function AlertSilencesPage() {
  return (
    <Box>
      <PageHeader title="静默" subtitle="Alertmanager Silences：临时屏蔽指定告警的通知" />

      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          py: 6,
          mb: 3,
          color: 'text.disabled',
          border: '1px dashed',
          borderColor: 'divider',
          borderRadius: 2,
        }}
      >
        <NotificationsPausedOutlinedIcon sx={{ fontSize: 56, mb: 2 }} />
        <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
          <Typography variant="h6" color="text.secondary">
            告警静默建设中
          </Typography>
          <Chip size="small" label="Roadmap" color="primary" variant="outlined" />
        </Stack>
        <Typography variant="body2" color="text.disabled">
          将对接各工作空间 Alertmanager 的 silence API，支持标签匹配与维护窗口。
        </Typography>
      </Box>

      <Grid container spacing={2.5}>
        {roadmap.map((item, idx) => (
          <Grid size={{ xs: 12, md: 4 }} key={item.title}>
            <Card sx={{ height: '100%' }}>
              <CardContent>
                <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1.5 }}>
                  <Box sx={{ color: 'primary.main', display: 'flex' }}>{item.icon}</Box>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                    {idx + 1}. {item.title}
                  </Typography>
                </Stack>
                <Typography variant="body2" color="text.secondary">
                  {item.description}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}
