import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { UpdatePostRequest } from '../../../api/types';
import { updatePost } from '../api';
import { updatePostInCaches } from '../cacheHelpers';

export const useUpdatePost = (postId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: UpdatePostRequest) => updatePost(postId, payload),
    onSuccess: (updated) => {
      updatePostInCaches(queryClient, postId, () => updated);
    },
  });
};
