import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { searchPosts } from '../api';

export const useSearchPosts = (q: string) =>
  useInfiniteQuery({
    queryKey: queryKeys.search.posts(q),
    queryFn: ({ pageParam }) => searchPosts(q, { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: q.length > 0,
  });
