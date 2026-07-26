import { Box } from '@mui/material';
import { Outlet } from 'react-router-dom';
import { ComposerProvider } from '../../features/posts/ComposerContext';
import { BottomNav } from './BottomNav';
import { Header } from './Header';
import { Sidebar } from './Sidebar';

export const AppLayout = () => (
  <ComposerProvider>
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <Sidebar />
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <Header />
        <Box component="main" sx={{ flexGrow: 1, p: { xs: 2, md: 3 }, pb: { xs: 9, md: 3 } }}>
          <Outlet />
        </Box>
      </Box>
      <BottomNav />
    </Box>
  </ComposerProvider>
);
