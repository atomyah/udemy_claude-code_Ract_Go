import { useQuery } from '@tanstack/react-query';
import { api } from '../../../api/client';
import { queryKeys } from '../../../api/queryKeys';
import type { NotificationListResponse } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';

export const useUnreadNotificationsCount = () => {
  const { isAuthenticated } = useAuth();

  return useQuery({
    queryKey: queryKeys.notifications.unreadCount(),
    queryFn: () =>
      api
        .get<NotificationListResponse>('/notifications', { params: { limit: 1 } })
        .then((res) => res.data.unread_count ?? 0),
    enabled: isAuthenticated,
    refetchInterval: 30_000,
  });
};
