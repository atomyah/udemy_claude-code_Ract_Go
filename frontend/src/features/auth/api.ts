import { api } from '../../api/client';
import type {
  AuthResponse,
  GoogleLoginRequest,
  LoginRequest,
  PasswordResetRequest,
  PasswordResetRequestRequest,
  RegisterRequest,
  SuccessResponse,
  UserResponse,
} from '../../api/types';

export const login = (payload: LoginRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/login', payload).then((res) => res.data);

export const loginWithGoogle = (payload: GoogleLoginRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/google', payload).then((res) => res.data);

export const register = (payload: RegisterRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/register', payload).then((res) => res.data);

/** パスワードリセットを申請する（登録の有無にかかわらず同じ応答が返る） */
export const requestPasswordReset = (payload: PasswordResetRequestRequest): Promise<SuccessResponse> =>
  api.post<SuccessResponse>('/auth/password-reset-requests', payload).then((res) => res.data);

/** 運営から届いたリセットリンクのトークンで新しいパスワードを設定する */
export const resetPassword = (payload: PasswordResetRequest): Promise<SuccessResponse> =>
  api.post<SuccessResponse>('/auth/password-reset', payload).then((res) => res.data);

export const logout = (): Promise<void> => api.post('/auth/logout').then(() => undefined);

export const getMe = (): Promise<UserResponse> =>
  api.get<UserResponse>('/users/me').then((res) => res.data);
