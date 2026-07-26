import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../../../api/client';
import { queryKeys } from '../../../api/queryKeys';
import type { UserListResponse } from '../../../api/types';

export const useSearchUsers = (q: string) =>
  useInfiniteQuery({
    queryKey: queryKeys.search.users(q),
    queryFn: ({ pageParam }) =>
      api
        .get<UserListResponse>('/search/users', { params: { q, cursor: pageParam, limit: 20 } })
        .then((res) => res.data),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: q.trim().length >= 2,
  });
