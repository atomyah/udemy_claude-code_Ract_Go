import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { useAdminDeletePost } from '../features/admin/hooks/useAdminDeletePost';
import { useAdminUsers } from '../features/admin/hooks/useAdminUsers';
import { useSuspendUser } from '../features/admin/hooks/useSuspendUser';
import { getApiErrorMessage } from '../utils/apiError';

const UserManagementTab = () => {
  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } = useAdminUsers();
  const suspendMutation = useSuspendUser();
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  const users = data?.pages.flatMap((page) => page.data ?? []) ?? [];

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) fetchNextPage();
      },
      { rootMargin: '200px' },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (isLoading) return <LoadingSpinner />;

  return (
    <Box>
      {users.map((user) => (
        <Stack
          key={user.id}
          direction="row"
          spacing={1.5}
          sx={{ alignItems: 'center', py: 1.5, borderBottom: 1, borderColor: 'divider' }}
        >
          <Avatar>{(user.display_name ?? '?').charAt(0)}</Avatar>
          <Box sx={{ flexGrow: 1, minWidth: 0 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
              {user.display_name} <Typography component="span" color="text.secondary">@{user.handle}</Typography>
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {user.email}
            </Typography>
          </Box>
          {user.is_suspended && <Chip label="停止中" color="error" size="small" />}
          <Button
            size="small"
            color={user.is_suspended ? 'primary' : 'error'}
            variant="outlined"
            disabled={suspendMutation.isPending}
            onClick={() =>
              suspendMutation.mutate({ id: user.id ?? '', isSuspended: user.is_suspended ?? false })
            }
          >
            {user.is_suspended ? '停止解除' : '停止'}
          </Button>
        </Stack>
      ))}
      <div ref={sentinelRef} />
      {isFetchingNextPage && <LoadingSpinner />}
    </Box>
  );
};

const PostManagementTab = () => {
  const [postId, setPostId] = useState('');
  const deleteMutation = useAdminDeletePost();

  return (
    <Box sx={{ maxWidth: 480 }}>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        投稿IDを入力して強制削除します（論理削除）。
      </Typography>
      <Stack direction="row" spacing={1}>
        <TextField
          label="投稿ID（UUID）"
          value={postId}
          onChange={(e) => setPostId(e.target.value)}
          fullWidth
          size="small"
        />
        <Button
          variant="contained"
          color="error"
          disabled={!postId || deleteMutation.isPending}
          onClick={() => deleteMutation.mutate(postId, { onSuccess: () => setPostId('') })}
        >
          削除
        </Button>
      </Stack>
      {deleteMutation.isError && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {getApiErrorMessage(deleteMutation.error, '削除に失敗しました')}
        </Alert>
      )}
      {deleteMutation.isSuccess && (
        <Alert severity="success" sx={{ mt: 2 }}>
          投稿を削除しました
        </Alert>
      )}
    </Box>
  );
};

const AdminPage = () => {
  const [tab, setTab] = useState(0);

  return (
    <Box>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 2 }}>
        管理者ダッシュボード
      </Typography>
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="ユーザー管理" />
        <Tab label="投稿管理" />
      </Tabs>
      {tab === 0 ? <UserManagementTab /> : <PostManagementTab />}
    </Box>
  );
};

export default AdminPage;
