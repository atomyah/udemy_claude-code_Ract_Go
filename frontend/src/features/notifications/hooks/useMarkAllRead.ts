import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import type { NotificationListResponse } from '../../../api/types';
import { markAllRead } from '../api';

export const useMarkAllRead = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      queryClient.setQueryData<{ pages: NotificationListResponse[]; pageParams: unknown[] }>(
        queryKeys.notifications.list(),
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              data: (page.data ?? []).map((n) => ({ ...n, is_read: true })),
              unread_count: 0,
            })),
          };
        },
      );
      queryClient.setQueryData(queryKeys.notifications.unreadCount(), 0);
    },
  });
};
