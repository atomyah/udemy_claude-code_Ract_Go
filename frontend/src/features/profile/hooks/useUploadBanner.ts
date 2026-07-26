import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import type { UserResponse } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';
import { uploadBanner } from '../api';

export const useUploadBanner = () => {
  const queryClient = useQueryClient();
  const { currentUser, updateCurrentUser } = useAuth();

  return useMutation({
    mutationFn: (file: File) => uploadBanner(file),
    onSuccess: ({ banner_url }) => {
      if (!currentUser) return;
      const updated: UserResponse = { ...currentUser, banner_url };
      updateCurrentUser(updated);
      queryClient.setQueryData(queryKeys.users.profile(currentUser.handle ?? ''), updated);
    },
  });
};
