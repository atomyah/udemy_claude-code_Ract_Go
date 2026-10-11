import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import type { ReactNode } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
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

// client.ts に登録されたセッション切れハンドラーを取り出して、テストから呼べるようにする
let sessionExpiredHandler: (() => void) | null = null;

vi.mock('../../api/client', () => ({
  setSessionExpiredHandler: (handler: () => void) => {
    sessionExpiredHandler = handler;
  },
}));

const user: UserResponse = { id: 'u1', handle: 'taro', display_name: 'タロウ' } as UserResponse;

const wrapper = ({ children }: { children: ReactNode }) => (
  <MemoryRouter initialEntries={['/bookmarks']}>
    <AuthProvider>{children}</AuthProvider>
  </MemoryRouter>
);

/** useAuth と現在のパスを同時に取得する */
const useAuthWithLocation = () => ({ auth: useAuth(), location: useLocation() });

describe('AuthContext', () => {
  beforeEach(() => {
    getMeMock.mockReset();
    loginMock.mockReset();
    logoutMock.mockReset();
    sessionExpiredHandler = null;
    localStorage.clear();
  });

  test('セッションが無いときは未ログイン状態になる', async () => {
    getMeMock.mockRejectedValue(new Error('unauthorized'));

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isAuthenticated).toBe(false);
    expect(result.current.currentUser).toBeNull();
  });

  test('既存セッション（Cookie）があるとログイン済み状態で初期化される', async () => {
    getMeMock.mockResolvedValue(user);

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.currentUser?.handle).toBe('taro');
  });

  test('ログインするとログイン済み状態になり、トークンを localStorage に保存しない', async () => {
    getMeMock.mockRejectedValue(new Error('unauthorized'));
    loginMock.mockResolvedValue({ user });

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.login('taro@example.com', 'password123');
    });

    expect(loginMock).toHaveBeenCalledWith({ email: 'taro@example.com', password: 'password123' });
    expect(result.current.isAuthenticated).toBe(true);
    expect(localStorage.length).toBe(0);
  });

  test('ログアウトすると未ログイン状態に戻る', async () => {
    getMeMock.mockResolvedValue(user);
    logoutMock.mockResolvedValue(undefined);

    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));

    await act(async () => {
      await result.current.logout();
    });

    expect(logoutMock).toHaveBeenCalled();
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

    expect(result.current.isAuthenticated).toBe(false);
  });

  test('ログイン中にリフレッシュも失敗するとログアウト状態になりログイン画面へ遷移する', async () => {
    getMeMock.mockResolvedValue(user);

    const { result } = renderHook(() => useAuthWithLocation(), { wrapper });
    await waitFor(() => expect(result.current.auth.isAuthenticated).toBe(true));
    expect(result.current.location.pathname).toBe('/bookmarks');

    act(() => {
      sessionExpiredHandler?.();
    });

    expect(result.current.auth.isAuthenticated).toBe(false);
    expect(result.current.location.pathname).toBe('/login');
  });

  test('未ログインのままセッション切れになっても画面遷移しない（公開ページの閲覧を妨げない）', async () => {
    getMeMock.mockRejectedValue(new Error('unauthorized'));

    const { result } = renderHook(() => useAuthWithLocation(), { wrapper });
    await waitFor(() => expect(result.current.auth.isLoading).toBe(false));

    act(() => {
      sessionExpiredHandler?.();
    });

    expect(result.current.location.pathname).toBe('/bookmarks');
  });
});
