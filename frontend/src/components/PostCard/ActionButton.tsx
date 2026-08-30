import { IconButton, Stack, Typography } from '@mui/material';
import type { MouseEvent, ReactNode } from 'react';

interface ActionButtonProps {
  icon: ReactNode;
  count: number;
  active?: boolean;
  label: string;
  onClick: (event: MouseEvent) => void;
  /** E2Eテスト用のdata-testid接頭辞（例: post-like → post-like-button / post-like-count） */
  testIdPrefix?: string;
}

export const ActionButton = ({ icon, count, active, label, onClick, testIdPrefix }: ActionButtonProps) => (
  <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
    <IconButton
      size="small"
      onClick={onClick}
      color={active ? 'primary' : 'default'}
      aria-label={label}
      data-testid={testIdPrefix ? `${testIdPrefix}-button` : undefined}
      data-active={active ? 'true' : 'false'}
    >
      {icon}
    </IconButton>
    <Typography
      variant="body2"
      color={active ? 'primary' : 'text.secondary'}
      data-testid={testIdPrefix ? `${testIdPrefix}-count` : undefined}
    >
      {count > 0 ? count : ''}
    </Typography>
  </Stack>
);
