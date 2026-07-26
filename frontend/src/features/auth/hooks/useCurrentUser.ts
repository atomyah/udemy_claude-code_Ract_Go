import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import { getMe } from '../api';
import { useAuth } from '../AuthContext';

export const useCurrentUser = () => {
  const { isAuthenticated } = useAuth();

  return useQuery({
    queryKey: queryKeys.auth.me(),
    queryFn: getMe,
    enabled: isAuthenticated,
  });
};
