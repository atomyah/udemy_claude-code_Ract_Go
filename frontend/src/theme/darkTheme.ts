import { createTheme } from '@mui/material/styles';
import { shape, typography } from './typography';

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#D0BCFF' },
    secondary: { main: '#4DD8AC' },
    background: { default: '#121016', paper: '#1D1B22' },
  },
  typography,
  shape,
});
