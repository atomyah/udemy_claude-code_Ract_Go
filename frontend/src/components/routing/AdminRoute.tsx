import { Box, CircularProgress } from '@mui/material';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../../features/auth/AuthContext';

export const AdminRoute = () => {
  const { currentUser, isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  // ロールは GET /users/me（本人の情報）で返る。表示制御のみで、権限の判定はバックエンドが DB で行う
  if (currentUser?.role !== 'admin') {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
};
