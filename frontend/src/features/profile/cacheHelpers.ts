import type { InfiniteData, Query, QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../api/queryKeys';
import type { UserListResponse, UserResponse } from '../../api/types';

type InfiniteUsersData = InfiniteData<UserListResponse>;

const isUserListQuery = (query: Query): boolean => {
  const key = query.queryKey as unknown[];
  if (key[0] === 'search' && key[1] === 'users') return true;
  if (key[0] === 'users' && (key[2] === 'followers' || key[2] === 'following')) return true;
  return false;
};

// updateUserInCaches はプロフィールキャッシュに加え、検索結果・フォロワー/フォロー中一覧など
// このユーザーが含まれうる全てのリストキャッシュも合わせて更新する
export const updateUserInCaches = (
  queryClient: QueryClient,
  handle: string,
  updater: (user: UserResponse) => UserResponse,
): void => {
  queryClient.setQueryData<UserResponse>(queryKeys.users.profile(handle), (old) => (old ? updater(old) : old));

  queryClient.setQueriesData<InfiniteUsersData>({ predicate: isUserListQuery }, (old) => {
    if (!old?.pages) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        data: (page.data ?? []).map((u) => (u.handle === handle ? updater(u) : u)),
      })),
    };
  });
};
