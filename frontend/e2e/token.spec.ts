import { createHmac } from 'node:crypto';
import { expect, test, type BrowserContext, type Cookie } from '@playwright/test';
import { makeUser, postCard, register, uniqueContent } from './helpers';

/** テスト用APIサーバー（api_test） */
const API_BASE_URL = 'http://localhost:8081/api/v1';
/** E2E 用フロントエンド（playwright.config.ts の baseURL）。手動で作るコンテキストにも指定する */
const APP_BASE_URL = 'http://localhost:5174';
/** テスト用 JWT 署名鍵（backend/.env.test・ci.yml と同じ値。秘密情報ではない） */
const TEST_JWT_SECRET = process.env.JWT_SECRET ?? 'test_jwt_secret_for_local_testing_only_32bytes';

/** API に送られる認証 Cookie（access_token: Path=/api/v1、refresh_token: Path=/api/v1/auth）を取得する */
const authCookies = async (context: BrowserContext) => {
  const cookies = await context.cookies(`${API_BASE_URL}/auth/refresh`);
  const find = (name: string): Cookie | undefined => cookies.find((c) => c.name === name);
  return { access: find('access_token'), refresh: find('refresh_token') };
};

/** JWT のペイロードをデコードする */
const decodePayload = (token: string): Record<string, unknown> =>
  JSON.parse(Buffer.from(token.split('.')[1], 'base64url').toString('utf8')) as Record<string, unknown>;

/** テスト用の鍵で HS256 の JWT を作る（期限切れトークンの再現に使う） */
const signJwt = (payload: Record<string, unknown>): string => {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  const unsigned = `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(payload)}`;
  const signature = createHmac('sha256', TEST_JWT_SECRET).update(unsigned).digest('base64url');
  return `${unsigned}.${signature}`;
};

/** Set-Cookie ヘッダーから指定の Cookie の値を取り出す */
const cookieValueFromSetCookie = (setCookie: string, name: string): string | undefined =>
  setCookie
    .split('\n')
    .map((line) => line.match(new RegExp(`^${name}=([^;]*)`))?.[1])
    .find((value) => value !== undefined && value !== '');

test.describe('トークン管理（HttpOnly Cookie）', () => {
  test('ログインするとトークンが HttpOnly Cookie で保存され、localStorage やヘッダーには出てこない', async ({
    page,
    context,
  }) => {
    const apiAuthHeaders: (string | undefined)[] = [];
    page.on('request', (req) => {
      if (req.url().startsWith(API_BASE_URL)) apiAuthHeaders.push(req.headers().authorization);
    });

    await register(page, makeUser('cookie'));

    const { access, refresh } = await authCookies(context);
    expect(access, 'access_token Cookie').toBeDefined();
    expect(refresh, 'refresh_token Cookie').toBeDefined();
    for (const cookie of [access!, refresh!]) {
      expect(cookie.httpOnly).toBe(true);
      expect(cookie.secure).toBe(true);
      expect(cookie.sameSite).toBe('None');
    }
    expect(access!.path).toBe('/api/v1');
    expect(refresh!.path).toBe('/api/v1/auth');

    // アクセストークンのペイロードは user_id と exp だけで、有効期限は1時間
    const payload = decodePayload(access!.value);
    expect(Object.keys(payload).sort()).toEqual(['exp', 'user_id']);
    const ttlSeconds = (payload.exp as number) - Math.floor(Date.now() / 1000);
    expect(ttlSeconds).toBeGreaterThan(3500);
    expect(ttlSeconds).toBeLessThanOrEqual(3600);

    // JavaScript からはトークンを読めない・保存していない
    const storage = await page.evaluate(() => ({ cookie: document.cookie, keys: Object.keys(localStorage) }));
    expect(storage.cookie).not.toContain('access_token');
    expect(storage.cookie).not.toContain('refresh_token');
    expect(storage.keys).not.toContain('access_token');

    // API へのリクエストに Authorization ヘッダーを付けていない（Cookie だけで認証している）
    expect(apiAuthHeaders.length).toBeGreaterThan(0);
    expect(apiAuthHeaders.every((h) => h === undefined)).toBe(true);
  });

  test('アクセストークンが期限切れでも自動でリフレッシュされ、そのまま投稿を続けられる', async ({ page, context }) => {
    await register(page, makeUser('refresh'));
    const before = await authCookies(context);

    // アクセストークンを期限切れのものに差し替える（1時間経過した状態を再現）
    const { user_id: userId } = decodePayload(before.access!.value);
    const expired = signJwt({ user_id: userId, exp: Math.floor(Date.now() / 1000) - 60 });
    await context.addCookies([{ ...before.access!, value: expired }]);

    const postStatuses: number[] = [];
    page.on('response', (res) => {
      if (res.url() === `${API_BASE_URL}/posts` && res.request().method() === 'POST') postStatuses.push(res.status());
    });
    const refreshed = page.waitForResponse(
      (res) => res.url() === `${API_BASE_URL}/auth/refresh` && res.request().method() === 'POST',
    );

    // 画面を再読み込みせずに操作を続ける
    const content = uniqueContent('自動リフレッシュ');
    await page.getByTestId('post-form-input').fill(content);
    await page.getByTestId('post-form-submit').click();

    expect((await refreshed).status()).toBe(200);
    await expect(postCard(page, content)).toBeVisible();
    // 1回目は期限切れで 401、リフレッシュ後の再送で 201
    expect(postStatuses).toEqual([401, 201]);
    await expect(page).not.toHaveURL(/\/login$/);

    // トークンはどちらも新しいものに入れ替わる（リフレッシュトークンもローテーションされる）
    const after = await authCookies(context);
    expect(after.access!.value).not.toBe(expired);
    expect(after.refresh!.value).not.toBe(before.refresh!.value);
  });

  test('リフレッシュもできないときはログイン画面にリダイレクトされる', async ({ page, context }) => {
    await register(page, makeUser('expired'));

    // Cookie をすべて失った状態（期限切れ・別端末でのログアウトなど）で操作する
    await context.clearCookies();
    await page.getByTestId('post-form-input').fill(uniqueContent('送れない投稿'));
    await page.getByTestId('post-form-submit').click();

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByTestId('login-page')).toBeVisible();
  });

  test('リフレッシュトークンは1回しか使えず、使用済みトークンが再利用されると系列ごと失効する', async ({
    page,
    context,
    request,
  }) => {
    await register(page, makeUser('rotate'));
    const { refresh } = await authCookies(context);
    const refreshWith = (token: string) =>
      request.post(`${API_BASE_URL}/auth/refresh`, { headers: { Cookie: `refresh_token=${token}` } });

    // 1回目: 新しいトークンに交換できる
    const first = await refreshWith(refresh!.value);
    expect(first.status()).toBe(200);
    const rotated = cookieValueFromSetCookie(first.headers()['set-cookie'] ?? '', 'refresh_token');
    expect(rotated).toBeDefined();
    expect(rotated).not.toBe(refresh!.value);

    // 2回目: 使用済みのトークンは使えない（漏洩とみなす）
    expect((await refreshWith(refresh!.value)).status()).toBe(401);
    // 再利用を検知したので、交換後の新しいトークンも失効している
    expect((await refreshWith(rotated!)).status()).toBe(401);
  });

  test('ログアウトするとリフレッシュトークンが失効し Cookie も削除される', async ({ page, context, request }) => {
    await register(page, makeUser('logout_revoke'));
    const { refresh } = await authCookies(context);

    await page.getByTestId('logout-button').click();
    await expect(page).toHaveURL(/\/login$/);

    const after = await authCookies(context);
    expect(after.access).toBeUndefined();
    expect(after.refresh).toBeUndefined();

    // ログアウト前のリフレッシュトークンを持ち出しても使えない
    const res = await request.post(`${API_BASE_URL}/auth/refresh`, {
      headers: { Cookie: `refresh_token=${refresh!.value}` },
    });
    expect(res.status()).toBe(401);
  });

  test('全デバイスからログアウトすると、他の端末のリフレッシュトークンも失効する', async ({ browser }) => {
    const user = makeUser('revoke_all');

    // 端末A で登録、端末B で同じユーザーがログインする
    const deviceA = await browser.newContext({ baseURL: APP_BASE_URL });
    const pageA = await deviceA.newPage();
    await register(pageA, user);

    const deviceB = await browser.newContext({ baseURL: APP_BASE_URL });
    const pageB = await deviceB.newPage();
    await pageB.goto('/login');
    await pageB.getByTestId('login-email').fill(user.email);
    await pageB.getByTestId('login-password').fill(user.password);
    await pageB.getByTestId('login-submit').click();
    await expect(pageB.getByTestId('post-form')).toBeVisible();

    // 端末A から全デバイスログアウト API を呼ぶ（Cookie はブラウザが送る）
    const status = await pageA.evaluate(async (url) => {
      const res = await fetch(`${url}/auth/revoke-all`, { method: 'POST', credentials: 'include' });
      return res.status;
    }, API_BASE_URL);
    expect(status).toBe(204);

    // 端末B はアクセストークンが切れた時点（ここでは削除して再現）でリフレッシュできず、ログイン画面に戻される
    await deviceB.clearCookies({ name: 'access_token' });
    await pageB.getByTestId('post-form-input').fill(uniqueContent('失効後の投稿'));
    await pageB.getByTestId('post-form-submit').click();
    await expect(pageB).toHaveURL(/\/login$/);

    await deviceA.close();
    await deviceB.close();
  });
});
