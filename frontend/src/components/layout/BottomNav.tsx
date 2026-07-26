import AddIcon from '@mui/icons-material/Add';
import BookmarkIcon from '@mui/icons-material/Bookmark';
import ExploreIcon from '@mui/icons-material/Explore';
import HomeIcon from '@mui/icons-material/Home';
import NotificationsIcon from '@mui/icons-material/Notifications';
import PersonIcon from '@mui/icons-material/Person';
import { Badge, Fab, Paper } from '@mui/material';
import BottomNavigation from '@mui/material/BottomNavigation';
import BottomNavigationAction from '@mui/material/BottomNavigationAction';
import { useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../../features/auth/AuthContext';
import { useComposer } from '../../features/posts/ComposerContext';
import { useUnreadNotificationsCount } from '../../features/notifications/hooks/useUnreadNotificationsCount';

export const BottomNav = () => {
  const { currentUser } = useAuth();
  const { data: unreadCount = 0 } = useUnreadNotificationsCount();
  const { openComposer } = useComposer();
  const location = useLocation();
  const navigate = useNavigate();

  const profilePath = currentUser ? `/${currentUser.handle}` : '/login';

  const navItems = [
    { label: 'ホーム', value: '/', icon: <HomeIcon /> },
    { label: '探索', value: '/explore', icon: <ExploreIcon /> },
    {
      label: '通知',
      value: '/notifications',
      icon: (
        <Badge badgeContent={unreadCount} color="secondary">
          <NotificationsIcon />
        </Badge>
      ),
    },
    { label: 'ブックマーク', value: '/bookmarks', icon: <BookmarkIcon /> },
    { label: 'プロフィール', value: profilePath, icon: <PersonIcon /> },
  ];

  return (
    <>
      <Fab
        color="primary"
        aria-label="投稿する"
        onClick={openComposer}
        sx={{
          display: { xs: 'flex', md: 'none' },
          position: 'fixed',
          bottom: 72,
          right: 16,
          zIndex: 1100,
        }}
      >
        <AddIcon />
      </Fab>
      <Paper
        elevation={3}
        sx={{ display: { xs: 'block', md: 'none' }, position: 'fixed', bottom: 0, left: 0, right: 0, zIndex: 1100 }}
      >
        <BottomNavigation showLabels value={location.pathname} onChange={(_, value) => navigate(value)}>
          {navItems.map((item) => (
            <BottomNavigationAction key={item.label} label={item.label} value={item.value} icon={item.icon} />
          ))}
        </BottomNavigation>
      </Paper>
    </>
  );
};
