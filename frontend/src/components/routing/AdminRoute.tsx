import { Box, CircularProgress } from '@mui/material';
import { Navigate, Outlet } from 'react-router-dom';
import { getAccessToken } from '../../api/client';
import { useAuth } from '../../features/auth/AuthContext';
import { decodeJwt } from '../../utils/jwt';

export const AdminRoute = () => {
  const { isAuthenticated, isLoading } = useAuth();

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

  const token = getAccessToken();
  const role = token ? decodeJwt(token)?.role : undefined;

  if (role !== 'admin') {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
};
