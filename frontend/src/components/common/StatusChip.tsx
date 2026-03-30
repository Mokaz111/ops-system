import { Chip, type ChipProps } from '@mui/material';

const statusMap: Record<string, { label: string; color: ChipProps['color'] }> = {
  running: { label: '运行中', color: 'success' },
  creating: { label: '创建中', color: 'info' },
  scaling: { label: '扩容中', color: 'info' },
  stopped: { label: '已停止', color: 'default' },
  error: { label: '异常', color: 'error' },
  deleting: { label: '删除中', color: 'warning' },
  active: { label: '正常', color: 'success' },
  inactive: { label: '禁用', color: 'default' },
  enabled: { label: '启用', color: 'success' },
  disabled: { label: '禁用', color: 'default' },
};

interface StatusChipProps {
  status: string;
  size?: 'small' | 'medium';
}

export default function StatusChip({ status, size = 'small' }: StatusChipProps) {
  const config = statusMap[status] || { label: status, color: 'default' as const };
  return <Chip label={config.label} color={config.color} variant="filled" size={size} />;
}
