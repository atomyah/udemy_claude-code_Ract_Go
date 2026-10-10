import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import type { PostResponse } from '../../api/types';
import { renderWithProviders } from '../../test/renderWithProviders';
import { PostCard } from './PostCard';

const currentUser = { id: 'me', handle: 'me', display_name: 'わたし' };

vi.mock('../../features/auth/AuthContext', () => ({
  useAuth: () => ({ currentUser }),
}));

const likeMutate = vi.fn();
const bookmarkMutate = vi.fn();
const deleteMutate = vi.fn();

vi.mock('../../features/posts/hooks/useLike', () => ({
  useLike: () => ({ mutate: likeMutate, isPending: false }),
}));
vi.mock('../../features/posts/hooks/useRepost', () => ({
  useRepost: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock('../../features/posts/hooks/useBookmark', () => ({
  useBookmark: () => ({ mutate: bookmarkMutate, isPending: false }),
}));
vi.mock('../../features/posts/hooks/useDeletePost', () => ({
  useDeletePost: () => ({ mutate: deleteMutate, isPending: false }),
}));

const makePost = (overrides: Partial<PostResponse> = {}): PostResponse =>
  ({
    id: 'p1',
    content: 'テスト投稿',
    created_at: new Date().toISOString(),
    likes_count: 0,
    comments_count: 0,
    reposts_count: 0,
    is_liked: false,
    is_bookmarked: false,
    is_reposted: false,
    user: { id: 'other', handle: 'other', display_name: 'ほかのひと', avatar_url: null },
    media: [],
    ...overrides,
  }) as PostResponse;

describe('PostCard', () => {
  beforeEach(() => {
    likeMutate.mockReset();
    bookmarkMutate.mockReset();
    deleteMutate.mockReset();
  });

  test('自分の投稿のメニューには編集と削除が表示される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostCard post={makePost({ user: { ...currentUser } })} />);

    await user.click(screen.getByTestId('post-menu-button'));

    expect(screen.getByTestId('post-menu-edit')).toBeInTheDocument();
    expect(screen.getByTestId('post-menu-delete')).toBeInTheDocument();
  });

  test('他人の投稿のメニューには編集と削除が表示されない', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostCard post={makePost()} />);

    await user.click(screen.getByTestId('post-menu-button'));

    expect(screen.queryByTestId('post-menu-edit')).not.toBeInTheDocument();
    expect(screen.queryByTestId('post-menu-delete')).not.toBeInTheDocument();
    expect(screen.getByText('報告')).toBeInTheDocument();
  });

  test('削除を選ぶと削除処理が投稿IDつきで呼ばれる', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostCard post={makePost({ user: { ...currentUser } })} />);

    await user.click(screen.getByTestId('post-menu-button'));
    await user.click(screen.getByTestId('post-menu-delete'));

    expect(deleteMutate).toHaveBeenCalledWith('p1');
  });

  test('いいねボタンを押すと現在のいいね状態が渡される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostCard post={makePost({ is_liked: false })} />);

    await user.click(screen.getByTestId('post-like-button'));

    expect(likeMutate).toHaveBeenCalledWith(false);
  });

  test('いいね済みの投稿はボタンがアクティブ表示になる', () => {
    renderWithProviders(<PostCard post={makePost({ is_liked: true, likes_count: 3 })} />);

    expect(screen.getByTestId('post-like-button')).toHaveAttribute('data-active', 'true');
    expect(screen.getByTestId('post-like-count')).toHaveTextContent('3');
  });

  test('ブックマークボタンを押すと現在のブックマーク状態が渡される', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PostCard post={makePost({ is_bookmarked: true })} />);

    await user.click(screen.getByTestId('post-bookmark-button'));

    expect(bookmarkMutate).toHaveBeenCalledWith(true);
  });
});
