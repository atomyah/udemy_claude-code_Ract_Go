import { expect, test } from '@playwright/test';
import { login, logout, makeUser, register } from './helpers';

test.describe('認証', () => {
  test('ユーザー登録してからログインできる', async ({ page }) => {
    const user = makeUser('auth');

    await register(page, user);
    await logout(page);
    await login(page, user);

    // ログイン後は自分のプロフィールが表示できる
    await page.goto(`/${user.handle}`);
    await expect(page.getByTestId('profile-handle')).toHaveText(`@${user.handle}`);
    await expect(page.getByTestId('profile-display-name')).toHaveText(user.displayName);
  });

  test('誤ったパスワードではログインできずエラーメッセージが日本語で表示される', async ({ page }) => {
    const user = makeUser('auth_ng');
    await register(page, user);
    await logout(page);

    await page.getByTestId('login-email').fill(user.email);
    await page.getByTestId('login-password').fill('wrong_password_123');
    await page.getByTestId('login-submit').click();

    await expect(page.getByTestId('login-error')).toHaveText('メールアドレスまたはパスワードが正しくありません');
    await expect(page).toHaveURL(/\/login$/);
  });

  test('ログアウト後に認証が必要なページへアクセスするとログインページにリダイレクトされる', async ({ page }) => {
    const user = makeUser('auth_guard');
    await register(page, user);
    await logout(page);

    await page.goto('/bookmarks');

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByTestId('login-page')).toBeVisible();
  });

  test('未ログイン状態で設定ページへ直接アクセスするとログインページにリダイレクトされる', async ({ page }) => {
    await page.goto('/settings');

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByTestId('login-page')).toBeVisible();
  });
});
