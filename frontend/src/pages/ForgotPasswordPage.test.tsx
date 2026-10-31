import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { renderWithProviders } from '../test/renderWithProviders';
import ForgotPasswordPage from './ForgotPasswordPage';

const mutateMock = vi.fn();
let mutationState = {
  isPending: false,
  isSuccess: false,
  data: undefined as { message: string } | undefined,
  error: null as unknown,
};

vi.mock('../features/auth/hooks/usePasswordReset', () => ({
  useRequestPasswordReset: () => ({ mutate: mutateMock, ...mutationState }),
}));

describe('ForgotPasswordPage', () => {
  beforeEach(() => {
    mutateMock.mockReset();
    mutationState = { isPending: false, isSuccess: false, data: undefined, error: null };
  });

  test('メールアドレスを入力して申請すると送信される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ForgotPasswordPage />);

    await user.type(screen.getByTestId('forgot-password-email'), 'taro@example.com');
    await user.click(screen.getByTestId('forgot-password-submit'));

    expect(mutateMock).toHaveBeenCalledTimes(1);
    expect(mutateMock.mock.calls[0][0]).toEqual({ email: 'taro@example.com' });
  });

  test('申請中は申請ボタンが disabled になる', () => {
    mutationState = { ...mutationState, isPending: true };
    renderWithProviders(<ForgotPasswordPage />);

    expect(screen.getByTestId('forgot-password-submit')).toBeDisabled();
  });

  test('申請が完了すると受付メッセージを表示しフォームを隠す', () => {
    mutationState = { ...mutationState, isSuccess: true, data: { message: '申請を受け付けました' } };
    renderWithProviders(<ForgotPasswordPage />);

    expect(screen.getByTestId('forgot-password-success')).toHaveTextContent('申請を受け付けました');
    expect(screen.queryByTestId('forgot-password-submit')).not.toBeInTheDocument();
  });
});
