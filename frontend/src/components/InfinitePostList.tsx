import { Box, Typography } from '@mui/material';
import type { InfiniteData } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import type { PostListResponse } from '../api/types';
import { PostCard } from './PostCard';
import { PostSkeleton } from './PostCard/PostSkeleton';

interface InfinitePostListProps {
  data?: InfiniteData<PostListResponse>;
  isLoading: boolean;
  isError: boolean;
  isFetchingNextPage: boolean;
  hasNextPage?: boolean;
  fetchNextPage: () => void;
  emptyMessage?: string;
}

export const InfinitePostList = ({
  data,
  isLoading,
  isError,
  isFetchingNextPage,
  hasNextPage,
  fetchNextPage,
  emptyMessage = '投稿はありません',
}: InfinitePostListProps) => {
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { rootMargin: '200px' },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (isLoading) {
    return (
      <Box>
        {Array.from({ length: 3 }).map((_, index) => (
          <PostSkeleton key={index} />
        ))}
      </Box>
    );
  }

  if (isError) {
    return (
      <Typography color="error" sx={{ p: 2 }}>
        投稿の取得に失敗しました
      </Typography>
    );
  }

  const posts = data?.pages.flatMap((page) => page.data ?? []) ?? [];

  if (posts.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
        {emptyMessage}
      </Typography>
    );
  }

  return (
    <Box>
      {posts.map((post) => (
        <PostCard key={post.id} post={post} />
      ))}
      <div ref={sentinelRef} />
      {isFetchingNextPage && <PostSkeleton />}
      {!hasNextPage && (
        <Typography color="text.secondary" sx={{ p: 2, textAlign: 'center' }}>
          これ以上投稿はありません
        </Typography>
      )}
    </Box>
  );
};
