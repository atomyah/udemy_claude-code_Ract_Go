import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getExploreTimeline } from '../api';

export const useExploreTimeline = (enabled = true) =>
  useInfiniteQuery({
    queryKey: queryKeys.posts.explore(),
    queryFn: ({ pageParam }) => getExploreTimeline({ cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled,
  });
