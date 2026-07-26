import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getAdminUsers } from '../api';

export const useAdminUsers = () =>
  useInfiniteQuery({
    queryKey: queryKeys.admin.users(),
    queryFn: ({ pageParam }) => getAdminUsers({ cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
  });
