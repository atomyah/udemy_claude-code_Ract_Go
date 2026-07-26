import { api } from '../../api/client';
import type { AuthResponse, GoogleLoginRequest, LoginRequest, RegisterRequest, UserResponse } from '../../api/types';

export const login = (payload: LoginRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/login', payload).then((res) => res.data);

export const loginWithGoogle = (payload: GoogleLoginRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/google', payload).then((res) => res.data);

export const register = (payload: RegisterRequest): Promise<AuthResponse> =>
  api.post<AuthResponse>('/auth/register', payload).then((res) => res.data);

export const logout = (): Promise<void> => api.post('/auth/logout').then(() => undefined);

export const getMe = (): Promise<UserResponse> =>
  api.get<UserResponse>('/users/me').then((res) => res.data);
