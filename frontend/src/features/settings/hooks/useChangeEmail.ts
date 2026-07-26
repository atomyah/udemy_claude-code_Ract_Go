import { useMutation } from '@tanstack/react-query';
import type { ChangeEmailRequest } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';
import { changeEmail } from '../api';

export const useChangeEmail = () => {
  const { currentUser, updateCurrentUser } = useAuth();

  return useMutation({
    mutationFn: (payload: ChangeEmailRequest) => changeEmail(payload),
    onSuccess: (_result, variables) => {
      if (currentUser) {
        updateCurrentUser({ ...currentUser, email: variables.new_email });
      }
    },
  });
};
