import { useMutation, useQueryClient } from '@tanstack/react-query';
import { followUser, unfollowUser } from '../api';
import { updateUserInCaches } from '../cacheHelpers';

export const useFollow = (handle: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (isFollowing: boolean) =>
      isFollowing ? unfollowUser(handle) : followUser(handle).then(() => undefined),
    onMutate: (isFollowing) => {
      updateUserInCaches(queryClient, handle, (user) => ({
        ...user,
        is_following: !isFollowing,
        followers_count: (user.followers_count ?? 0) + (isFollowing ? -1 : 1),
      }));
    },
    onError: (_err, isFollowing) => {
      updateUserInCaches(queryClient, handle, (user) => ({
        ...user,
        is_following: isFollowing,
        followers_count: (user.followers_count ?? 0) + (isFollowing ? 1 : -1),
      }));
    },
  });
};
