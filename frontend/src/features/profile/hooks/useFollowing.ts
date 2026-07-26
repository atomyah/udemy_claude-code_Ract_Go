import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getFollowing } from '../api';

export const useFollowing = (handle: string | undefined, enabled: boolean) =>
  useInfiniteQuery({
    queryKey: queryKeys.users.following(handle ?? ''),
    queryFn: ({ pageParam }) => getFollowing(handle ?? '', { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: !!handle && enabled,
  });
