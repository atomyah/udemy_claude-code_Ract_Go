import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';

/**
 * API クライアント。
 * 認証トークン（access_token / refresh_token）はバックエンドが HttpOnly Cookie で発行し、ブラウザが自動で送る。
 * フロントエンドはトークンを一切扱わない（localStorage への保存・Authorization ヘッダーの付与はしない）。
 * 別オリジンの API に Cookie を送るため withCredentials を必ず有効にする。
 */
export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true,
});

type RetryableConfig = InternalAxiosRequestConfig & { _retry?: boolean };

/** 401 が返ってもリフレッシュしないエンドポイント（認証そのものの成否を返すAPI） */
const ENDPOINTS_WITHOUT_REFRESH = ['/auth/refresh', '/auth/login', '/auth/register', '/auth/google', '/auth/logout'];

let onSessionExpired: (() => void) | null = null;

/** リフレッシュにも失敗した（ログインが切れた）ときに呼ぶ処理を登録する */
export const setSessionExpiredHandler = (handler: () => void): void => {
  onSessionExpired = handler;
};

let refreshPromise: Promise<void> | null = null;

/**
 * リフレッシュAPIを呼び、Cookie のアクセストークン・リフレッシュトークンを更新する。
 * リフレッシュトークンは1回しか使えない（ローテーション）ため、同時に複数の 401 が返っても
 * リフレッシュのリクエストは1回だけ送り、全員がその結果を待つ。
 */
export const refreshSession = (): Promise<void> => {
  if (!refreshPromise) {
    refreshPromise = api
      .post('/auth/refresh')
      .then(() => undefined)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
};

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const config = error.config as RetryableConfig | undefined;

    if (
      error.response?.status !== 401 ||
      !config ||
      config._retry ||
      ENDPOINTS_WITHOUT_REFRESH.includes(config.url ?? '')
    ) {
      return Promise.reject(error);
    }

    config._retry = true;
    try {
      await refreshSession();
    } catch {
      onSessionExpired?.();
      return Promise.reject(error);
    }
    // 新しいアクセストークンは Cookie に入っているので、同じリクエストをそのまま再送する
    return api(config);
  },
);
