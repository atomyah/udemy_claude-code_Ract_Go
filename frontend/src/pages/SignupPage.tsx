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
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { useGoogleLogin } from '../features/auth/hooks/useGoogleLogin';
import { useRegister } from '../features/auth/hooks/useRegister';
import { getApiErrorMessage } from '../utils/apiError';

interface SignupFormValues {
  email: string;
  displayName: string;
  handle: string;
  password: string;
  confirmPassword: string;
}

const SignupPage = () => {
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();
  const { mutate: registerUser, isPending, error } = useRegister();
  const { mutate: googleLogin, isPending: isGooglePending, error: googleError } = useGoogleLogin();

  const handleGoogleLogin = () => {
    googleLogin(undefined, { onSuccess: () => navigate('/', { replace: true }) });
  };

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<SignupFormValues>({
    defaultValues: { email: '', displayName: '', handle: '', password: '', confirmPassword: '' },
  });

  const onSubmit = (values: SignupFormValues) => {
    registerUser(
      {
        email: values.email,
        display_name: values.displayName,
        handle: values.handle,
        password: values.password,
      },
      { onSuccess: () => navigate('/', { replace: true }) },
    );
  };

  return (
    <Box
      data-testid="signup-page"
      sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', p: 2 }}
    >
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Typography variant="h5" sx={{ fontWeight: 700 }}>
            サインアップ
          </Typography>

          {error && (
            <Alert severity="error" data-testid="signup-error">
              {getApiErrorMessage(error, '登録に失敗しました。入力内容をご確認ください')}
            </Alert>
          )}
          {googleError && (
            <Alert severity="error">{getApiErrorMessage(googleError, 'Google登録に失敗しました')}</Alert>
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
              slotProps={{ htmlInput: { 'data-testid': 'signup-email' } }}
              {...register('email', {
                required: 'メールアドレスを入力してください',
                pattern: {
                  value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                  message: 'メールアドレスの形式が正しくありません',
                },
              })}
            />
            <TextField
              label="表示名"
              autoComplete="name"
              error={!!errors.displayName}
              helperText={errors.displayName?.message}
              slotProps={{ htmlInput: { 'data-testid': 'signup-display-name' } }}
              {...register('displayName', {
                required: '表示名を入力してください',
                maxLength: { value: 50, message: '表示名は50文字以内で入力してください' },
              })}
            />
            <TextField
              label="ユーザーID（@handle）"
              autoComplete="username"
              error={!!errors.handle}
              helperText={errors.handle?.message ?? '英数字とアンダースコアのみ使用できます'}
              slotProps={{ htmlInput: { 'data-testid': 'signup-handle' } }}
              {...register('handle', {
                required: 'ユーザーIDを入力してください',
                pattern: {
                  value: /^[A-Za-z0-9_]{1,50}$/,
                  message: '英数字とアンダースコアのみ使用できます',
                },
              })}
            />
            <TextField
              label="パスワード"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
              error={!!errors.password}
              helperText={errors.password?.message}
              slotProps={{
                htmlInput: { 'data-testid': 'signup-password' },
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
              {...register('password', {
                required: 'パスワードを入力してください',
                minLength: { value: 8, message: 'パスワードは8文字以上で入力してください' },
                maxLength: { value: 72, message: 'パスワードは72文字以内で入力してください' },
              })}
            />
            <TextField
              label="パスワード（確認）"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
              error={!!errors.confirmPassword}
              helperText={errors.confirmPassword?.message}
              slotProps={{ htmlInput: { 'data-testid': 'signup-confirm-password' } }}
              {...register('confirmPassword', {
                required: '確認用パスワードを入力してください',
                validate: (value) => value === watch('password') || 'パスワードが一致しません',
              })}
            />
            <Button type="submit" variant="contained" size="large" disabled={isPending} data-testid="signup-submit">
              {isPending ? <CircularProgress size={24} color="inherit" /> : '登録する'}
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
            Google で登録
          </Button>

          <Typography variant="body2" sx={{ textAlign: 'center' }}>
            すでにアカウントをお持ちの方は <Link component={RouterLink} to="/login">ログイン</Link>
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
};

export default SignupPage;
