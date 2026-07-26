import SearchIcon from '@mui/icons-material/Search';
import { Box, InputAdornment, Tab, Tabs, TextField, Typography } from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { InfinitePostList } from '../components/InfinitePostList';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { useExploreTimeline } from '../features/posts/hooks/useExploreTimeline';
import { useSearchPosts } from '../features/posts/hooks/useSearchPosts';
import { useSearchUsers } from '../features/posts/hooks/useSearchUsers';
import { UserListItem } from '../features/profile/components/UserListItem';

const ExplorePage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [query, setQuery] = useState(searchParams.get('q') ?? '');
  const [tab, setTab] = useState(0);

  useEffect(() => {
    setQuery(searchParams.get('q') ?? '');
  }, [searchParams]);

  const trimmedQuery = query.trim();
  const isSearching = trimmedQuery.length > 0;

  const explore = useExploreTimeline(!isSearching);
  const search = useSearchPosts(trimmedQuery);
  const activePosts = isSearching ? search : explore;

  const userSearch = useSearchUsers(trimmedQuery);
  const users = userSearch.data?.pages.flatMap((page) => page.data ?? []) ?? [];
  const { hasNextPage: userHasNextPage, isFetchingNextPage: userIsFetchingNextPage, fetchNextPage: fetchNextUserPage } = userSearch;
  const userSentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const target = userSentinelRef.current;
    if (!target || tab !== 1) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && userHasNextPage && !userIsFetchingNextPage) {
          fetchNextUserPage();
        }
      },
      { rootMargin: '200px' },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [tab, userHasNextPage, userIsFetchingNextPage, fetchNextUserPage]);

  return (
    <Box>
      <TextField
        fullWidth
        placeholder="投稿やユーザーを検索"
        value={query}
        onChange={(e) => {
          const value = e.target.value;
          setQuery(value);
          setSearchParams(value ? { q: value } : {});
        }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          },
        }}
        sx={{ mb: 1 }}
      />

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="投稿" />
        <Tab label="ユーザー" />
      </Tabs>

      {tab === 0 ? (
        <InfinitePostList
          data={activePosts.data}
          isLoading={activePosts.isLoading}
          isError={activePosts.isError}
          isFetchingNextPage={activePosts.isFetchingNextPage}
          hasNextPage={activePosts.hasNextPage}
          fetchNextPage={activePosts.fetchNextPage}
          emptyMessage={isSearching ? '該当する投稿が見つかりません' : 'まだ投稿がありません'}
        />
      ) : !isSearching || trimmedQuery.length < 2 ? (
        <Typography color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
          2文字以上入力するとユーザーを検索できます
        </Typography>
      ) : userSearch.isLoading ? (
        <LoadingSpinner />
      ) : users.length === 0 ? (
        <Typography color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
          該当するユーザーが見つかりません
        </Typography>
      ) : (
        <Box>
          {users.map((user) => (
            <UserListItem key={user.id} user={user} />
          ))}
          <div ref={userSentinelRef} />
        </Box>
      )}
    </Box>
  );
};

export default ExplorePage;
