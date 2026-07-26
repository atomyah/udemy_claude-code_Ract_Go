import { IconButton, Stack, Typography } from '@mui/material';
import type { MouseEvent, ReactNode } from 'react';

interface ActionButtonProps {
  icon: ReactNode;
  count: number;
  active?: boolean;
  label: string;
  onClick: (event: MouseEvent) => void;
}

export const ActionButton = ({ icon, count, active, label, onClick }: ActionButtonProps) => (
  <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
    <IconButton size="small" onClick={onClick} color={active ? 'primary' : 'default'} aria-label={label}>
      {icon}
    </IconButton>
    {count > 0 && (
      <Typography variant="body2" color={active ? 'primary' : 'text.secondary'}>
        {count}
      </Typography>
    )}
  </Stack>
);
