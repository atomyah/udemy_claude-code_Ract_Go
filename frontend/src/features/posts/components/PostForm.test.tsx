import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { renderWithProviders } from '../../../test/renderWithProviders';
import { PostForm } from './PostForm';

const mutateMock = vi.fn();

vi.mock('../hooks/useCreatePost', () => ({
  useCreatePost: () => ({ mutate: mutateMock, isPending: false, error: null }),
}));

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ currentUser: { id: 'u1', display_name: 'タロウ', avatar_url: null } }),
}));

describe('PostForm', () => {
  beforeEach(() => {
    mutateMock.mockReset();
  });

  test('本文が空のときは投稿ボタンが disabled になる', () => {
    renderWithProviders(<PostForm />);

    expect(screen.getByTestId('post-form-submit')).toBeDisabled();
  });

  test('空白のみのときも投稿ボタンが disabled のままになる', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostForm />);

    await user.type(screen.getByTestId('post-form-input'), '   ');

    expect(screen.getByTestId('post-form-submit')).toBeDisabled();
  });

  test('本文を入力すると投稿ボタンが押せるようになり、入力内容が送信される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostForm />);

    await user.type(screen.getByTestId('post-form-input'), 'こんにちは');
    const submit = screen.getByTestId('post-form-submit');
    expect(submit).toBeEnabled();

    await user.click(submit);

    expect(mutateMock).toHaveBeenCalledTimes(1);
    expect(mutateMock.mock.calls[0][0]).toEqual({ content: 'こんにちは', media: [] });
  });

  test('280文字を超える入力は280文字で打ち切られ、文字数カウンターに反映される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostForm />);

    const input = screen.getByTestId('post-form-input');
    await user.click(input);
    // userEvent.type は1文字ずつ入力するため、長文は paste で入れる
    await user.paste('あ'.repeat(285));

    expect(input).toHaveValue('あ'.repeat(280));
    expect(screen.getByText('280 / 280')).toBeInTheDocument();
  });
});
