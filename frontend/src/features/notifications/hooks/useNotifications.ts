import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { useAuth } from '../../auth/AuthContext';
import { getNotifications } from '../api';

export const useNotifications = () => {
  const { isAuthenticated } = useAuth();

  return useInfiniteQuery({
    queryKey: queryKeys.notifications.list(),
    queryFn: ({ pageParam }) => getNotifications({ cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: isAuthenticated,
  });
};
