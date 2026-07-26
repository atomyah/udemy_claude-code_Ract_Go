import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { useAuth } from '../../auth/AuthContext';
import { getHomeTimeline } from '../api';

export const useHomeTimeline = () => {
  const { isAuthenticated } = useAuth();

  return useInfiniteQuery({
    queryKey: queryKeys.posts.home(),
    queryFn: ({ pageParam }) => getHomeTimeline({ cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: isAuthenticated,
  });
};
