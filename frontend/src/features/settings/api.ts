import { api } from '../../api/client';
import type { ChangeEmailRequest, ChangePasswordRequest, SuccessResponse } from '../../api/types';

export const changeEmail = (payload: ChangeEmailRequest): Promise<SuccessResponse> =>
  api.put<SuccessResponse>('/users/me/email', payload).then((res) => res.data);

export const changePassword = (payload: ChangePasswordRequest): Promise<SuccessResponse> =>
  api.put<SuccessResponse>('/users/me/password', payload).then((res) => res.data);
