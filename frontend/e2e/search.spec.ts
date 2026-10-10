import { expect, test } from '@playwright/test';
import { createPost, logout, makeUser, postCard, register, uniqueContent } from './helpers';

test.describe('検索・ハッシュタグ', () => {
  test('ハッシュタグをクリックするとハッシュタグページに遷移して関連投稿が表示される', async ({ page }) => {
    await register(page, makeUser('hashtag'));

    const tag = `tag${Date.now().toString(36)}`;
    const content = `${uniqueContent('ハッシュタグ投稿')} #${tag}`;
    await createPost(page, content);

    await postCard(page, content).getByTestId('hashtag-link').click();

    await expect(page).toHaveURL(new RegExp(`/hashtag/${tag}$`));
    await expect(page.getByTestId('hashtag-page')).toBeVisible();
    await expect(page.getByTestId('hashtag-title')).toHaveText(`#${tag}`);
    await expect(postCard(page, content)).toBeVisible();
  });

  test('使われていないハッシュタグのページでは投稿がない旨が表示される', async ({ page }) => {
    await register(page, makeUser('hashtag_empty'));

    await page.goto(`/hashtag/unused${Date.now().toString(36)}`);

    await expect(page.getByTestId('hashtag-page')).toBeVisible();
    await expect(page.getByTestId('post-list-empty')).toHaveText('このハッシュタグの投稿はありません');
  });

  test('検索バーに入力するとユーザー候補がドロップダウン表示され、クリックでプロフィールへ遷移する', async ({
    page,
  }) => {
    const target = makeUser('suggested');
    const searcher = makeUser('searcher');

    await register(page, target);
    await logout(page);
    await register(page, searcher);

    await page.goto('/explore');
    await page.getByTestId('explore-search-input').fill(target.handle);

    const suggestion = page.getByTestId('user-suggestion-item').filter({ hasText: `@${target.handle}` });
    await expect(suggestion).toBeVisible();

    await suggestion.click();

    await expect(page).toHaveURL(new RegExp(`/${target.handle}$`));
    await expect(page.getByTestId('profile-display-name')).toHaveText(target.displayName);
  });

  test('検索結果を投稿タブとユーザータブで切り替えられる', async ({ page }) => {
    const user = makeUser('searchtab');
    await register(page, user);

    const keyword = `kw${Date.now().toString(36)}`;
    const content = `${uniqueContent('検索対象の投稿')} ${keyword}`;
    await createPost(page, content);

    // 投稿タブ: 本文に一致する投稿が表示される
    await page.goto('/explore');
    await page.getByTestId('explore-search-input').fill(keyword);
    await expect(postCard(page, content)).toBeVisible();

    // ユーザータブ: ハンドルに一致するユーザーが表示される
    await page.getByTestId('explore-search-input').fill(user.handle);
    // サジェストがタブに重なるのでEscapeで閉じてから切り替える
    await page.getByTestId('explore-search-input').press('Escape');
    await expect(page.getByTestId('user-suggestions')).toBeHidden();
    await page.getByTestId('explore-tab-users').click();
    await expect(page.getByTestId('user-list-item').filter({ hasText: `@${user.handle}` })).toBeVisible();

    // 投稿タブに戻してキーワードで検索すると投稿の結果が表示される
    await page.getByTestId('explore-tab-posts').click();
    await page.getByTestId('explore-search-input').fill(keyword);
    await expect(postCard(page, content)).toBeVisible();
  });
});
