import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import type { UserResponse } from '../../api/types';
import { AuthProvider, useAuth } from './AuthContext';

const getMeMock = vi.fn();
const loginMock = vi.fn();
const logoutMock = vi.fn();

vi.mock('./api', () => ({
  getMe: () => getMeMock(),
  login: (payload: unknown) => loginMock(payload),
  logout: () => logoutMock(),
  register: vi.fn(),
}));

const setAccessTokenMock = vi.fn();
const clearAccessTokenMock = vi.fn();

vi.mock('../../api/client', () => ({
  setAccessToken: (token: string) => setAccessTokenMock(token),
  clearAccessToken: () => clearAccessTokenMock(),
  setSessionExpiredHandler: vi.fn(),
}));

const user: UserResponse = { id: 'u1', handle: 'taro', display_name: 'タロウ' } as UserResponse;

const wrapper = ({ children }: { children: ReactNode }) => <AuthProvider>{children}</AuthProvider>;

describe('AuthContext', () => {
  beforeEach(() => {
    getMeMock.mockReset();
    loginMock.mockReset();
    logoutMock.mockReset();
    setAccessTokenMock.mockReset();
    clearAccessTokenMock.mockReset();
  });

  test('セッションが無いときは未ログイン状態になる', async () => {
    getMeMock.mockRejectedValue(new Error('unauthorized'));

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.currentUser).toBeNull();
  });

  test('既存セッションがあるとログイン済み状態で初期化される', async () => {
    getMeMock.mockResolvedValue(user);

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.currentUser?.handle).toBe('taro');
  });

  test('ログインするとトークンが保存されログイン済み状態になる', async () => {
    getMeMock.mockRejectedValue(new Error('unauthorized'));
    loginMock.mockResolvedValue({ access_token: 'token-123', user });

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.login('taro@example.com', 'password123');
    });

    expect(loginMock).toHaveBeenCalledWith({ email: 'taro@example.com', password: 'password123' });
    expect(setAccessTokenMock).toHaveBeenCalledWith('token-123');
    expect(result.current.isAuthenticated).toBe(true);
  });

  test('ログアウトするとトークンが削除され未ログイン状態に戻る', async () => {
    getMeMock.mockResolvedValue(user);
    logoutMock.mockResolvedValue(undefined);

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout();
    });

    expect(clearAccessTokenMock).toHaveBeenCalled();
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.currentUser).toBeNull();
  });

  test('ログアウトAPIが失敗してもクライアント側はログアウト状態にする', async () => {
    getMeMock.mockResolvedValue(user);
    logoutMock.mockRejectedValue(new Error('network error'));

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout().catch(() => undefined);
    });

    expect(clearAccessTokenMock).toHaveBeenCalled();
    expect(result.current.isAuthenticated).toBe(false);
  });
});
