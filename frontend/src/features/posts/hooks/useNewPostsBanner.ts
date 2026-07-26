import { useQuery, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { useAuth } from '../../auth/AuthContext';
import { getHomeTimeline } from '../api';

export const useNewPostsBanner = () => {
  const { isAuthenticated } = useAuth();
  const queryClient = useQueryClient();

  const { data: latestId } = useQuery({
    queryKey: queryKeys.posts.homeLatest(),
    queryFn: async () => {
      const result = await getHomeTimeline({ limit: 1 });
      return result.data?.[0]?.id ?? null;
    },
    enabled: isAuthenticated,
    refetchInterval: 30_000,
  });

  const cachedFirstId = queryClient
    .getQueryData<{ pages: { data: { id: string }[] }[] }>(queryKeys.posts.home())
    ?.pages[0]?.data[0]?.id;

  const hasNewPosts = Boolean(latestId && cachedFirstId && latestId !== cachedFirstId);

  const showLatest = (): void => {
    queryClient.invalidateQueries({ queryKey: queryKeys.posts.home() });
  };

  return { hasNewPosts, showLatest };
};
