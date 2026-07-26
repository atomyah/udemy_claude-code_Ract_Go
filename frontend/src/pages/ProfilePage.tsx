import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import LinkIcon from '@mui/icons-material/Link';
import PlaceIcon from '@mui/icons-material/Place';
import { Avatar, Box, Button, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { InfinitePostList } from '../components/InfinitePostList';
import { LoadingSpinner } from '../components/LoadingSpinner';
import { useAuth } from '../features/auth/AuthContext';
import { EditProfileDialog } from '../features/profile/components/EditProfileDialog';
import { FollowListDialog } from '../features/profile/components/FollowListDialog';
import { useFollow } from '../features/profile/hooks/useFollow';
import { useProfile } from '../features/profile/hooks/useProfile';
import { useUserPosts } from '../features/profile/hooks/useUserPosts';
import { useUserReplies } from '../features/profile/hooks/useUserReplies';

const TABS = ['投稿', '返信'] as const;

const ProfilePage = () => {
  const { handle } = useParams<{ handle: string }>();
  const { currentUser } = useAuth();
  const { data: profile, isLoading } = useProfile(handle);
  const posts = useUserPosts(handle);
  const followMutation = useFollow(handle ?? '');

  const [tab, setTab] = useState(0);
  const [editOpen, setEditOpen] = useState(false);
  const [followDialog, setFollowDialog] = useState<'followers' | 'following' | null>(null);

  // 返信一覧はタブが選択されたときに初めて取得する
  const replies = useUserReplies(handle, tab === 1);

  if (isLoading) return <LoadingSpinner />;
  if (!profile) {
    return (
      <Typography color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
        ユーザーが見つかりません
      </Typography>
    );
  }

  const isSelf = currentUser?.id !== undefined && currentUser.id === profile.id;

  return (
    <Box>
      <Box
        sx={{
          height: 160,
          borderRadius: 1,
          bgcolor: 'action.hover',
          backgroundImage: profile.banner_url ? `url(${profile.banner_url})` : undefined,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
        }}
      />
      <Box sx={{ px: 2, display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', mt: -6 }}>
        <Avatar
          src={profile.avatar_url ?? undefined}
          sx={{ width: 96, height: 96, border: '4px solid', borderColor: 'background.default' }}
        >
          {(profile.display_name ?? '?').charAt(0)}
        </Avatar>
        <Box sx={{ pb: 1 }}>
          {isSelf ? (
            <Button variant="outlined" onClick={() => setEditOpen(true)}>
              プロフィールを編集
            </Button>
          ) : (
            <Button
              variant={profile.is_following ? 'outlined' : 'contained'}
              onClick={() => followMutation.mutate(profile.is_following ?? false)}
              disabled={followMutation.isPending}
            >
              {profile.is_following ? 'フォロー中' : 'フォロー'}
            </Button>
          )}
        </Box>
      </Box>

      <Box sx={{ px: 2, mt: 1 }}>
        <Typography variant="h6" sx={{ fontWeight: 700 }}>
          {profile.display_name}
        </Typography>
        <Typography color="text.secondary">@{profile.handle}</Typography>

        {profile.bio && <Typography sx={{ mt: 1, whiteSpace: 'pre-wrap' }}>{profile.bio}</Typography>}

        <Stack direction="row" spacing={2} sx={{ mt: 1, rowGap: 0.5, flexWrap: 'wrap' }}>
          {profile.location && (
            <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
              <PlaceIcon fontSize="small" color="disabled" />
              <Typography variant="body2" color="text.secondary">
                {profile.location}
              </Typography>
            </Stack>
          )}
          {profile.website_url && (
            <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
              <LinkIcon fontSize="small" color="disabled" />
              <Typography
                variant="body2"
                component="a"
                href={profile.website_url}
                target="_blank"
                rel="noopener noreferrer"
                sx={{ color: 'primary.main' }}
              >
                {profile.website_url}
              </Typography>
            </Stack>
          )}
          {profile.birthday && (
            <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
              <CalendarMonthIcon fontSize="small" color="disabled" />
              <Typography variant="body2" color="text.secondary">
                {profile.birthday}
              </Typography>
            </Stack>
          )}
        </Stack>

        <Stack direction="row" spacing={2} sx={{ mt: 1.5 }}>
          <Typography
            variant="body2"
            sx={{ cursor: 'pointer' }}
            onClick={() => setFollowDialog('following')}
          >
            <strong>{profile.following_count ?? 0}</strong> <Box component="span" sx={{ color: 'text.secondary' }}>フォロー中</Box>
          </Typography>
          <Typography
            variant="body2"
            sx={{ cursor: 'pointer' }}
            onClick={() => setFollowDialog('followers')}
          >
            <strong>{profile.followers_count ?? 0}</strong> <Box component="span" sx={{ color: 'text.secondary' }}>フォロワー</Box>
          </Typography>
        </Stack>
      </Box>

      <Tabs value={tab} onChange={(_, v) => setTab(v)} variant="fullWidth" sx={{ mt: 2, borderBottom: 1, borderColor: 'divider' }}>
        {TABS.map((label) => (
          <Tab key={label} label={label} />
        ))}
      </Tabs>

      {tab === 0 ? (
        <InfinitePostList
          data={posts.data}
          isLoading={posts.isLoading}
          isError={posts.isError}
          isFetchingNextPage={posts.isFetchingNextPage}
          hasNextPage={posts.hasNextPage}
          fetchNextPage={posts.fetchNextPage}
          emptyMessage="まだ投稿がありません"
        />
      ) : (
        <InfinitePostList
          data={replies.data}
          isLoading={replies.isLoading}
          isError={replies.isError}
          isFetchingNextPage={replies.isFetchingNextPage}
          hasNextPage={replies.hasNextPage}
          fetchNextPage={replies.fetchNextPage}
          emptyMessage="まだ返信がありません"
        />
      )}

      <EditProfileDialog user={profile} open={editOpen} onClose={() => setEditOpen(false)} />
      {followDialog && (
        <FollowListDialog
          handle={handle ?? ''}
          mode={followDialog}
          open={!!followDialog}
          onClose={() => setFollowDialog(null)}
        />
      )}
    </Box>
  );
};

export default ProfilePage;
