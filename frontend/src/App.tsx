import { CssBaseline, ThemeProvider } from '@mui/material';
import { Route, Routes } from 'react-router-dom';
import { AdminRoute } from './components/routing/AdminRoute';
import { PrivateRoute } from './components/routing/PrivateRoute';
import { AppLayout } from './components/layout/AppLayout';
import { ToastProvider } from './components/ToastProvider';
import AdminPage from './pages/AdminPage';
import BookmarksPage from './pages/BookmarksPage';
import ExplorePage from './pages/ExplorePage';
import ForgotPasswordPage from './pages/ForgotPasswordPage';
import HashtagPage from './pages/HashtagPage';
import HomePage from './pages/HomePage';
import LoginPage from './pages/LoginPage';
import NotFoundPage from './pages/NotFoundPage';
import NotificationsPage from './pages/NotificationsPage';
import PostDetailPage from './pages/PostDetailPage';
import ProfilePage from './pages/ProfilePage';
import ResetPasswordPage from './pages/ResetPasswordPage';
import SettingsPage from './pages/SettingsPage';
import SignupPage from './pages/SignupPage';
import { darkTheme, lightTheme } from './theme';
import { useThemeMode } from './theme/ThemeContext';

function App() {
  const { mode } = useThemeMode();
  const theme = mode === 'dark' ? darkTheme : lightTheme;

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ToastProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />

          <Route element={<AppLayout />}>
            <Route element={<PrivateRoute />}>
              <Route path="/" element={<HomePage />} />
              <Route path="/explore" element={<ExplorePage />} />
              <Route path="/notifications" element={<NotificationsPage />} />
              <Route path="/bookmarks" element={<BookmarksPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Route>

            <Route element={<AdminRoute />}>
              <Route path="/admin" element={<AdminPage />} />
            </Route>

            <Route path="/posts/:id" element={<PostDetailPage />} />
            {/* /:handle より前に置く（ハッシュタグがハンドルとして解釈されるのを防ぐ） */}
            <Route path="/hashtag/:tag" element={<HashtagPage />} />
            <Route path="/:handle" element={<ProfilePage />} />
          </Route>

          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </ToastProvider>
    </ThemeProvider>
  );
}

export default App;
