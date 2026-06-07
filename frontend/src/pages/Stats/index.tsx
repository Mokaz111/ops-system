import { useState } from 'react';
import { Box, Tab, Tabs, Typography } from '@mui/material';
import BarChartOutlinedIcon from '@mui/icons-material/BarChartOutlined';
import PageHeader from '../../components/common/PageHeader';

function Placeholder({ type }: { type: string }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        py: 10,
        color: 'text.disabled',
        border: '1px dashed',
        borderColor: 'divider',
        borderRadius: 2,
        mt: 2,
      }}
    >
      <BarChartOutlinedIcon sx={{ fontSize: 56, mb: 2 }} />
      <Typography variant="h6" color="text.secondary" gutterBottom>
        {type}用量统计功能开发中
      </Typography>
      <Typography variant="body2" color="text.disabled">
        {type === 'VictoriaMetrics'
          ? 'VictoriaMetrics 集群的存储用量、写入速率、查询 QPS 等统计视图即将上线。'
          : '日志实例的存储用量、写入速率、日志量趋势等统计视图即将上线。'}
      </Typography>
    </Box>
  );
}

export default function StatsPage() {
  const [tab, setTab] = useState(0);

  return (
    <Box>
      <PageHeader
        title="用量统计"
        subtitle="监控实例的存储用量、写入速率与查询趋势"
      />
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 1 }}>
        <Tab label="VictoriaMetrics" />
        <Tab label="日志" />
      </Tabs>
      {tab === 0 && <Placeholder type="VictoriaMetrics" />}
      {tab === 1 && <Placeholder type="日志" />}
    </Box>
  );
}
