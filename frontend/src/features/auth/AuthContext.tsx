import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { clearAccessToken, setAccessToken, setSessionExpiredHandler } from '../../api/client';
import { queryClient } from '../../api/queryClient';
import type { RegisterRequest, UserResponse } from '../../api/types';
import { getMe, login as loginRequest, logout as logoutRequest, register as registerRequest } from './api';

interface AuthContextValue {
  currentUser: UserResponse | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  loginWithToken: (accessToken: string, user: UserResponse) => void;
  register: (payload: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  updateCurrentUser: (user: UserResponse) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [currentUser, setCurrentUser] = useState<UserResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getMe()
      .then((user) => setCurrentUser(user))
      .catch(() => setCurrentUser(null))
      .finally(() => setIsLoading(false));
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      setCurrentUser(null);
      queryClient.clear();
    });
    return () => setSessionExpiredHandler(() => {});
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const { access_token, user } = await loginRequest({ email, password });
    if (access_token) setAccessToken(access_token);
    setCurrentUser(user ?? null);
  }, []);

  const loginWithToken = useCallback((accessToken: string, user: UserResponse) => {
    setAccessToken(accessToken);
    setCurrentUser(user);
  }, []);

  const updateCurrentUser = useCallback((user: UserResponse) => {
    setCurrentUser(user);
  }, []);

  const register = useCallback(async (payload: RegisterRequest) => {
    const { access_token, user } = await registerRequest(payload);
    if (access_token) setAccessToken(access_token);
    setCurrentUser(user ?? null);
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutRequest();
    } finally {
      clearAccessToken();
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
      loginWithToken,
      register,
      logout,
      updateCurrentUser,
    }),
    [currentUser, isLoading, login, loginWithToken, register, logout, updateCurrentUser],
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
