import type { InfiniteData, Query, QueryClient } from '@tanstack/react-query';
import type { PostListResponse, PostResponse } from '../../api/types';
import { queryKeys } from '../../api/queryKeys';

type InfinitePostsData = InfiniteData<PostListResponse>;

const isPostListQuery = (query: Query): boolean => {
  const key = query.queryKey[0];
  return key === 'posts' || key === 'bookmarks' || key === 'search';
};

export const updatePostInCaches = (
  queryClient: QueryClient,
  postId: string,
  updater: (post: PostResponse) => PostResponse,
): void => {
  queryClient.setQueryData<PostResponse>(queryKeys.posts.detail(postId), (old) => (old ? updater(old) : old));

  queryClient.setQueriesData<InfinitePostsData>({ predicate: isPostListQuery }, (old) => {
    if (!old?.pages) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        data: (page.data ?? []).map((post) => (post.id === postId ? updater(post) : post)),
      })),
    };
  });
};

export const removePostFromCaches = (queryClient: QueryClient, postId: string): void => {
  queryClient.removeQueries({ queryKey: queryKeys.posts.detail(postId) });

  queryClient.setQueriesData<InfinitePostsData>({ predicate: isPostListQuery }, (old) => {
    if (!old?.pages) return old;
    return {
      ...old,
      pages: old.pages.map((page) => ({
        ...page,
        data: (page.data ?? []).filter((post) => post.id !== postId),
      })),
    };
  });
};
