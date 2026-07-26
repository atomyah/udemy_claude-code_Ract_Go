import { Avatar, Box, Button, CircularProgress, Fade, Paper, Popper, Stack, Typography } from '@mui/material';
import type { SxProps, Theme } from '@mui/material/styles';
import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { useAuth } from '../../auth/AuthContext';
import { useFollow } from '../hooks/useFollow';
import { useProfile } from '../hooks/useProfile';

const OPEN_DELAY_MS = 400;
const CLOSE_DELAY_MS = 250;

interface UserHoverCardProps {
  handle: string;
  children: ReactNode;
  sx?: SxProps<Theme>;
}

// UserHoverCard はアバターや表示名にマウスオーバーしたときに、
// プロフィール概要とフォロー / フォロー中ボタンをポップアップ表示する。
export const UserHoverCard = ({ handle, children, sx }: UserHoverCardProps) => {
  const { currentUser, isAuthenticated } = useAuth();
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const [isActivated, setIsActivated] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 一度ホバーされるまでプロフィールは取得しない
  const { data: profile, isLoading, isError } = useProfile(isActivated ? handle : undefined);
  const followMutation = useFollow(handle);

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    },
    [],
  );

  const clearTimer = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  };

  const handleEnter = (event: MouseEvent<HTMLElement>) => {
    // タッチデバイスではホバーが成立しないため表示しない
    if (!window.matchMedia('(hover: hover)').matches) return;
    const target = event.currentTarget;
    clearTimer();
    timerRef.current = setTimeout(() => {
      setIsActivated(true);
      setAnchorEl(target);
    }, OPEN_DELAY_MS);
  };

  const handleLeave = () => {
    clearTimer();
    timerRef.current = setTimeout(() => setAnchorEl(null), CLOSE_DELAY_MS);
  };

  const isSelf = currentUser?.id !== undefined && currentUser.id === profile?.id;
  const profilePath = `/${profile?.handle ?? handle}`;

  return (
    <>
      <Box
        component="span"
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        sx={[
          { display: 'inline-flex', minWidth: 0, maxWidth: '100%' },
          ...(Array.isArray(sx) ? sx : [sx]),
        ]}
      >
        {children}
      </Box>

      <Popper
        open={Boolean(anchorEl)}
        anchorEl={anchorEl}
        placement="bottom-start"
        transition
        modifiers={[{ name: 'offset', options: { offset: [0, 8] } }]}
        sx={{ zIndex: (theme) => theme.zIndex.tooltip }}
      >
        {({ TransitionProps }) => (
          <Fade {...TransitionProps} timeout={150}>
            <Paper
              elevation={8}
              onMouseEnter={clearTimer}
              onMouseLeave={handleLeave}
              // ポップアップは Portal 経由でも React ツリー上は投稿カードの子なので、
              // クリックが投稿詳細への遷移に伝播しないよう止める
              onClick={(event) => event.stopPropagation()}
              sx={{ width: 300, p: 2, borderRadius: 3 }}
            >
              {isError ? (
                <Typography variant="body2" color="error">
                  プロフィールを取得できませんでした
                </Typography>
              ) : isLoading || !profile ? (
                <Stack sx={{ alignItems: 'center', py: 2 }}>
                  <CircularProgress size={24} />
                </Stack>
              ) : (
                <>
                  <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <Avatar
                      component={RouterLink}
                      to={profilePath}
                      src={profile.avatar_url ?? undefined}
                      sx={{ width: 56, height: 56 }}
                    >
                      {(profile.display_name ?? '?').charAt(0)}
                    </Avatar>
                    {isAuthenticated && !isSelf && (
                      <Button
                        size="small"
                        variant={profile.is_following ? 'outlined' : 'contained'}
                        onClick={() => followMutation.mutate(profile.is_following ?? false)}
                        disabled={followMutation.isPending}
                      >
                        {profile.is_following ? 'フォロー中' : 'フォローする'}
                      </Button>
                    )}
                  </Stack>

                  <Box sx={{ mt: 1 }}>
                    <Typography
                      component={RouterLink}
                      to={profilePath}
                      variant="subtitle1"
                      sx={{ fontWeight: 700, color: 'text.primary', textDecoration: 'none', display: 'block' }}
                    >
                      {profile.display_name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      @{profile.handle}
                    </Typography>
                  </Box>

                  {profile.bio && (
                    <Typography variant="body2" sx={{ mt: 1, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                      {profile.bio}
                    </Typography>
                  )}

                  <Stack direction="row" spacing={2} sx={{ mt: 1.5 }}>
                    <Typography
                      component={RouterLink}
                      to={profilePath}
                      variant="body2"
                      sx={{ color: 'text.secondary', textDecoration: 'none' }}
                    >
                      <Box component="strong" sx={{ color: 'text.primary' }}>
                        {profile.following_count ?? 0}
                      </Box>{' '}
                      フォロー中
                    </Typography>
                    <Typography
                      component={RouterLink}
                      to={profilePath}
                      variant="body2"
                      sx={{ color: 'text.secondary', textDecoration: 'none' }}
                    >
                      <Box component="strong" sx={{ color: 'text.primary' }}>
                        {profile.followers_count ?? 0}
                      </Box>{' '}
                      フォロワー
                    </Typography>
                  </Stack>
                </>
              )}
            </Paper>
          </Fade>
        )}
      </Popper>
    </>
  );
};
