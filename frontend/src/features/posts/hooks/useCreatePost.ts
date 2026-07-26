import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { createPost, type CreatePostPayload } from '../api';

export const useCreatePost = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreatePostPayload) => createPost(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.home() });
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.explore() });
    },
  });
};
