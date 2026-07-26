import { useMutation } from '@tanstack/react-query';
import type { ChangePasswordRequest } from '../../../api/types';
import { changePassword } from '../api';

export const useChangePassword = () =>
  useMutation({
    mutationFn: (payload: ChangePasswordRequest) => changePassword(payload),
  });
