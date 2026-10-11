import { AxiosError, AxiosHeaders, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { api, setSessionExpiredHandler } from './client';

type Reply = { status: number; data?: unknown };

/**
 * 実際の通信の代わりに、URL ごとに用意した応答を順番に返すアダプター。
 * 受け取ったリクエストは requests に記録する
 */
const createAdapter = (replies: Record<string, Reply[]>) => {
  const requests: InternalAxiosRequestConfig[] = [];
  const adapter: AxiosAdapter = async (config) => {
    requests.push(config);
    const queue = replies[`${config.method?.toUpperCase()} ${config.url}`] ?? [];
    const reply = queue.shift() ?? { status: 500 };
    const response = {
      data: reply.data ?? {},
      status: reply.status,
      statusText: String(reply.status),
      headers: new AxiosHeaders(),
      config,
    };
    if (reply.status >= 400) {
      throw new AxiosError(`status ${reply.status}`, String(reply.status), config, null, response);
    }
    return response;
  };
  return { adapter, requests };
};

const countRequests = (requests: InternalAxiosRequestConfig[], key: string) =>
  requests.filter((r) => `${r.method?.toUpperCase()} ${r.url}` === key).length;

describe('API クライアント（Cookie 認証）', () => {
  const originalAdapter = api.defaults.adapter;
  const onExpired = vi.fn();

  beforeEach(() => {
    onExpired.mockReset();
    setSessionExpiredHandler(onExpired);
    localStorage.clear();
  });

  afterEach(() => {
    api.defaults.adapter = originalAdapter;
  });

  test('Cookie を送るため withCredentials が有効で、Authorization ヘッダーは付けない', async () => {
    const { adapter, requests } = createAdapter({ 'GET /users/me': [{ status: 200 }] });
    api.defaults.adapter = adapter;
    localStorage.setItem('access_token', 'stale-token-from-old-version');

    await api.get('/users/me');

    expect(requests[0].withCredentials).toBe(true);
    expect(requests[0].headers.Authorization).toBeUndefined();
  });

  test('401 が返るとリフレッシュAPIを呼び、元のリクエストを再送して成功する', async () => {
    const { adapter, requests } = createAdapter({
      'POST /posts': [{ status: 401 }, { status: 201, data: { id: 'p1' } }],
      'POST /auth/refresh': [{ status: 200 }],
    });
    api.defaults.adapter = adapter;

    const res = await api.post('/posts', { content: 'hello' });

    expect(res.status).toBe(201);
    expect(requests.map((r) => `${r.method?.toUpperCase()} ${r.url}`)).toEqual([
      'POST /posts',
      'POST /auth/refresh',
      'POST /posts',
    ]);
    expect(onExpired).not.toHaveBeenCalled();
  });

  test('同時に複数の 401 が返ってもリフレッシュは1回だけ行う', async () => {
    const { adapter, requests } = createAdapter({
      'GET /posts/home': [{ status: 401 }, { status: 200 }],
      'GET /notifications': [{ status: 401 }, { status: 200 }],
      'GET /bookmarks': [{ status: 401 }, { status: 200 }],
      'POST /auth/refresh': [{ status: 200 }],
    });
    api.defaults.adapter = adapter;

    await Promise.all([api.get('/posts/home'), api.get('/notifications'), api.get('/bookmarks')]);

    expect(countRequests(requests, 'POST /auth/refresh')).toBe(1);
  });

  test('リフレッシュも失敗するとセッション切れハンドラーを呼び、元のエラーを返す', async () => {
    const { adapter, requests } = createAdapter({
      'GET /bookmarks': [{ status: 401 }],
      'POST /auth/refresh': [{ status: 401 }],
    });
    api.defaults.adapter = adapter;

    await expect(api.get('/bookmarks')).rejects.toMatchObject({ response: { status: 401 } });

    expect(onExpired).toHaveBeenCalledTimes(1);
    expect(countRequests(requests, 'GET /bookmarks')).toBe(1);
  });

  test('再送しても 401 なら、もう一度リフレッシュはしない（無限ループしない）', async () => {
    const { adapter, requests } = createAdapter({
      'GET /users/me': [{ status: 401 }, { status: 401 }],
      'POST /auth/refresh': [{ status: 200 }, { status: 200 }],
    });
    api.defaults.adapter = adapter;

    await expect(api.get('/users/me')).rejects.toMatchObject({ response: { status: 401 } });

    expect(countRequests(requests, 'POST /auth/refresh')).toBe(1);
  });

  test.each(['/auth/login', '/auth/register', '/auth/refresh', '/auth/google', '/auth/logout'])(
    '%s の 401 ではリフレッシュしない',
    async (url) => {
      const { adapter, requests } = createAdapter({ [`POST ${url}`]: [{ status: 401 }] });
      api.defaults.adapter = adapter;

      await expect(api.post(url)).rejects.toMatchObject({ response: { status: 401 } });

      expect(requests).toHaveLength(1);
      expect(onExpired).not.toHaveBeenCalled();
    },
  );

  test('401 以外のエラーではリフレッシュしない', async () => {
    const { adapter, requests } = createAdapter({ 'GET /posts': [{ status: 500 }] });
    api.defaults.adapter = adapter;

    await expect(api.get('/posts')).rejects.toMatchObject({ response: { status: 500 } });

    expect(requests).toHaveLength(1);
  });
});
