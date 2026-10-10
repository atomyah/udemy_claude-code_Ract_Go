import { expect, test } from '@playwright/test';
import { makeUser, register } from './helpers';

test.describe('管理者画面の認可', () => {
  test('一般ユーザーが管理者ページを開くとホームにリダイレクトされる', async ({ page }) => {
    await register(page, makeUser('admin_guard'));

    await page.goto('/admin');

    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByTestId('post-form')).toBeVisible();
  });

  test('未ログインで管理者ページを開くとログインページにリダイレクトされる', async ({ page }) => {
    await page.goto('/admin');

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByTestId('login-submit')).toBeVisible();
  });
});
