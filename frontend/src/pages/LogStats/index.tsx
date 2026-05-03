import { Box, Typography } from '@mui/material';
import BarChartOutlinedIcon from '@mui/icons-material/BarChartOutlined';

export default function LogStatsPage() {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 1 }}>用量统计</Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        日志实例用量统计与趋势分析
      </Typography>
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
        }}
      >
        <BarChartOutlinedIcon sx={{ fontSize: 56, mb: 2 }} />
        <Typography variant="h6" color="text.secondary" gutterBottom>
          用量统计功能开发中
        </Typography>
        <Typography variant="body2" color="text.disabled">
          日志实例的存储用量、写入速率、日志量趋势等统计视图即将上线。
        </Typography>
      </Box>
    </Box>
  );
}
