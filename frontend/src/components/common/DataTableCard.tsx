import { Card } from '@mui/material';

interface DataTableCardProps {
  children: React.ReactNode;
  pagination?: React.ReactNode;
}

export default function DataTableCard({ children, pagination }: DataTableCardProps) {
  return (
    <Card>
      {children}
      {pagination}
    </Card>
  );
}
