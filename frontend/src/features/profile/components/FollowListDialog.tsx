import CloseIcon from '@mui/icons-material/Close';
import { AppBar, Box, CircularProgress, Dialog, IconButton, Toolbar, Typography } from '@mui/material';
import { useEffect, useRef } from 'react';
import { useFollowers } from '../hooks/useFollowers';
import { useFollowing } from '../hooks/useFollowing';
import { UserListItem } from './UserListItem';

interface FollowListDialogProps {
  handle: string;
  mode: 'followers' | 'following';
  open: boolean;
  onClose: () => void;
}

export const FollowListDialog = ({ handle, mode, open, onClose }: FollowListDialogProps) => {
  const followers = useFollowers(handle, open && mode === 'followers');
  const following = useFollowing(handle, open && mode === 'following');
  const active = mode === 'followers' ? followers : following;
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = active;
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target || !open) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { rootMargin: '200px' },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [open, hasNextPage, isFetchingNextPage, fetchNextPage]);

  const users = active.data?.pages.flatMap((page) => page.data ?? []) ?? [];

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="xs">
      <AppBar position="static" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar>
          <IconButton edge="start" onClick={onClose} aria-label="閉じる">
            <CloseIcon />
          </IconButton>
          <Typography variant="h6" sx={{ ml: 1 }}>
            {mode === 'followers' ? 'フォロワー' : 'フォロー中'}
          </Typography>
        </Toolbar>
      </AppBar>
      <Box sx={{ minHeight: 200, maxHeight: '70vh', overflowY: 'auto' }}>
        {active.isLoading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
            <CircularProgress size={28} />
          </Box>
        ) : users.length === 0 ? (
          <Typography color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
            {mode === 'followers' ? 'フォロワーはいません' : 'フォロー中のユーザーはいません'}
          </Typography>
        ) : (
          users.map((user) => <UserListItem key={user.id} user={user} onNavigate={onClose} />)
        )}
        <div ref={sentinelRef} />
      </Box>
    </Dialog>
  );
};
