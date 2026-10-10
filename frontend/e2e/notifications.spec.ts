import { expect, test } from '@playwright/test';
import {
  createComment,
  createPost,
  login,
  logout,
  makeUser,
  openPostDetail,
  postCard,
  register,
  uniqueContent,
} from './helpers';

test.describe('通知', () => {
  test('いいねされると通知ページに通知が表示され、未読バッジが消える', async ({ page }) => {
    const target = makeUser('notified');
    const actor = makeUser('notifier');

    // 通知を受け取る側が投稿する
    await register(page, target);
    const content = uniqueContent('通知のもとになる投稿');
    await createPost(page, content);
    await logout(page);

    // 別のユーザーが相手のプロフィールからその投稿にいいねする
    await register(page, actor);
    await page.goto(`/${target.handle}`);
    const card = postCard(page, content);
    await card.getByTestId('post-like-button').click();
    await expect(card.getByTestId('post-like-count')).toHaveText('1');
    // ログアウト元のページが再ログイン後の遷移先になるためホームへ戻ってからログアウトする
    await page.goto('/');
    await logout(page);

    // 投稿者側で未読バッジが表示される
    await login(page, target);
    const notificationNav = page.getByTestId('sidebar-nav-通知');
    await expect(notificationNav).toContainText('1');

    // 通知ページに「いいね」の通知が表示される
    await notificationNav.click();
    await expect(page.getByTestId('notifications-page')).toBeVisible();
    const notification = page.getByTestId('notification-item').first();
    await expect(notification).toHaveAttribute('data-type', 'like');
    await expect(notification).toContainText(actor.displayName);
    await expect(notification).toContainText('があなたの投稿をいいねしました');

    // ページを開いたことで既読になり、バッジが消える
    await expect(notification).toHaveAttribute('data-read', 'true');
    await expect(notificationNav).not.toContainText('1');

    // 「すべて既読にする」を押しても既読のまま保たれる
    await page.getByTestId('notifications-mark-all-read').click();
    await expect(page.getByTestId('notification-item').first()).toHaveAttribute('data-read', 'true');

    // 通知をクリックすると対象の投稿詳細へ遷移する
    await page.getByTestId('notification-item').first().click();
    await expect(page.getByTestId('post-detail-page')).toBeVisible();
    await expect(page.getByTestId('post-detail-page')).toContainText(content);
  });

  test('コメントされるとコメント種別の通知が表示される', async ({ page }) => {
    const target = makeUser('commentnotified');
    const actor = makeUser('commentnotifier');

    await register(page, target);
    const content = uniqueContent('コメントされる投稿');
    await createPost(page, content);
    await logout(page);

    // 別のユーザーがコメントする
    await register(page, actor);
    await page.goto(`/${target.handle}`);
    await openPostDetail(page, content);
    await createComment(page, uniqueContent('通知のためのコメント'));
    await logout(page);

    // 投稿者側に「コメント」通知が届く
    await login(page, target);
    await page.goto('/notifications');
    const notification = page.getByTestId('notification-item').first();
    await expect(notification).toHaveAttribute('data-type', 'comment');
    await expect(notification).toContainText(actor.displayName);
    await expect(notification).toContainText('があなたの投稿にコメントしました');
  });

  test('フォローされると通知が表示され、クリックするとフォロワーのプロフィールへ遷移する', async ({ page }) => {
    const target = makeUser('follownotified');
    const actor = makeUser('follownotifier');

    await register(page, target);
    await logout(page);

    // 別のユーザーがフォローする
    await register(page, actor);
    await page.goto(`/${target.handle}`);
    await page.getByTestId('profile-follow-button').click();
    await expect(page.getByTestId('profile-follow-button')).toHaveAttribute('data-following', 'true');
    await page.goto('/');
    await logout(page);

    // フォローされた側に「フォロー」通知が届く
    await login(page, target);
    await page.goto('/notifications');
    const notification = page.getByTestId('notification-item').first();
    await expect(notification).toHaveAttribute('data-type', 'follow');
    await expect(notification).toContainText(actor.displayName);
    await expect(notification).toContainText('にフォローされました');

    // 通知をクリックするとフォローしたユーザーのプロフィールへ遷移する
    await notification.click();
    await expect(page).toHaveURL(new RegExp(`/${actor.handle}$`));
    await expect(page.getByTestId('profile-display-name')).toHaveText(actor.displayName);
  });

  test('メンションされるとメンション種別の通知が表示される', async ({ page }) => {
    const target = makeUser('mentioned');
    const actor = makeUser('mentioner');

    await register(page, target);
    await logout(page);

    // 別のユーザーが@ハンドル付きで投稿する
    await register(page, actor);
    const content = `@${target.handle} ${uniqueContent('メンション投稿')}`;
    await createPost(page, content);
    await logout(page);

    await login(page, target);
    await page.goto('/notifications');
    const notification = page.getByTestId('notification-item').first();
    await expect(notification).toHaveAttribute('data-type', 'mention');
    await expect(notification).toContainText(actor.displayName);
    await expect(notification).toContainText('があなたをメンションしました');
  });

  test('通知がないときは通知がない旨が表示される', async ({ page }) => {
    await register(page, makeUser('nonotify'));

    await page.goto('/notifications');

    await expect(page.getByTestId('notifications-empty')).toHaveText('通知はありません');
  });
});
