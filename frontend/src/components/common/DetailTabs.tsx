import { Box, Tab, Tabs } from '@mui/material';

export interface DetailTabItem {
  key: string;
  label: string;
  content: React.ReactNode;
}

interface DetailTabsProps {
  value: string;
  onChange: (value: string) => void;
  items: DetailTabItem[];
}

export default function DetailTabs({ value, onChange, items }: DetailTabsProps) {
  const active = items.find((item) => item.key === value) || items[0];

  return (
    <>
      <Tabs value={active.key} onChange={(_, next) => onChange(next)} sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        {items.map((item) => (
          <Tab key={item.key} value={item.key} label={item.label} />
        ))}
      </Tabs>
      <Box>{active.content}</Box>
    </>
  );
}
