import AlternateEmailIcon from '@mui/icons-material/AlternateEmail';
import ChatBubbleOutlineIcon from '@mui/icons-material/ChatBubbleOutlineOutlined';
import FavoriteIcon from '@mui/icons-material/Favorite';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import RepeatIcon from '@mui/icons-material/Repeat';
import { Avatar, Box, Button, Stack, Typography } from '@mui/material';
import { type ReactNode, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner } from '../components/LoadingSpinner';
import type { NotificationResponse } from '../api/types';
import { useMarkAllRead } from '../features/notifications/hooks/useMarkAllRead';
import { useNotifications } from '../features/notifications/hooks/useNotifications';
import { formatRelativeTime } from '../utils/formatDate';

const TYPE_ICON: Record<string, ReactNode> = {
  like: <FavoriteIcon fontSize="small" color="error" />,
  comment: <ChatBubbleOutlineIcon fontSize="small" color="primary" />,
  follow: <PersonAddIcon fontSize="small" color="success" />,
  repost: <RepeatIcon fontSize="small" color="success" />,
  mention: <AlternateEmailIcon fontSize="small" color="primary" />,
};

const TYPE_LABEL: Record<string, string> = {
  like: 'があなたの投稿をいいねしました',
  comment: 'があなたの投稿にコメントしました',
  follow: 'にフォローされました',
  repost: 'があなたの投稿をリポストしました',
  mention: 'があなたをメンションしました',
};

const NotificationItem = ({ notification }: { notification: NotificationResponse }) => {
  const navigate = useNavigate();
  const actor = notification.actor;
  const type = notification.type ?? '';

  const handleClick = () => {
    if (type === 'follow') {
      navigate(`/${actor?.handle ?? ''}`);
    } else if (notification.post?.id) {
      navigate(`/posts/${notification.post.id}`);
    }
  };

  return (
    <Stack
      direction="row"
      spacing={1.5}
      onClick={handleClick}
      sx={{
        p: 2,
        cursor: 'pointer',
        borderBottom: 1,
        borderColor: 'divider',
        bgcolor: notification.is_read ? 'transparent' : 'action.hover',
        '&:hover': { bgcolor: 'action.selected' },
      }}
    >
      <Box sx={{ pt: 0.5 }}>{TYPE_ICON[type] ?? null}</Box>
      <Avatar src={actor?.avatar_url ?? undefined} sx={{ width: 36, height: 36 }}>
        {(actor?.display_name ?? '?').charAt(0)}
      </Avatar>
      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Typography variant="body2">
          <strong>{actor?.display_name}</strong>
          {TYPE_LABEL[type] ?? ''}
          <Typography component="span" variant="body2" color="text.secondary" sx={{ ml: 1 }}>
            ・{formatRelativeTime(notification.created_at ?? '')}
          </Typography>
        </Typography>
        {notification.post?.content && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }} noWrap>
            {notification.post.content}
          </Typography>
        )}
      </Box>
    </Stack>
  );
};

const NotificationsPage = () => {
  const notifications = useNotifications();
  const markAllReadMutation = useMarkAllRead();
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const markedRef = useRef(false);

  const items = notifications.data?.pages.flatMap((page) => page.data ?? []) ?? [];
  const unreadCount = notifications.data?.pages[0]?.unread_count ?? 0;
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = notifications;

  useEffect(() => {
    if (!markedRef.current && unreadCount > 0) {
      markedRef.current = true;
      markAllReadMutation.mutate();
    }
  }, [unreadCount, markAllReadMutation]);

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return undefined;

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
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <Box>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h5" sx={{ fontWeight: 700 }}>
          通知
        </Typography>
        <Button size="small" onClick={() => markAllReadMutation.mutate()} disabled={markAllReadMutation.isPending}>
          すべて既読にする
        </Button>
      </Stack>

      {notifications.isLoading ? (
        <LoadingSpinner />
      ) : items.length === 0 ? (
        <Typography color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
          通知はありません
        </Typography>
      ) : (
        <Box>
          {items.map((n) => (
            <NotificationItem key={n.id} notification={n} />
          ))}
          <div ref={sentinelRef} />
          {notifications.isFetchingNextPage && <LoadingSpinner />}
        </Box>
      )}
    </Box>
  );
};

export default NotificationsPage;
