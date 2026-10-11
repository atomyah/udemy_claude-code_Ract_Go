import { expect, test } from '@playwright/test';

/** テスト用APIサーバー（api_test） */
const API_BASE_URL = 'http://localhost:8081/api/v1';
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

test.describe('リクエストID・エラーレスポンス', () => {
  test('API のレスポンスにはリクエストごとに異なる X-Request-ID が付き、ブラウザから読める', async ({ page }) => {
    await page.goto('/login');

    // フロントエンドのオリジンから CORS で取得し、公開ヘッダーとして読めることを確認する
    const ids = await page.evaluate(async (url) => {
      const read = async () => (await fetch(`${url}/posts?limit=1`, { credentials: 'include' })).headers.get('x-request-id');
      return [await read(), await read()];
    }, API_BASE_URL);

    expect(ids[0]).toMatch(UUID_PATTERN);
    expect(ids[1]).toMatch(UUID_PATTERN);
    expect(ids[0]).not.toBe(ids[1]);
  });

  test('クライアントが送った X-Request-ID はそのまま使われない', async ({ request }) => {
    const res = await request.get(`${API_BASE_URL}/posts?limit=1`, { headers: { 'X-Request-ID': 'forged-id' } });

    expect(res.status()).toBe(200);
    expect(res.headers()['x-request-id']).toMatch(UUID_PATTERN);
  });

  test('存在しない API は内部情報を含まない日本語の 404 を返す', async ({ request }) => {
    const res = await request.get(`${API_BASE_URL}/no-such-endpoint`);

    expect(res.status()).toBe(404);
    expect(res.headers()['x-request-id']).toMatch(UUID_PATTERN);
    const body = (await res.json()) as { code: string; message: string };
    expect(body.code).toBe('NOT_FOUND');
    expect(body.message).toBe('指定されたページが見つかりません');
  });
});
