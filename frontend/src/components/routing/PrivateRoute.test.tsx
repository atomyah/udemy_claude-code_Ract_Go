import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { PrivateRoute } from './PrivateRoute';

const useAuthMock = vi.fn();

vi.mock('../../features/auth/AuthContext', () => ({
  useAuth: () => useAuthMock(),
}));

const renderRoutes = () =>
  render(
    <MemoryRouter initialEntries={['/settings']}>
      <Routes>
        <Route element={<PrivateRoute />}>
          <Route path="/settings" element={<div>設定ページ</div>} />
        </Route>
        <Route path="/login" element={<div>ログインページ</div>} />
      </Routes>
    </MemoryRouter>,
  );

describe('PrivateRoute', () => {
  beforeEach(() => {
    useAuthMock.mockReset();
  });

  test('未ログインのときログインページにリダイレクトされる', () => {
    useAuthMock.mockReturnValue({ isAuthenticated: false, isLoading: false });

    renderRoutes();

    expect(screen.getByText('ログインページ')).toBeInTheDocument();
    expect(screen.queryByText('設定ページ')).not.toBeInTheDocument();
  });

  test('ログイン済みのとき子ルートが表示される', () => {
    useAuthMock.mockReturnValue({ isAuthenticated: true, isLoading: false });

    renderRoutes();

    expect(screen.getByText('設定ページ')).toBeInTheDocument();
    expect(screen.queryByText('ログインページ')).not.toBeInTheDocument();
  });

  test('認証状態の判定中はリダイレクトせずローディングを表示する', () => {
    useAuthMock.mockReturnValue({ isAuthenticated: false, isLoading: true });

    renderRoutes();

    // 判定中にログインページへ飛ばすとリロード時に一瞬ログイン画面が出てしまう
    expect(screen.queryByText('ログインページ')).not.toBeInTheDocument();
    expect(screen.queryByText('設定ページ')).not.toBeInTheDocument();
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });
});
