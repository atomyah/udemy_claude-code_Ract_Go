import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getProfile } from '../api';

export const useProfile = (handle: string | undefined) =>
  useQuery({
    queryKey: queryKeys.users.profile(handle ?? ''),
    queryFn: () => getProfile(handle ?? ''),
    enabled: !!handle,
  });
