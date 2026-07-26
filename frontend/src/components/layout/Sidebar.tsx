import AddIcon from '@mui/icons-material/Add';
import BookmarkIcon from '@mui/icons-material/Bookmark';
import ExploreIcon from '@mui/icons-material/Explore';
import HomeIcon from '@mui/icons-material/Home';
import NotificationsIcon from '@mui/icons-material/Notifications';
import PersonIcon from '@mui/icons-material/Person';
import SettingsIcon from '@mui/icons-material/Settings';
import {
  Badge,
  Box,
  Button,
  Divider,
  Drawer,
  Fab,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography,
} from '@mui/material';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../../features/auth/AuthContext';
import { useLogout } from '../../features/auth/hooks/useLogout';
import { useComposer } from '../../features/posts/ComposerContext';
import { useUnreadNotificationsCount } from '../../features/notifications/hooks/useUnreadNotificationsCount';

const DRAWER_WIDTH = 240;

export const Sidebar = () => {
  const { currentUser, isAuthenticated } = useAuth();
  const { mutate: logout } = useLogout();
  const { data: unreadCount = 0 } = useUnreadNotificationsCount();
  const { openComposer } = useComposer();

  const navItems = [
    { label: 'ホーム', to: '/', icon: <HomeIcon /> },
    { label: '探索', to: '/explore', icon: <ExploreIcon /> },
    { label: '通知', to: '/notifications', icon: <NotificationsIcon />, badge: unreadCount },
    { label: 'ブックマーク', to: '/bookmarks', icon: <BookmarkIcon /> },
    { label: 'プロフィール', to: currentUser ? `/${currentUser.handle}` : '/login', icon: <PersonIcon /> },
    { label: '設定', to: '/settings', icon: <SettingsIcon /> },
  ];

  return (
    <Drawer
      variant="permanent"
      sx={{
        display: { xs: 'none', md: 'block' },
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
          display: 'flex',
          flexDirection: 'column',
        },
      }}
    >
      <Box sx={{ p: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 700 }}>
          SNS App
        </Typography>
      </Box>
      <List>
        {navItems.map((item) => (
          <ListItemButton
            key={item.label}
            component={NavLink}
            to={item.to}
            sx={{
              '&.active': {
                bgcolor: 'action.selected',
                fontWeight: 700,
              },
            }}
          >
            <ListItemIcon>
              {item.badge ? (
                <Badge badgeContent={item.badge} color="secondary">
                  {item.icon}
                </Badge>
              ) : (
                item.icon
              )}
            </ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        ))}
      </List>

      <Box sx={{ px: 2, mt: 1 }}>
        <Fab color="primary" variant="extended" onClick={openComposer} sx={{ width: '100%' }}>
          <AddIcon sx={{ mr: 1 }} />
          投稿する
        </Fab>
      </Box>

      <Box sx={{ mt: 'auto', p: 2 }}>
        <Divider sx={{ mb: 2 }} />
        {isAuthenticated ? (
          <Button fullWidth variant="outlined" onClick={() => logout()}>
            ログアウト
          </Button>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Button fullWidth variant="contained" component={NavLink} to="/login">
              ログイン
            </Button>
            <Button fullWidth variant="outlined" component={NavLink} to="/signup">
              サインアップ
            </Button>
          </Box>
        )}
      </Box>
    </Drawer>
  );
};
