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
import { api } from '../api/client';
import { useAuth } from '../features/auth/AuthContext';
import type { ThemeMode } from './index';

const STORAGE_KEY = 'theme_mode';

const readStoredMode = (): ThemeMode => {
  const stored = localStorage.getItem(STORAGE_KEY);
  return stored === 'dark' ? 'dark' : 'light';
};

interface ThemeModeContextValue {
  mode: ThemeMode;
  toggleTheme: () => void;
}

const ThemeModeContext = createContext<ThemeModeContextValue | undefined>(undefined);

export const ThemeModeProvider = ({ children }: { children: ReactNode }) => {
  const { currentUser, isAuthenticated } = useAuth();
  const [mode, setMode] = useState<ThemeMode>(readStoredMode);
  const syncedUserId = useRef<string | null>(null);

  useEffect(() => {
    const userTheme = currentUser?.theme;
    if (currentUser?.id && currentUser.id !== syncedUserId.current && userTheme) {
      syncedUserId.current = currentUser.id;
      setMode(userTheme === 'dark' ? 'dark' : 'light');
    }
  }, [currentUser]);

  const toggleTheme = useCallback(() => {
    const next: ThemeMode = mode === 'light' ? 'dark' : 'light';
    localStorage.setItem(STORAGE_KEY, next);
    setMode(next);
    if (isAuthenticated) {
      void api.put('/users/me/theme', { theme: next });
    }
  }, [mode, isAuthenticated]);

  const value = useMemo(() => ({ mode, toggleTheme }), [mode, toggleTheme]);

  return <ThemeModeContext.Provider value={value}>{children}</ThemeModeContext.Provider>;
};

export const useThemeMode = (): ThemeModeContextValue => {
  const context = useContext(ThemeModeContext);
  if (!context) {
    throw new Error('useThemeMode must be used within a ThemeModeProvider');
  }
  return context;
};
