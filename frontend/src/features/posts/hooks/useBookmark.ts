import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { bookmark, unbookmark } from '../api';
import { updatePostInCaches } from '../cacheHelpers';

export const useBookmark = (postId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (isBookmarked: boolean) => (isBookmarked ? unbookmark(postId) : bookmark(postId)),
    onMutate: (isBookmarked) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_bookmarked: !isBookmarked,
      }));
    },
    onError: (_err, isBookmarked) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_bookmarked: isBookmarked,
      }));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bookmarks.list() });
    },
  });
};
