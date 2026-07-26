import GoogleIcon from '@mui/icons-material/Google';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  IconButton,
  Link,
  TextField,
  Typography,
} from '@mui/material';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link as RouterLink, useLocation, useNavigate, type Location } from 'react-router-dom';
import { useGoogleLogin } from '../features/auth/hooks/useGoogleLogin';
import { useLogin } from '../features/auth/hooks/useLogin';
import { getApiErrorMessage } from '../utils/apiError';

interface LoginFormValues {
  email: string;
  password: string;
}

const LoginPage = () => {
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { mutate: login, isPending, error } = useLogin();
  const { mutate: googleLogin, isPending: isGooglePending, error: googleError } = useGoogleLogin();

  const handleGoogleLogin = () => {
    googleLogin(undefined, {
      onSuccess: () => {
        const from = (location.state as { from?: Location } | null)?.from?.pathname ?? '/';
        navigate(from, { replace: true });
      },
    });
  };

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({ defaultValues: { email: '', password: '' } });

  const onSubmit = (values: LoginFormValues) => {
    login(values, {
      onSuccess: () => {
        const from = (location.state as { from?: Location } | null)?.from?.pathname ?? '/';
        navigate(from, { replace: true });
      },
    });
  };

  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', p: 2 }}>
      <Card sx={{ width: '100%', maxWidth: 400 }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Typography variant="h5" sx={{ fontWeight: 700 }}>
            ログイン
          </Typography>

          {error && (
            <Alert severity="error">
              {getApiErrorMessage(error, 'メールアドレスまたはパスワードが正しくありません')}
            </Alert>
          )}
          {googleError && (
            <Alert severity="error">{getApiErrorMessage(googleError, 'Googleログインに失敗しました')}</Alert>
          )}

          <Box
            component="form"
            onSubmit={(e) => void handleSubmit(onSubmit)(e)}
            sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}
          >
            <TextField
              label="メールアドレス"
              type="email"
              autoComplete="email"
              error={!!errors.email}
              helperText={errors.email?.message}
              {...register('email', {
                required: 'メールアドレスを入力してください',
                pattern: {
                  value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                  message: 'メールアドレスの形式が正しくありません',
                },
              })}
            />
            <TextField
              label="パスワード"
              type={showPassword ? 'text' : 'password'}
              autoComplete="current-password"
              error={!!errors.password}
              helperText={errors.password?.message}
              slotProps={{
                input: {
                  endAdornment: (
                    <IconButton
                      onClick={() => setShowPassword((v) => !v)}
                      edge="end"
                      aria-label="パスワード表示切替"
                    >
                      {showPassword ? <VisibilityOffIcon /> : <VisibilityIcon />}
                    </IconButton>
                  ),
                },
              }}
              {...register('password', { required: 'パスワードを入力してください' })}
            />
            <Button type="submit" variant="contained" size="large" disabled={isPending}>
              {isPending ? <CircularProgress size={24} color="inherit" /> : 'ログイン'}
            </Button>
          </Box>

          <Divider>または</Divider>

          <Button
            variant="outlined"
            size="large"
            startIcon={isGooglePending ? <CircularProgress size={18} /> : <GoogleIcon />}
            onClick={handleGoogleLogin}
            disabled={isGooglePending}
          >
            Google でログイン
          </Button>

          <Typography variant="body2" sx={{ textAlign: 'center' }}>
            アカウントをお持ちでない方は <Link component={RouterLink} to="/signup">新規登録</Link>
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
};

export default LoginPage;
