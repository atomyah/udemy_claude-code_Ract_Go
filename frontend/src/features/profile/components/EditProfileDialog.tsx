import PhotoCameraIcon from '@mui/icons-material/PhotoCamera';
import {
  Avatar,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import type { UserResponse } from '../../../api/types';
import { getApiErrorMessage } from '../../../utils/apiError';
import { useUpdateProfile } from '../hooks/useUpdateProfile';
import { useUploadAvatar } from '../hooks/useUploadAvatar';
import { useUploadBanner } from '../hooks/useUploadBanner';

interface EditProfileDialogProps {
  user: UserResponse;
  open: boolean;
  onClose: () => void;
}

const BIO_MAX = 160;

export const EditProfileDialog = ({ user, open, onClose }: EditProfileDialogProps) => {
  const [displayName, setDisplayName] = useState(user.display_name ?? '');
  const [bio, setBio] = useState(user.bio ?? '');
  const [location, setLocation] = useState(user.location ?? '');
  const [websiteUrl, setWebsiteUrl] = useState(user.website_url ?? '');
  const [birthday, setBirthday] = useState(user.birthday ?? '');
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [bannerFile, setBannerFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [bannerPreview, setBannerPreview] = useState<string | null>(null);

  const updateProfileMutation = useUpdateProfile();
  const uploadAvatarMutation = useUploadAvatar();
  const uploadBannerMutation = useUploadBanner();

  useEffect(() => {
    if (!open) return;
    setDisplayName(user.display_name ?? '');
    setBio(user.bio ?? '');
    setLocation(user.location ?? '');
    setWebsiteUrl(user.website_url ?? '');
    setBirthday(user.birthday ?? '');
    setAvatarFile(null);
    setBannerFile(null);
    setAvatarPreview(null);
    setBannerPreview(null);
  }, [open, user]);

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleBannerChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setBannerFile(file);
    setBannerPreview(URL.createObjectURL(file));
  };

  const isPending = updateProfileMutation.isPending || uploadAvatarMutation.isPending || uploadBannerMutation.isPending;
  const error = updateProfileMutation.error ?? uploadAvatarMutation.error ?? uploadBannerMutation.error;

  const handleSave = async () => {
    try {
      await updateProfileMutation.mutateAsync({
        display_name: displayName,
        bio,
        location,
        website_url: websiteUrl,
        birthday,
      });
      if (avatarFile) await uploadAvatarMutation.mutateAsync(avatarFile);
      if (bannerFile) await uploadBannerMutation.mutateAsync(bannerFile);
      onClose();
    } catch {
      // エラーは各ミューテーションのerrorステートで表示する
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="sm"
    >
      <DialogTitle data-testid="edit-profile-dialog">プロフィールを編集</DialogTitle>
      <DialogContent>
        <Box
          sx={{
            position: 'relative',
            height: 120,
            borderRadius: 1,
            bgcolor: 'action.hover',
            backgroundImage: `url(${bannerPreview ?? user.banner_url ?? ''})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            mb: -5,
          }}
        >
          <IconButton
            component="label"
            sx={{ position: 'absolute', top: 8, right: 8, bgcolor: 'background.paper' }}
            aria-label="バナー画像を変更"
          >
            <PhotoCameraIcon fontSize="small" />
            <input type="file" accept="image/jpeg,image/png,image/webp" hidden onChange={handleBannerChange} />
          </IconButton>
        </Box>
        <Box sx={{ position: 'relative', width: 72, ml: 2 }}>
          <Avatar src={avatarPreview ?? user.avatar_url ?? undefined} sx={{ width: 72, height: 72, border: '3px solid', borderColor: 'background.paper' }}>
            {(user.display_name ?? '?').charAt(0)}
          </Avatar>
          <IconButton
            component="label"
            size="small"
            sx={{ position: 'absolute', bottom: 0, right: 0, bgcolor: 'background.paper' }}
            aria-label="アバター画像を変更"
          >
            <PhotoCameraIcon fontSize="small" />
            <input type="file" accept="image/jpeg,image/png,image/webp" hidden onChange={handleAvatarChange} />
          </IconButton>
        </Box>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
          <TextField
            label="表示名"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value.slice(0, 50))}
            fullWidth
            slotProps={{ htmlInput: { 'data-testid': 'edit-profile-display-name' } }}
          />
          <TextField
            label="自己紹介"
            value={bio}
            onChange={(e) => setBio(e.target.value.slice(0, BIO_MAX))}
            multiline
            minRows={2}
            fullWidth
            helperText={`${bio.length} / ${BIO_MAX}`}
            slotProps={{ htmlInput: { 'data-testid': 'edit-profile-bio' } }}
          />
          <TextField
            label="場所"
            value={location}
            onChange={(e) => setLocation(e.target.value.slice(0, 30))}
            fullWidth
            slotProps={{ htmlInput: { 'data-testid': 'edit-profile-location' } }}
          />
          <TextField
            label="ウェブサイトURL"
            value={websiteUrl}
            onChange={(e) => setWebsiteUrl(e.target.value.slice(0, 100))}
            fullWidth
          />
          <TextField
            label="誕生日"
            type="date"
            value={birthday}
            onChange={(e) => setBirthday(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
            fullWidth
          />
          {error && (
            <Typography color="error" variant="body2">
              {getApiErrorMessage(error, '更新に失敗しました')}
            </Typography>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>キャンセル</Button>
        <Button
          onClick={() => void handleSave()}
          variant="contained"
          disabled={isPending}
          data-testid="edit-profile-save"
        >
          保存
        </Button>
      </DialogActions>
    </Dialog>
  );
};
