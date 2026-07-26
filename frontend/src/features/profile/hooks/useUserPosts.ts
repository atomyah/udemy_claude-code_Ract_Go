import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getUserPosts } from '../api';

export const useUserPosts = (handle: string | undefined) =>
  useInfiniteQuery({
    queryKey: queryKeys.users.posts(handle ?? ''),
    queryFn: ({ pageParam }) => getUserPosts(handle ?? '', { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: !!handle,
  });
