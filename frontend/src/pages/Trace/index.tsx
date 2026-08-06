import { Box, Card, CardContent, Chip, Grid, Stack, Typography } from '@mui/material';
import AccountTreeOutlinedIcon from '@mui/icons-material/AccountTreeOutlined';
import SensorsOutlinedIcon from '@mui/icons-material/SensorsOutlined';
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined';
import QueryStatsOutlinedIcon from '@mui/icons-material/QueryStatsOutlined';
import PageHeader from '../../components/common/PageHeader';

const roadmap = [
  {
    icon: <SensorsOutlinedIcon />,
    title: 'OTLP 接入',
    description: '业务集群部署 OpenTelemetry Collector，通过 OTLP gRPC/HTTP 上报 Trace 数据。',
  },
  {
    icon: <StorageOutlinedIcon />,
    title: 'Trace 存储',
    description: '按工作空间隔离的 Trace 存储后端（Tempo / Jaeger），与 Zone 架构对齐。',
  },
  {
    icon: <QueryStatsOutlinedIcon />,
    title: '调用链查询',
    description: 'TraceID 检索、服务拓扑、火焰图，以及与日志（trace_id 关联）和指标的联动跳转。',
  },
];

export default function TracePage() {
  return (
    <Box>
      <PageHeader title="调用链查询" subtitle="分布式链路追踪（Traces）" />

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
        <AccountTreeOutlinedIcon sx={{ fontSize: 56, mb: 2 }} />
        <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
          <Typography variant="h6" color="text.secondary">
            链路追踪系统建设中
          </Typography>
          <Chip size="small" label="Roadmap" color="primary" variant="outlined" />
        </Stack>
        <Typography variant="body2" color="text.disabled">
          链路追踪将作为独立的可观测性数据平面，与监控（Metrics）、日志（Logs）并列。
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
