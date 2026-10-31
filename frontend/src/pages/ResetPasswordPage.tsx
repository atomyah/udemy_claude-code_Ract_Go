import { Alert, Box, Button, Card, CardContent, CircularProgress, Link, TextField, Typography } from '@mui/material';
import { useForm } from 'react-hook-form';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { useResetPassword } from '../features/auth/hooks/usePasswordReset';
import { getApiErrorMessage } from '../utils/apiError';

interface ResetPasswordFormValues {
  password: string;
  confirmPassword: string;
}

/**
 * 運営から届いたリセットリンク（/reset-password?token=...）で新しいパスワードを設定する画面。
 * リンクは有効期限付き・一回限りで、無効な場合はバックエンドが INVALID_RESET_TOKEN を返す。
 */
const ResetPasswordPage = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';
  const { mutate, isPending, isSuccess, error } = useResetPassword();
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<ResetPasswordFormValues>({ defaultValues: { password: '', confirmPassword: '' } });

  const onSubmit = (values: ResetPasswordFormValues) => {
    mutate({ token, password: values.password });
  };

  const renderBody = () => {
    if (!token) {
      return (
        <Alert severity="error" data-testid="reset-password-missing-token">
          リセット用のリンクが正しくありません。メールに記載されたリンクをもう一度開くか、再度申請してください。
        </Alert>
      );
    }
    if (isSuccess) {
      return (
        <Alert severity="success" data-testid="reset-password-success">
          パスワードを再設定しました。新しいパスワードでログインしてください。
        </Alert>
      );
    }
    return (
      <>
        {error && (
          <Alert severity="error" data-testid="reset-password-error">
            {getApiErrorMessage(error, 'パスワードの再設定に失敗しました')}
          </Alert>
        )}
        <Box
          component="form"
          onSubmit={(e) => void handleSubmit(onSubmit)(e)}
          sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}
        >
          <TextField
            label="新しいパスワード"
            type="password"
            autoComplete="new-password"
            error={!!errors.password}
            helperText={errors.password?.message}
            slotProps={{ htmlInput: { 'data-testid': 'reset-password-password' } }}
            {...register('password', {
              required: 'パスワードを入力してください',
              minLength: { value: 8, message: 'パスワードは8文字以上で入力してください' },
              maxLength: { value: 72, message: 'パスワードは72文字以内で入力してください' },
            })}
          />
          <TextField
            label="新しいパスワード（確認）"
            type="password"
            autoComplete="new-password"
            error={!!errors.confirmPassword}
            helperText={errors.confirmPassword?.message}
            slotProps={{ htmlInput: { 'data-testid': 'reset-password-confirm' } }}
            {...register('confirmPassword', {
              required: '確認用パスワードを入力してください',
              validate: (value) => value === watch('password') || 'パスワードが一致しません',
            })}
          />
          <Button
            type="submit"
            variant="contained"
            size="large"
            disabled={isPending}
            data-testid="reset-password-submit"
          >
            {isPending ? <CircularProgress size={24} color="inherit" /> : '再設定する'}
          </Button>
        </Box>
      </>
    );
  };

  return (
    <Box
      data-testid="reset-password-page"
      sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', p: 2 }}
    >
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Typography variant="h5" sx={{ fontWeight: 700 }}>
            新しいパスワードの設定
          </Typography>
          {renderBody()}
          <Typography variant="body2" sx={{ textAlign: 'center' }}>
            <Link component={RouterLink} to="/login">
              ログイン画面へ
            </Link>
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
};

export default ResetPasswordPage;
