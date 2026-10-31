import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { renderWithProviders } from '../test/renderWithProviders';
import ResetPasswordPage from './ResetPasswordPage';

const mutateMock = vi.fn();
let mutationState = { isPending: false, isSuccess: false, error: null as unknown };

vi.mock('../features/auth/hooks/usePasswordReset', () => ({
  useResetPassword: () => ({ mutate: mutateMock, ...mutationState }),
}));

describe('ResetPasswordPage', () => {
  beforeEach(() => {
    mutateMock.mockReset();
    mutationState = { isPending: false, isSuccess: false, error: null };
  });

  test('URLにトークンがないときはエラーを表示しフォームを出さない', () => {
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password' });

    expect(screen.getByTestId('reset-password-missing-token')).toBeInTheDocument();
    expect(screen.queryByTestId('reset-password-submit')).not.toBeInTheDocument();
  });

  test('確認用パスワードが一致しないときは送信しない', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=abc' });

    await user.type(screen.getByTestId('reset-password-password'), 'newpassword123');
    await user.type(screen.getByTestId('reset-password-confirm'), 'different123');
    await user.click(screen.getByTestId('reset-password-submit'));

    expect(await screen.findByText('パスワードが一致しません')).toBeInTheDocument();
    expect(mutateMock).not.toHaveBeenCalled();
  });

  test('トークンと新しいパスワードを送信する', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=abc' });

    await user.type(screen.getByTestId('reset-password-password'), 'newpassword123');
    await user.type(screen.getByTestId('reset-password-confirm'), 'newpassword123');
    await user.click(screen.getByTestId('reset-password-submit'));

    expect(mutateMock).toHaveBeenCalledTimes(1);
    expect(mutateMock.mock.calls[0][0]).toEqual({ token: 'abc', password: 'newpassword123' });
  });

  test('再設定に成功すると完了メッセージを表示する', () => {
    mutationState = { isPending: false, isSuccess: true, error: null };
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=abc' });

    expect(screen.getByTestId('reset-password-success')).toBeInTheDocument();
  });
});
