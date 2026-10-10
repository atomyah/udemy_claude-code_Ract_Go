import SearchIcon from '@mui/icons-material/Search';
import {
  Avatar,
  Box,
  ClickAwayListener,
  InputAdornment,
  List,
  ListItemButton,
  Paper,
  Popper,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { InfinitePostList } from '../components/InfinitePostList';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { useExploreTimeline } from '../features/posts/hooks/useExploreTimeline';
import { useSearchPosts } from '../features/posts/hooks/useSearchPosts';
import { useSearchUsers } from '../features/posts/hooks/useSearchUsers';
import { UserListItem } from '../features/profile/components/UserListItem';

/** サジェストに表示する候補の最大件数 */
const SUGGESTION_LIMIT = 5;

const ExplorePage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const [query, setQuery] = useState(searchParams.get('q') ?? '');
  const [tab, setTab] = useState(0);
  const [suggestOpen, setSuggestOpen] = useState(false);
  const searchFieldRef = useRef<HTMLDivElement | null>(null);

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
  const {
    hasNextPage: userHasNextPage,
    isFetchingNextPage: userIsFetchingNextPage,
    fetchNextPage: fetchNextUserPage,
  } = userSearch;
  const userSentinelRef = useRef<HTMLDivElement | null>(null);

  const suggestions = users.slice(0, SUGGESTION_LIMIT);
  const canSuggest = trimmedQuery.length >= 2;
  const showSuggestions = suggestOpen && canSuggest && suggestions.length > 0;

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
      <ClickAwayListener onClickAway={() => setSuggestOpen(false)}>
        <Box>
          <TextField
            fullWidth
            ref={searchFieldRef}
            placeholder="投稿やユーザーを検索"
            value={query}
            onFocus={() => setSuggestOpen(true)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') setSuggestOpen(false);
            }}
            onChange={(e) => {
              const value = e.target.value;
              setQuery(value);
              setSuggestOpen(true);
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
              htmlInput: { 'data-testid': 'explore-search-input' },
            }}
            sx={{ mb: 1 }}
          />

          <Popper
            open={showSuggestions}
            anchorEl={searchFieldRef.current}
            placement="bottom-start"
            sx={{ zIndex: (theme) => theme.zIndex.modal, width: searchFieldRef.current?.clientWidth }}
          >
            <Paper elevation={4} data-testid="user-suggestions">
              <List disablePadding>
                {suggestions.map((user) => (
                  <ListItemButton
                    key={user.id}
                    data-testid="user-suggestion-item"
                    data-handle={user.handle ?? ''}
                    onClick={() => {
                      setSuggestOpen(false);
                      navigate(`/${user.handle ?? ''}`);
                    }}
                  >
                    <Avatar src={user.avatar_url ?? undefined} sx={{ width: 32, height: 32, mr: 1.5 }}>
                      {(user.display_name ?? '?').charAt(0)}
                    </Avatar>
                    <Box sx={{ minWidth: 0 }}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 700 }} noWrap>
                        {user.display_name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" noWrap>
                        @{user.handle}
                      </Typography>
                    </Box>
                  </ListItemButton>
                ))}
              </List>
            </Paper>
          </Popper>
        </Box>
      </ClickAwayListener>

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="投稿" data-testid="explore-tab-posts" />
        <Tab label="ユーザー" data-testid="explore-tab-users" />
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
      ) : !canSuggest ? (
        <Typography color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
          2文字以上入力するとユーザーを検索できます
        </Typography>
      ) : userSearch.isLoading ? (
        <LoadingSpinner />
      ) : users.length === 0 ? (
        <Typography color="text.secondary" data-testid="user-list-empty" sx={{ p: 2, textAlign: 'center' }}>
          該当するユーザーが見つかりません
        </Typography>
      ) : (
        <Box data-testid="user-list">
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
