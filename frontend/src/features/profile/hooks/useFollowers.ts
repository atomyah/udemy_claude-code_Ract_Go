import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getFollowers } from '../api';

export const useFollowers = (handle: string | undefined, enabled: boolean) =>
  useInfiniteQuery({
    queryKey: queryKeys.users.followers(handle ?? ''),
    queryFn: ({ pageParam }) => getFollowers(handle ?? '', { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: !!handle && enabled,
  });
