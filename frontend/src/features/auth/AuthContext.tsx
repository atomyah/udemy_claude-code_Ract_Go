import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { useNavigate } from 'react-router-dom';
import { setSessionExpiredHandler } from '../../api/client';
import { queryClient } from '../../api/queryClient';
import type { RegisterRequest, UserResponse } from '../../api/types';
import { getMe, login as loginRequest, logout as logoutRequest, register as registerRequest } from './api';

interface AuthContextValue {
  currentUser: UserResponse | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  /** ログインAPI以外（Google ログインなど）でセッションを開始したときにユーザー情報を反映する */
  loginWithUser: (user: UserResponse) => void;
  register: (payload: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  updateCurrentUser: (user: UserResponse) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/**
 * ログイン状態を管理する。トークンは HttpOnly Cookie にありフロントエンドからは読めないため、
 * 起動時に GET /users/me を呼んでセッションの有無を確認する（期限切れなら client.ts が自動でリフレッシュする）。
 */
export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [currentUser, setCurrentUser] = useState<UserResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const navigate = useNavigate();
  // セッション切れハンドラーから最新のログイン状態を参照するため ref でも保持する
  const isAuthenticatedRef = useRef(false);
  isAuthenticatedRef.current = currentUser !== null;

  useEffect(() => {
    getMe()
      .then((user) => setCurrentUser(user))
      .catch(() => setCurrentUser(null))
      .finally(() => setIsLoading(false));
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      const wasAuthenticated = isAuthenticatedRef.current;
      setCurrentUser(null);
      queryClient.clear();
      // ログイン中にリフレッシュも失敗したらログイン画面へ。
      // 未ログインで公開ページを見ているだけのとき（起動時の確認など）は遷移させない
      if (wasAuthenticated) {
        navigate('/login', { replace: true });
      }
    });
    return () => setSessionExpiredHandler(() => {});
  }, [navigate]);

  const login = useCallback(async (email: string, password: string) => {
    const { user } = await loginRequest({ email, password });
    setCurrentUser(user ?? null);
  }, []);

  const loginWithUser = useCallback((user: UserResponse) => {
    setCurrentUser(user);
  }, []);

  const updateCurrentUser = useCallback((user: UserResponse) => {
    setCurrentUser(user);
  }, []);

  const register = useCallback(async (payload: RegisterRequest) => {
    const { user } = await registerRequest(payload);
    setCurrentUser(user ?? null);
  }, []);

  const logout = useCallback(async () => {
    try {
      // バックエンドがリフレッシュトークンを失効させ、Set-Cookie で Cookie を削除する
      await logoutRequest();
    } finally {
      setCurrentUser(null);
      queryClient.clear();
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      currentUser,
      isAuthenticated: currentUser !== null,
      isLoading,
      login,
      loginWithUser,
      register,
      logout,
      updateCurrentUser,
    }),
    [currentUser, isLoading, login, loginWithUser, register, logout, updateCurrentUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextValue => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
