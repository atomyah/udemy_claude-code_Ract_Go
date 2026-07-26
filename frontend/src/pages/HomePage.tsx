import { Box, Button, Paper } from '@mui/material';
import { InfinitePostList } from '../components/InfinitePostList';
import { PostForm } from '../features/posts/components/PostForm';
import { useHomeTimeline } from '../features/posts/hooks/useHomeTimeline';
import { useNewPostsBanner } from '../features/posts/hooks/useNewPostsBanner';

const HomePage = () => {
  const { data, isLoading, isError, isFetchingNextPage, hasNextPage, fetchNextPage } = useHomeTimeline();
  const { hasNewPosts, showLatest } = useNewPostsBanner();

  return (
    <Box>
      <Box sx={{ display: { xs: 'none', md: 'block' } }}>
        <PostForm />
      </Box>

      {hasNewPosts && (
        <Paper sx={{ position: 'sticky', top: 0, zIndex: 1, textAlign: 'center', py: 1 }} elevation={2}>
          <Button onClick={showLatest}>新しい投稿を見る</Button>
        </Paper>
      )}

      <InfinitePostList
        data={data}
        isLoading={isLoading}
        isError={isError}
        isFetchingNextPage={isFetchingNextPage}
        hasNextPage={hasNextPage}
        fetchNextPage={fetchNextPage}
        emptyMessage="まだ投稿がありません。最初の投稿をしてみましょう。"
      />
    </Box>
  );
};

export default HomePage;
