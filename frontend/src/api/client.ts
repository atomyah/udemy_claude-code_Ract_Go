import axios, { type InternalAxiosRequestConfig } from 'axios';
import type { RefreshResponse } from './types';

const ACCESS_TOKEN_KEY = 'access_token';

export const getAccessToken = (): string | null => localStorage.getItem(ACCESS_TOKEN_KEY);
export const setAccessToken = (token: string): void => localStorage.setItem(ACCESS_TOKEN_KEY, token);
export const clearAccessToken = (): void => localStorage.removeItem(ACCESS_TOKEN_KEY);

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true,
});

api.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

type RetryableConfig = InternalAxiosRequestConfig & { _retry?: boolean };

const AUTH_ENDPOINTS_WITHOUT_REFRESH = ['/auth/refresh', '/auth/login', '/auth/register'];

let onSessionExpired: (() => void) | null = null;

export const setSessionExpiredHandler = (handler: () => void): void => {
  onSessionExpired = handler;
};

let refreshPromise: Promise<string> | null = null;

const refreshAccessToken = async (): Promise<string> => {
  if (!refreshPromise) {
    refreshPromise = axios
      .create({ baseURL: import.meta.env.VITE_API_BASE_URL, withCredentials: true })
      .post<RefreshResponse>('/auth/refresh')
      .then((res) => {
        const token = res.data.access_token;
        if (!token) {
          throw new Error('access_token missing in refresh response');
        }
        setAccessToken(token);
        return token;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
};

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const config = error.config as RetryableConfig | undefined;
    const status = error.response?.status;

    if (
      status !== 401 ||
      !config ||
      config._retry ||
      AUTH_ENDPOINTS_WITHOUT_REFRESH.includes(config.url ?? '')
    ) {
      return Promise.reject(error);
    }

    config._retry = true;
    try {
      const token = await refreshAccessToken();
      config.headers = config.headers ?? {};
      config.headers.Authorization = `Bearer ${token}`;
      return api(config);
    } catch (refreshError) {
      clearAccessToken();
      onSessionExpired?.();
      return Promise.reject(refreshError);
    }
  },
);
