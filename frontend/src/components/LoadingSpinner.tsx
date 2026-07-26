import { Box, CircularProgress } from '@mui/material';

interface LoadingSpinnerProps {
  fullHeight?: boolean;
}

export const LoadingSpinner = ({ fullHeight = false }: LoadingSpinnerProps) => (
  <Box
    sx={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      p: 4,
      minHeight: fullHeight ? '100vh' : undefined,
    }}
  >
    <CircularProgress />
  </Box>
);
