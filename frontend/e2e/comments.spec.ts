import { expect, test } from '@playwright/test';
import {
  createComment,
  createPost,
  makeUser,
  openPostDetail,
  openPostMenu,
  postCard,
  register,
  uniqueContent,
} from './helpers';

test.describe('コメント', () => {
  test('コメントを投稿してから削除できる', async ({ page }) => {
    await register(page, makeUser('cmt'));
    const postContent = uniqueContent('コメントされる投稿');
    const commentContent = uniqueContent('削除するコメント');

    await createPost(page, postContent);
    await openPostDetail(page, postContent);
    await createComment(page, commentContent);

    const commentList = page.getByTestId('comment-list');
    const comment = postCard(commentList, commentContent);
    await expect(comment).toBeVisible();

    await openPostMenu(comment);
    await page.getByTestId('post-menu-delete').click();

    await expect(commentList.getByTestId('post-card').filter({ hasText: commentContent })).toHaveCount(0);
  });

  test('コメントにいいねといいね解除ができる', async ({ page }) => {
    await register(page, makeUser('cmt_like'));
    const postContent = uniqueContent('いいねされるコメントの元投稿');
    const commentContent = uniqueContent('いいねされるコメント');

    await createPost(page, postContent);
    await openPostDetail(page, postContent);
    await createComment(page, commentContent);

    const comment = postCard(page.getByTestId('comment-list'), commentContent);

    await comment.getByTestId('post-like-button').click();
    await expect(comment.getByTestId('post-like-count')).toHaveText('1');
    await expect(comment.getByTestId('post-like-button')).toHaveAttribute('data-active', 'true');

    await comment.getByTestId('post-like-button').click();
    await expect(comment.getByTestId('post-like-count')).toHaveText('');
    await expect(comment.getByTestId('post-like-button')).toHaveAttribute('data-active', 'false');
  });

  test('コメントに返信すると返信として表示される', async ({ page }) => {
    const user = makeUser('cmt_reply');
    await register(page, user);
    const postContent = uniqueContent('返信ツリーの元投稿');
    const commentContent = uniqueContent('返信されるコメント');
    const replyContent = uniqueContent('コメントへの返信');

    await createPost(page, postContent);
    await openPostDetail(page, postContent);
    await createComment(page, commentContent);

    // コメント自身の詳細ページを開き、そこへ返信する
    await postCard(page.getByTestId('comment-list'), commentContent).getByTestId('post-content').click();
    await expect(page.getByTestId('post-detail-main')).toContainText(commentContent);

    await createComment(page, replyContent);

    const reply = postCard(page.getByTestId('comment-list'), replyContent);
    await expect(reply).toBeVisible();
    // 返信先ユーザーが表示される
    await expect(reply).toContainText(`@${user.handle}`);

    // 元のコメントの返信数が1件になる
    await expect(page.getByTestId('post-detail-main').getByTestId('post-comment-count')).toHaveText('1');
  });
});
