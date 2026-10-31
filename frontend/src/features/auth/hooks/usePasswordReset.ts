import { useMutation } from '@tanstack/react-query';
import type { PasswordResetRequest, PasswordResetRequestRequest } from '../../../api/types';
import { requestPasswordReset, resetPassword } from '../api';

/** パスワードリセットを申請する */
export const useRequestPasswordReset = () =>
  useMutation({
    mutationFn: (payload: PasswordResetRequestRequest) => requestPasswordReset(payload),
  });

/** リセットリンクのトークンで新しいパスワードを設定する */
export const useResetPassword = () =>
  useMutation({
    mutationFn: (payload: PasswordResetRequest) => resetPassword(payload),
  });
