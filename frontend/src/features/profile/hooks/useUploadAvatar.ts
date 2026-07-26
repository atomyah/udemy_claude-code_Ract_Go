import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../../api/queryKeys';
import type { UserResponse } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';
import { uploadAvatar } from '../api';

export const useUploadAvatar = () => {
  const queryClient = useQueryClient();
  const { currentUser, updateCurrentUser } = useAuth();

  return useMutation({
    mutationFn: (file: File) => uploadAvatar(file),
    onSuccess: ({ avatar_url }) => {
      if (!currentUser) return;
      const updated: UserResponse = { ...currentUser, avatar_url };
      updateCurrentUser(updated);
      queryClient.setQueryData(queryKeys.users.profile(currentUser.handle ?? ''), updated);
    },
  });
};
