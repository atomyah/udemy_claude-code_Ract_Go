import DarkModeIcon from '@mui/icons-material/DarkMode';
import LightModeIcon from '@mui/icons-material/LightMode';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Divider,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import { useState } from 'react';
import { useAuth } from '../features/auth/AuthContext';
import { useChangeEmail } from '../features/settings/hooks/useChangeEmail';
import { useChangePassword } from '../features/settings/hooks/useChangePassword';
import { useThemeMode } from '../theme/ThemeContext';
import { getApiErrorMessage } from '../utils/apiError';

const SettingsPage = () => {
  const { currentUser } = useAuth();
  const { mode, toggleTheme } = useThemeMode();

  const [newEmail, setNewEmail] = useState('');
  const [emailPassword, setEmailPassword] = useState('');
  const changeEmailMutation = useChangeEmail();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const changePasswordMutation = useChangePassword();

  const handleChangeEmail = () => {
    changeEmailMutation.mutate(
      { new_email: newEmail, current_password: emailPassword },
      { onSuccess: () => { setNewEmail(''); setEmailPassword(''); } },
    );
  };

  const handleChangePassword = () => {
    changePasswordMutation.mutate(
      { current_password: currentPassword, new_password: newPassword },
      { onSuccess: () => { setCurrentPassword(''); setNewPassword(''); } },
    );
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: 480 }}>
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        設定
      </Typography>

      <Card variant="outlined">
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 1.5 }}>
            テーマ
          </Typography>
          <ToggleButtonGroup
            value={mode}
            exclusive
            onChange={(_, value) => {
              if (value && value !== mode) toggleTheme();
            }}
          >
            <ToggleButton value="light">
              <LightModeIcon fontSize="small" sx={{ mr: 1 }} />
              ライト
            </ToggleButton>
            <ToggleButton value="dark">
              <DarkModeIcon fontSize="small" sx={{ mr: 1 }} />
              ダーク
            </ToggleButton>
          </ToggleButtonGroup>
        </CardContent>
      </Card>

      <Card variant="outlined">
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 0.5 }}>
            メールアドレス変更
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
            現在のメールアドレス: {currentUser?.email ?? '取得できませんでした'}
          </Typography>
          <Stack spacing={2}>
            <TextField
              label="新しいメールアドレス"
              type="email"
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              fullWidth
            />
            <TextField
              label="現在のパスワード"
              type="password"
              value={emailPassword}
              onChange={(e) => setEmailPassword(e.target.value)}
              fullWidth
            />
            {changeEmailMutation.isError && (
              <Alert severity="error">{getApiErrorMessage(changeEmailMutation.error, '変更に失敗しました')}</Alert>
            )}
            {changeEmailMutation.isSuccess && <Alert severity="success">メールアドレスを変更しました</Alert>}
            <Button
              variant="contained"
              onClick={handleChangeEmail}
              disabled={!newEmail || !emailPassword || changeEmailMutation.isPending}
              sx={{ alignSelf: 'flex-start' }}
            >
              変更する
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card variant="outlined">
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 1.5 }}>
            パスワード変更
          </Typography>
          <Stack spacing={2}>
            <TextField
              label="現在のパスワード"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              fullWidth
            />
            <TextField
              label="新しいパスワード（8文字以上）"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              fullWidth
            />
            {changePasswordMutation.isError && (
              <Alert severity="error">{getApiErrorMessage(changePasswordMutation.error, '変更に失敗しました')}</Alert>
            )}
            {changePasswordMutation.isSuccess && <Alert severity="success">パスワードを変更しました</Alert>}
            <Button
              variant="contained"
              onClick={handleChangePassword}
              disabled={!currentPassword || newPassword.length < 8 || changePasswordMutation.isPending}
              sx={{ alignSelf: 'flex-start' }}
            >
              変更する
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Divider />
    </Box>
  );
};

export default SettingsPage;
