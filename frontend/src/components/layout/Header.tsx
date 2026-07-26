import DarkModeIcon from '@mui/icons-material/DarkMode';
import LightModeIcon from '@mui/icons-material/LightMode';
import { AppBar, IconButton, Toolbar, Tooltip, Typography } from '@mui/material';
import { useThemeMode } from '../../theme/ThemeContext';

export const Header = () => {
  const { mode, toggleTheme } = useThemeMode();

  return (
    <AppBar position="sticky" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
      <Toolbar sx={{ justifyContent: 'space-between' }}>
        <Typography variant="h6" component="div" sx={{ fontWeight: 700 }}>
          SNS App
        </Typography>
        <Tooltip title={mode === 'light' ? 'ダークモードに切り替え' : 'ライトモードに切り替え'}>
          <IconButton onClick={toggleTheme} color="inherit" aria-label="テーマ切り替え">
            {mode === 'light' ? <DarkModeIcon /> : <LightModeIcon />}
          </IconButton>
        </Tooltip>
      </Toolbar>
    </AppBar>
  );
};
