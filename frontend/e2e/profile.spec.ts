import { expect, test } from '@playwright/test';
import { logout, makeUser, register, uniqueContent } from './helpers';

test.describe('プロフィール・フォロー', () => {
  test('フォローと解除がフォロー一覧に反映される', async ({ page }) => {
    const target = makeUser('followee');
    const follower = makeUser('follower');

    await register(page, target);
    await logout(page);
    await register(page, follower);

    // 相手のプロフィールでフォローする
    await page.goto(`/${target.handle}`);
    const followButton = page.getByTestId('profile-follow-button');
    await expect(followButton).toHaveAttribute('data-following', 'false');

    await followButton.click();
    await expect(followButton).toHaveAttribute('data-following', 'true');
    await expect(followButton).toHaveText('フォロー中');

    // 自分のプロフィールの「フォロー中」一覧に相手が表示される
    await page.goto(`/${follower.handle}`);
    await expect(page.getByTestId('profile-following-count')).toContainText('1');
    await page.getByTestId('profile-following-count').click();
    await expect(page.getByTestId('follow-list-dialog')).toHaveText('フォロー中');
    await expect(page.getByTestId('user-list-item').filter({ hasText: `@${target.handle}` })).toBeVisible();
    await page.keyboard.press('Escape');

    // フォローを解除すると一覧から消える
    await page.goto(`/${target.handle}`);
    await page.getByTestId('profile-follow-button').click();
    await expect(page.getByTestId('profile-follow-button')).toHaveAttribute('data-following', 'false');
    await expect(page.getByTestId('profile-follow-button')).toHaveText('フォロー');

    await page.goto(`/${follower.handle}`);
    await expect(page.getByTestId('profile-following-count')).toContainText('0');
    await page.getByTestId('profile-following-count').click();
    await expect(page.getByTestId('follow-list-empty')).toBeVisible();
  });

  test('プロフィールを編集できる', async ({ page }) => {
    const user = makeUser('profile_edit');
    await register(page, user);

    const newDisplayName = uniqueContent('新しい表示名').slice(0, 50);
    const newBio = uniqueContent('新しい自己紹介');

    await page.goto(`/${user.handle}`);
    await page.getByTestId('profile-edit-button').click();
    await expect(page.getByTestId('edit-profile-dialog')).toBeVisible();

    await page.getByTestId('edit-profile-display-name').fill(newDisplayName);
    await page.getByTestId('edit-profile-bio').fill(newBio);
    await page.getByTestId('edit-profile-location').fill('東京');
    await page.getByTestId('edit-profile-save').click();

    await expect(page.getByTestId('edit-profile-dialog')).toBeHidden();
    await expect(page.getByTestId('profile-display-name')).toHaveText(newDisplayName);
    await expect(page.getByTestId('profile-bio')).toHaveText(newBio);

    // リロードしてもサーバーに保存された内容が表示される
    await page.reload();
    await expect(page.getByTestId('profile-display-name')).toHaveText(newDisplayName);
    await expect(page.getByTestId('profile-bio')).toHaveText(newBio);
  });
});
