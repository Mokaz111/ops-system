import { Box, Button, Typography } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  actionLabel?: string;
  onAction?: () => void;
  actionIcon?: React.ReactNode;
  extra?: React.ReactNode;
}

export default function PageHeader({ title, subtitle, actionLabel, onAction, actionIcon, extra }: PageHeaderProps) {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
      <Box>
        <Typography variant="h5" sx={{ fontWeight: 500, color: 'text.primary' }}>
          {title}
        </Typography>
        {subtitle && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            {subtitle}
          </Typography>
        )}
      </Box>
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
        {extra}
        {actionLabel && onAction && (
          <Button variant="contained" startIcon={actionIcon || <AddIcon />} onClick={onAction}>
            {actionLabel}
          </Button>
        )}
      </Box>
    </Box>
  );
}
