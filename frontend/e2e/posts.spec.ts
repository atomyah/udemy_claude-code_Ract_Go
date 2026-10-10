import { expect, test } from '@playwright/test';
import { createPost, logout, makeUser, openPostMenu, postCard, register, uniqueContent } from './helpers';

test.describe('投稿', () => {
  test('投稿するとタイムラインに表示される', async ({ page }) => {
    await register(page, makeUser('post'));
    const content = uniqueContent('タイムライン投稿');

    await createPost(page, content);

    await expect(postCard(page, content)).toBeVisible();
    await expect(postCard(page, content).getByTestId('post-content')).toHaveText(content);

    // リロード後もサーバーから取得した投稿が表示される
    await page.reload();
    await expect(postCard(page, content)).toBeVisible();
  });

  test('投稿を編集できる', async ({ page }) => {
    await register(page, makeUser('post_edit'));
    const content = uniqueContent('編集前の投稿');
    const editedContent = uniqueContent('編集後の投稿');
    await createPost(page, content);

    await openPostMenu(postCard(page, content));
    await page.getByTestId('post-menu-edit').click();
    await expect(page.getByTestId('post-edit-dialog')).toBeVisible();

    await page.getByTestId('post-edit-input').fill(editedContent);
    await page.getByTestId('post-edit-save').click();

    await expect(postCard(page, editedContent)).toBeVisible();
    await expect(postCard(page, editedContent)).toContainText('（編集済み）');
    await expect(page.getByTestId('post-card').filter({ hasText: content })).toHaveCount(0);
  });

  test('投稿を削除できる', async ({ page }) => {
    await register(page, makeUser('post_delete'));
    const content = uniqueContent('削除する投稿');
    await createPost(page, content);

    await openPostMenu(postCard(page, content));
    await page.getByTestId('post-menu-delete').click();

    await expect(page.getByTestId('toast')).toHaveText('投稿を削除しました');
    await expect(page.getByTestId('post-card').filter({ hasText: content })).toHaveCount(0);

    // リロードしてもサーバー側で削除されている
    await page.reload();
    await expect(page.getByTestId('post-card').filter({ hasText: content })).toHaveCount(0);
  });

  test('いいねといいね解除ができる', async ({ page }) => {
    await register(page, makeUser('post_like'));
    const content = uniqueContent('いいねされる投稿');
    await createPost(page, content);

    const card = postCard(page, content);
    await expect(card.getByTestId('post-like-count')).toHaveText('');

    await card.getByTestId('post-like-button').click();
    await expect(card.getByTestId('post-like-count')).toHaveText('1');
    await expect(card.getByTestId('post-like-button')).toHaveAttribute('data-active', 'true');

    await card.getByTestId('post-like-button').click();
    await expect(card.getByTestId('post-like-count')).toHaveText('');
    await expect(card.getByTestId('post-like-button')).toHaveAttribute('data-active', 'false');
  });

  test('本文が空のときは投稿ボタンが押せない', async ({ page }) => {
    await register(page, makeUser('post_empty'));

    await expect(page.getByTestId('post-form-submit')).toBeDisabled();

    // 空白だけでも投稿できない
    await page.getByTestId('post-form-input').fill('   ');
    await expect(page.getByTestId('post-form-submit')).toBeDisabled();

    // 1文字入れると押せるようになる
    await page.getByTestId('post-form-input').fill('あ');
    await expect(page.getByTestId('post-form-submit')).toBeEnabled();
  });

  test('280文字ちょうどの投稿ができ、281文字目は入力できない', async ({ page }) => {
    await register(page, makeUser('post_max'));

    // 他のテストの投稿と区別できるよう末尾に一意な文字列を入れて280文字にする
    const marker = uniqueContent('max');
    const content = 'あ'.repeat(280 - marker.length) + marker;
    expect(content).toHaveLength(280);

    const input = page.getByTestId('post-form-input');
    // 281文字入力しても280文字で打ち切られる
    await input.fill(content + 'X');
    await expect(input).toHaveValue(content);

    await page.getByTestId('post-form-submit').click();
    await expect(postCard(page, marker)).toBeVisible();

    // サーバーにも280文字が保存されている
    await page.reload();
    await expect(postCard(page, marker).getByTestId('post-content')).toHaveText(content);
  });

  test('他人の投稿には編集・削除メニューが表示されない', async ({ page }) => {
    const author = makeUser('post_author');
    const viewer = makeUser('post_viewer');

    await register(page, author);
    const content = uniqueContent('他人の投稿');
    await createPost(page, content);
    await page.goto('/');
    await logout(page);

    await register(page, viewer);
    await page.goto('/explore');

    const card = postCard(page, content);
    await expect(card).toBeVisible();
    await openPostMenu(card);

    await expect(page.getByTestId('post-menu-edit')).toHaveCount(0);
    await expect(page.getByTestId('post-menu-delete')).toHaveCount(0);
  });
});
