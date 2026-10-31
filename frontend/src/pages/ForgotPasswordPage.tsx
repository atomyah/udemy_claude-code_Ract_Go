import { Alert, Box, Button, Card, CardContent, CircularProgress, Link, TextField, Typography } from '@mui/material';
import { useForm } from 'react-hook-form';
import { Link as RouterLink } from 'react-router-dom';
import { useRequestPasswordReset } from '../features/auth/hooks/usePasswordReset';
import { getApiErrorMessage } from '../utils/apiError';

interface ForgotPasswordFormValues {
  email: string;
}

/**
 * パスワードリセットの申請画面。
 * 申請を受けた運営が本人確認のうえリセットリンクを発行し、メールで送付する（自動送信はしない）。
 */
const ForgotPasswordPage = () => {
  const { mutate, isPending, isSuccess, data, error } = useRequestPasswordReset();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotPasswordFormValues>({ defaultValues: { email: '' } });

  const onSubmit = (values: ForgotPasswordFormValues) => {
    mutate({ email: values.email });
  };

  return (
    <Box
      data-testid="forgot-password-page"
      sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', p: 2 }}
    >
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Typography variant="h5" sx={{ fontWeight: 700 }}>
            パスワードの再設定
          </Typography>

          {isSuccess ? (
            <Alert severity="success" data-testid="forgot-password-success">
              {data.message}
            </Alert>
          ) : (
            <>
              <Typography variant="body2" color="text.secondary">
                登録したメールアドレスを入力してください。運営が確認のうえ、再設定用のリンクをメールでお送りします。
              </Typography>

              {error && (
                <Alert severity="error" data-testid="forgot-password-error">
                  {getApiErrorMessage(error, '申請に失敗しました。時間をおいて再度お試しください')}
                </Alert>
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
                  slotProps={{ htmlInput: { 'data-testid': 'forgot-password-email' } }}
                  {...register('email', {
                    required: 'メールアドレスを入力してください',
                    pattern: {
                      value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                      message: 'メールアドレスの形式が正しくありません',
                    },
                  })}
                />
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  disabled={isPending}
                  data-testid="forgot-password-submit"
                >
                  {isPending ? <CircularProgress size={24} color="inherit" /> : '申請する'}
                </Button>
              </Box>
            </>
          )}

          <Typography variant="body2" sx={{ textAlign: 'center' }}>
            <Link component={RouterLink} to="/login">
              ログイン画面に戻る
            </Link>
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
};

export default ForgotPasswordPage;
