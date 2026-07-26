import { Box, Stack, Typography } from '@mui/material';
import { useRef } from 'react';
import { useParams } from 'react-router-dom';
import { InfinitePostList } from '../components/InfinitePostList';
import { PostCard } from '../components/PostCard';
import { PostSkeleton } from '../components/PostCard/PostSkeleton';
import { useAuth } from '../features/auth/AuthContext';
import { CommentForm } from '../features/posts/components/CommentForm';
import { useComments } from '../features/posts/hooks/useComments';
import { usePost } from '../features/posts/hooks/usePost';

const PostDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const { isAuthenticated } = useAuth();
  const { data: post, isLoading, isError } = usePost(id ?? '');
  const comments = useComments(id ?? '');
  const commentInputRef = useRef<HTMLTextAreaElement>(null);

  // 詳細ページでは返信ボタンで返信欄にフォーカスする（同じページへの遷移は起きないため）
  const focusCommentInput = () => {
    const input = commentInputRef.current;
    if (!input) return;
    input.scrollIntoView({ block: 'center', behavior: 'smooth' });
    input.focus();
  };

  if (isLoading) return <PostSkeleton />;

  if (isError || !post) {
    return (
      <Typography color="error" sx={{ p: 2 }}>
        投稿が見つかりません
      </Typography>
    );
  }

  return (
    <Box>
      <PostCard post={post} onCommentClick={focusCommentInput} />

      {isAuthenticated && <CommentForm postId={post.id ?? ''} inputRef={commentInputRef} />}

      <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline', px: 2, py: 1.5 }}>
        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
          返信
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {post.comments_count ?? 0}件
        </Typography>
      </Stack>

      <InfinitePostList
        data={comments.data}
        isLoading={comments.isLoading}
        isError={comments.isError}
        isFetchingNextPage={comments.isFetchingNextPage}
        hasNextPage={comments.hasNextPage}
        fetchNextPage={comments.fetchNextPage}
        emptyMessage="まだコメントはありません"
      />
    </Box>
  );
};

export default PostDetailPage;
