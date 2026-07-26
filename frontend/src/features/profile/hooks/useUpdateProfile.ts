import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import type { UpdateProfileRequest } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';
import { updateProfile } from '../api';

export const useUpdateProfile = () => {
  const queryClient = useQueryClient();
  const { updateCurrentUser } = useAuth();

  return useMutation({
    mutationFn: (payload: UpdateProfileRequest) => updateProfile(payload),
    onSuccess: (updated) => {
      updateCurrentUser(updated);
      queryClient.setQueryData(queryKeys.users.profile(updated.handle ?? ''), updated);
    },
  });
};
