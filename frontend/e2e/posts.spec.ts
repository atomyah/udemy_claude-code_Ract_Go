import { expect, test } from '@playwright/test';
import { createPost, makeUser, openPostMenu, postCard, register, uniqueContent } from './helpers';

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
});
