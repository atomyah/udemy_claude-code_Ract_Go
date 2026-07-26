import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { useAuth } from '../../auth/AuthContext';
import { getBookmarks } from '../api';

export const useBookmarksList = () => {
  const { isAuthenticated } = useAuth();

  return useInfiniteQuery({
    queryKey: queryKeys.bookmarks.list(),
    queryFn: ({ pageParam }) => getBookmarks({ cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: isAuthenticated,
  });
};
