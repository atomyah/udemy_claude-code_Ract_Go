import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import type { AdminUserListResponse } from '../../../api/types';
import { suspendUser, unsuspendUser } from '../api';

export const useSuspendUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, isSuspended }: { id: string; isSuspended: boolean }) =>
      isSuspended ? unsuspendUser(id) : suspendUser(id),
    onSuccess: (_result, { id, isSuspended }) => {
      queryClient.setQueryData<{ pages: AdminUserListResponse[]; pageParams: unknown[] }>(
        queryKeys.admin.users(),
        (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              data: (page.data ?? []).map((u) => (u.id === id ? { ...u, is_suspended: !isSuspended } : u)),
            })),
          };
        },
      );
    },
  });
};
