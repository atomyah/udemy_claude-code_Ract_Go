import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deletePost } from '../api';
import { removePostFromCaches } from '../cacheHelpers';

export const useDeletePost = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (postId: string) => deletePost(postId),
    onSuccess: (_, postId) => {
      removePostFromCaches(queryClient, postId);
    },
  });
};
