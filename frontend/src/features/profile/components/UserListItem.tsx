import { Avatar, Box, Button, Stack, Typography } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import type { UserResponse } from '../../../api/types';
import { useAuth } from '../../auth/AuthContext';
import { useFollow } from '../hooks/useFollow';

interface UserListItemProps {
  user: UserResponse;
  onNavigate?: () => void;
}

export const UserListItem = ({ user, onNavigate }: UserListItemProps) => {
  const { currentUser } = useAuth();
  const followMutation = useFollow(user.handle ?? '');
  const isSelf = currentUser?.id !== undefined && currentUser.id === user.id;

  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', px: 2, py: 1.5 }}>
      <Avatar
        component={RouterLink}
        to={`/${user.handle ?? ''}`}
        onClick={onNavigate}
        src={user.avatar_url ?? undefined}
      >
        {(user.display_name ?? '?').charAt(0)}
      </Avatar>
      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Typography
          component={RouterLink}
          to={`/${user.handle ?? ''}`}
          onClick={onNavigate}
          variant="subtitle2"
          sx={{ fontWeight: 700, color: 'text.primary', textDecoration: 'none', display: 'block' }}
        >
          {user.display_name}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          @{user.handle}
        </Typography>
        {user.bio && (
          <Typography variant="body2" sx={{ mt: 0.5 }}>
            {user.bio}
          </Typography>
        )}
      </Box>
      {!isSelf && (
        <Button
          size="small"
          variant={user.is_following ? 'outlined' : 'contained'}
          onClick={() => followMutation.mutate(user.is_following ?? false)}
          disabled={followMutation.isPending}
        >
          {user.is_following ? 'フォロー中' : 'フォロー'}
        </Button>
      )}
    </Stack>
  );
};
