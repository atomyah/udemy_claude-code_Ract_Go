import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { Box, IconButton, Stack, Typography } from '@mui/material';
import axios from 'axios';
import { useNavigate, useParams } from 'react-router-dom';
import { InfinitePostList } from '../components/InfinitePostList';
import { useHashtagPosts } from '../features/posts/hooks/useHashtagPosts';

const HashtagPage = () => {
  const { tag } = useParams<{ tag: string }>();
  const navigate = useNavigate();
  const decodedTag = tag ?? '';
  const posts = useHashtagPosts(decodedTag);

  // 未使用のハッシュタグはバックエンドが404を返すため、エラーではなく空の結果として扱う
  const isNotFound = axios.isAxiosError(posts.error) && posts.error.response?.status === 404;

  return (
    <Box data-testid="hashtag-page">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 2 }}>
        <IconButton onClick={() => navigate(-1)} aria-label="戻る" data-testid="hashtag-back">
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h5" data-testid="hashtag-title" sx={{ fontWeight: 700 }}>
          #{decodedTag}
        </Typography>
      </Stack>

      <InfinitePostList
        data={posts.data}
        isLoading={posts.isLoading}
        isError={posts.isError && !isNotFound}
        isFetchingNextPage={posts.isFetchingNextPage}
        hasNextPage={posts.hasNextPage}
        fetchNextPage={posts.fetchNextPage}
        emptyMessage="このハッシュタグの投稿はありません"
      />
    </Box>
  );
};

export default HashtagPage;
