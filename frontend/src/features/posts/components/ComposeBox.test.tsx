import { fireEvent, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeAll, beforeEach, describe, expect, test, vi } from 'vitest';
import { renderWithProviders } from '../../../test/renderWithProviders';
import { ComposeBox } from './ComposeBox';

const onSubmit = vi.fn();

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ currentUser: { id: 'u1', display_name: 'タロウ', avatar_url: null } }),
}));

const pngFile = (name: string, size = 10) => {
  const file = new File(['x'], name, { type: 'image/png' });
  Object.defineProperty(file, 'size', { value: size });
  return file;
};

const renderComposeBox = () =>
  renderWithProviders(
    <ComposeBox placeholder="いまどうしてる？" submitLabel="投稿" isPending={false} error={null} onSubmit={onSubmit} testIdPrefix="post-form" />,
  );

// accept 属性による絞り込みも検証の対象にするため、userEvent ではなく change イベントで選択する
const selectImages = (...files: File[]) => {
  fireEvent.change(screen.getByTestId('post-form-image-input'), { target: { files } });
};

describe('ComposeBox の画像添付', () => {
  beforeAll(() => {
    // jsdom には createObjectURL / revokeObjectURL がない
    URL.createObjectURL = vi.fn(() => 'blob:preview');
    URL.revokeObjectURL = vi.fn();
  });

  beforeEach(() => {
    onSubmit.mockReset();
  });

  test('本文と画像4枚を選ぶと、選んだ順にプレビューされ本文と一緒に送信される', async () => {
    const user = userEvent.setup();
    renderComposeBox();
    const files = [pngFile('1.png'), pngFile('2.png'), pngFile('3.png'), pngFile('4.png')];

    await user.type(screen.getByTestId('post-form-input'), '画像付き');
    selectImages(...files);

    expect(screen.getAllByRole('img')).toHaveLength(4);
    expect(screen.getByRole('button', { name: '画像を添付' })).toBeDisabled();

    await user.click(screen.getByTestId('post-form-submit'));

    expect(onSubmit.mock.calls[0][0]).toEqual({ content: '画像付き', media: files });
  });

  test('合計が4枚を超えるとエラーになり、選択は追加されない', () => {
    renderComposeBox();

    selectImages(pngFile('1.png'), pngFile('2.png'), pngFile('3.png'));
    selectImages(pngFile('4.png'), pngFile('5.png'));

    expect(screen.getByTestId('post-form-error')).toHaveTextContent('画像は4枚まで添付できます');
    expect(screen.getAllByRole('img')).toHaveLength(3);
  });

  test('対応していない形式の画像はエラーになる', () => {
    renderComposeBox();

    selectImages(new File(['x'], 'anim.gif', { type: 'image/gif' }));

    expect(screen.getByTestId('post-form-error')).toHaveTextContent('画像はJPEG/PNG/WebP形式、5MB以内にしてください');
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });

  test('5MBを超える画像はエラーになる', () => {
    renderComposeBox();

    selectImages(pngFile('large.png', 5 * 1024 * 1024 + 1));

    expect(screen.getByTestId('post-form-error')).toHaveTextContent('画像はJPEG/PNG/WebP形式、5MB以内にしてください');
  });

  test('画像があっても本文が空なら投稿ボタンは disabled のままになる', () => {
    renderComposeBox();

    selectImages(pngFile('1.png'));

    expect(screen.getByTestId('post-form-submit')).toBeDisabled();
  });
});
