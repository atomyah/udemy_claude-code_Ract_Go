import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getHashtagPosts } from '../api';

export const useHashtagPosts = (tag: string) =>
  useInfiniteQuery({
    queryKey: queryKeys.search.hashtag(tag),
    queryFn: ({ pageParam }) => getHashtagPosts(tag, { cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: tag.length > 0,
    // 存在しないハッシュタグは404が返るためリトライしない
    retry: false,
  });
