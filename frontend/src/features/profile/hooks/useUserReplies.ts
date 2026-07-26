import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getUserReplies } from '../api';

export const useUserReplies = (handle: string | undefined, enabled = true) =>
  useInfiniteQuery({
    queryKey: queryKeys.users.replies(handle ?? ''),
    queryFn: ({ pageParam }) => getUserReplies(handle ?? '', { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: !!handle && enabled,
  });
