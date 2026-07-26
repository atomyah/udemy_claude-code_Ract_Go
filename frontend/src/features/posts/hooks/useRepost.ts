import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { repost, unrepost } from '../api';
import { updatePostInCaches } from '../cacheHelpers';

interface RepostInput {
  isReposted: boolean;
  content?: string;
  media?: File[];
}

export const useRepost = (postId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ isReposted, content, media }: RepostInput) =>
      isReposted ? unrepost(postId) : repost(postId, { content: content ?? '', media }),
    onMutate: ({ isReposted }) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_reposted: !isReposted,
        reposts_count: (post.reposts_count ?? 0) + (isReposted ? -1 : 1),
      }));
    },
    onError: (_err, { isReposted }) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_reposted: isReposted,
        reposts_count: (post.reposts_count ?? 0) + (isReposted ? 1 : -1),
      }));
    },
    onSuccess: (data) => {
      updatePostInCaches(queryClient, postId, (post) => ({
        ...post,
        is_reposted: data.is_reposted,
        reposts_count: data.reposts_count,
      }));
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.home() });
      queryClient.invalidateQueries({ queryKey: queryKeys.posts.explore() });
    },
  });
};
