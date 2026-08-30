import { expect, type Locator, type Page } from '@playwright/test';

/**
 * E2Eテスト用のユーザー情報。
 * バックエンドのバリデーション（dto.RegisterRequest）を満たす値を生成する:
 * - email: email形式
 * - password: 8〜72文字
 * - handle: 3〜50文字（フロントエンドの制約により英数字とアンダースコアのみ）
 * - display_name: 1〜50文字
 */
export interface TestUser {
  email: string;
  password: string;
  handle: string;
  displayName: string;
}

let sequence = 0;

/** 実行ごとに一意なテストユーザーを作る */
export const makeUser = (prefix: string): TestUser => {
  sequence += 1;
  const suffix = `${Date.now().toString(36)}_${sequence}`;
  const handle = `${prefix}_${suffix}`.slice(0, 50);
  return {
    email: `${handle}@example.com`,
    password: 'password123',
    handle,
    displayName: `${prefix}テスト`,
  };
};

/** 一意な投稿本文を作る（280文字以内） */
export const uniqueContent = (label: string): string => {
  sequence += 1;
  return `${label}_${Date.now().toString(36)}_${sequence}`;
};

/** 新規登録する（登録に成功するとホームタイムラインへ遷移する） */
export const register = async (page: Page, user: TestUser): Promise<void> => {
  await page.goto('/signup');
  await page.getByTestId('signup-email').fill(user.email);
  await page.getByTestId('signup-display-name').fill(user.displayName);
  await page.getByTestId('signup-handle').fill(user.handle);
  await page.getByTestId('signup-password').fill(user.password);
  await page.getByTestId('signup-confirm-password').fill(user.password);
  await page.getByTestId('signup-submit').click();

  await expect(page.getByTestId('post-form')).toBeVisible();
};

/** ログインする */
export const login = async (page: Page, user: TestUser): Promise<void> => {
  await page.goto('/login');
  await page.getByTestId('login-email').fill(user.email);
  await page.getByTestId('login-password').fill(user.password);
  await page.getByTestId('login-submit').click();

  await expect(page.getByTestId('post-form')).toBeVisible();
};

/** ログアウトする */
export const logout = async (page: Page): Promise<void> => {
  await page.getByTestId('logout-button').click();
  await expect(page).toHaveURL(/\/login$/);
};

/** ホームタイムラインから投稿する */
export const createPost = async (page: Page, content: string): Promise<void> => {
  await page.goto('/');
  await page.getByTestId('post-form-input').fill(content);
  await page.getByTestId('post-form-submit').click();

  await expect(postCard(page, content)).toBeVisible();
};

/** 本文で投稿カードを特定する */
export const postCard = (scope: Page | Locator, content: string): Locator =>
  scope.getByTestId('post-card').filter({ hasText: content }).first();

/** 投稿詳細ページを開く */
export const openPostDetail = async (page: Page, content: string): Promise<void> => {
  await postCard(page, content).getByTestId('post-content').click();
  await expect(page.getByTestId('post-detail-page')).toBeVisible();
};

/** 投稿詳細ページでコメントを投稿する */
export const createComment = async (page: Page, content: string): Promise<void> => {
  await page.getByTestId('comment-form-input').fill(content);
  await page.getByTestId('comment-form-submit').click();

  await expect(postCard(page.getByTestId('comment-list'), content)).toBeVisible();
};

/** 投稿カードのメニューから項目を選択する */
export const openPostMenu = async (card: Locator): Promise<void> => {
  await card.getByTestId('post-menu-button').click();
};
