import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useToast } from '../../../components/ToastProvider';
import { getApiErrorMessage } from '../../../utils/apiError';
import { followUser, unfollowUser } from '../api';
import { updateUserInCaches } from '../cacheHelpers';

export const useFollow = (handle: string) => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

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
    onError: (error, isFollowing) => {
      updateUserInCaches(queryClient, handle, (user) => ({
        ...user,
        is_following: isFollowing,
        followers_count: (user.followers_count ?? 0) + (isFollowing ? 1 : -1),
      }));
      showToast(
        getApiErrorMessage(error, isFollowing ? 'フォロー解除に失敗しました' : 'フォローに失敗しました'),
        'error',
      );
    },
  });
};
