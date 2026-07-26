import { createTheme } from '@mui/material/styles';
import { shape, typography } from './typography';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#6750A4' },
    secondary: { main: '#00A879' },
    background: { default: '#F7F6FB', paper: '#FFFFFF' },
  },
  typography,
  shape,
});
