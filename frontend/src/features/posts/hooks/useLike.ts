import { useMutation, useQueryClient } from '@tanstack/react-query';
import { like, unlike } from '../api';
import { updatePostInCaches } from '../cacheHelpers';

export const useLike = (postId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (isLiked: boolean) => (isLiked ? unlike(postId) : like(postId)),
    onMutate: (isLiked) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_liked: !isLiked,
        likes_count: (post.likes_count ?? 0) + (isLiked ? -1 : 1),
      }));
    },
    onError: (_err, isLiked) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_liked: isLiked,
        likes_count: (post.likes_count ?? 0) + (isLiked ? 1 : -1),
      }));
    },
    onSuccess: (data) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_liked: data.is_liked,
        likes_count: data.likes_count,
      }));
    },
  });
};
