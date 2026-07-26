import { Typography } from '@mui/material';
import { InfinitePostList } from '../components/InfinitePostList';
import { useBookmarksList } from '../features/posts/hooks/useBookmarksList';

const BookmarksPage = () => {
  const bookmarks = useBookmarksList();

  return (
    <>
      <Typography variant="h5" sx={{ fontWeight: 700, mb: 2 }}>
        ブックマーク
      </Typography>
      <InfinitePostList
        data={bookmarks.data}
        isLoading={bookmarks.isLoading}
        isError={bookmarks.isError}
        isFetchingNextPage={bookmarks.isFetchingNextPage}
        hasNextPage={bookmarks.hasNextPage}
        fetchNextPage={bookmarks.fetchNextPage}
        emptyMessage="ブックマークした投稿はありません"
      />
    </>
  );
};

export default BookmarksPage;
