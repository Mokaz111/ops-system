import { Box, Card } from '@mui/material';

interface FilterToolbarProps {
  children: React.ReactNode;
}

export default function FilterToolbar({ children }: FilterToolbarProps) {
  return (
    <Card sx={{ mb: 2 }}>
      <Box sx={{ p: 2, display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
        {children}
      </Box>
    </Card>
  );
}
