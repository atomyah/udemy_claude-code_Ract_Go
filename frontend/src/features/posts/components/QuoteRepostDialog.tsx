import CloseIcon from '@mui/icons-material/Close';
import ImageIcon from '@mui/icons-material/Image';
import {
  Avatar,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useMemo, useRef, useState, type ChangeEvent } from 'react';
import type { PostResponse } from '../../../api/types';
import { getApiErrorMessage } from '../../../utils/apiError';
import { formatRelativeTime } from '../../../utils/formatDate';
import { useRepost } from '../hooks/useRepost';

interface QuoteRepostDialogProps {
  post: PostResponse;
  open: boolean;
  onClose: () => void;
}

const MAX_LENGTH = 280;
const MAX_IMAGES = 2;
const MAX_IMAGE_SIZE = 5 * 1024 * 1024;
const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];

export const QuoteRepostDialog = ({ post, open, onClose }: QuoteRepostDialogProps) => {
  const [content, setContent] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [fileError, setFileError] = useState<string | null>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const repostMutation = useRepost(post.id ?? '');

  const previewUrls = useMemo(() => files.map((file) => URL.createObjectURL(file)), [files]);
  const user = post.user;

  const handleImageSelect = (event: ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(event.target.files ?? []);
    event.target.value = '';
    if (selected.length === 0) return;
    const invalid = selected.find((file) => !IMAGE_TYPES.includes(file.type) || file.size > MAX_IMAGE_SIZE);
    if (invalid) {
      setFileError('画像はJPEG/PNG/WebP形式、5MB以内にしてください');
      return;
    }
    setFileError(null);
    setFiles((prev) => [...prev, ...selected].slice(0, MAX_IMAGES));
  };

  const handleClose = () => {
    setContent('');
    setFiles([]);
    setFileError(null);
    onClose();
  };

  const handleSubmit = () => {
    repostMutation.mutate(
      { isReposted: false, content, media: files },
      { onSuccess: handleClose },
    );
  };

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm" onClick={(e) => e.stopPropagation()}>
      <DialogTitle>引用リポスト</DialogTitle>
      <DialogContent>
        <TextField
          autoFocus
          multiline
          minRows={2}
          fullWidth
          placeholder="コメントを追加"
          value={content}
          onChange={(e) => setContent(e.target.value.slice(0, MAX_LENGTH))}
          variant="standard"
        />

        {files.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap' }}>
            {files.map((_file, index) => (
              <Box key={index} sx={{ position: 'relative', width: 96, height: 96 }}>
                <Box
                  component="img"
                  src={previewUrls[index]}
                  alt=""
                  sx={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: 1 }}
                />
                <IconButton
                  size="small"
                  onClick={() => setFiles((prev) => prev.filter((_, i) => i !== index))}
                  sx={{ position: 'absolute', top: -8, right: -8, bgcolor: 'background.paper', boxShadow: 1 }}
                >
                  <CloseIcon fontSize="small" />
                </IconButton>
              </Box>
            ))}
          </Stack>
        )}

        <Box sx={{ mt: 1.5, p: 1.5, border: '1px solid', borderColor: 'divider', borderRadius: 2 }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Avatar src={user?.avatar_url ?? undefined} sx={{ width: 24, height: 24 }}>
              {(user?.display_name ?? '?').charAt(0)}
            </Avatar>
            <Typography variant="body2" sx={{ fontWeight: 700 }}>
              {user?.display_name}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              @{user?.handle}・{formatRelativeTime(post.created_at ?? '')}
            </Typography>
          </Stack>
          <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', mt: 0.5 }}>
            {post.content}
          </Typography>
        </Box>

        <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mt: 1.5 }}>
          <IconButton
            aria-label="画像を添付"
            onClick={() => imageInputRef.current?.click()}
            disabled={files.length >= MAX_IMAGES}
          >
            <ImageIcon />
          </IconButton>
          <input
            ref={imageInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            multiple
            hidden
            onChange={handleImageSelect}
          />
          <Typography variant="caption" color={content.length >= MAX_LENGTH ? 'error' : 'text.secondary'}>
            {content.length} / {MAX_LENGTH}
          </Typography>
        </Stack>

        {(fileError || repostMutation.isError) && (
          <Typography color="error" variant="body2" sx={{ mt: 1 }}>
            {fileError ?? getApiErrorMessage(repostMutation.error, 'リポストに失敗しました')}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>キャンセル</Button>
        <Button onClick={handleSubmit} variant="contained" disabled={repostMutation.isPending}>
          引用リポスト
        </Button>
      </DialogActions>
    </Dialog>
  );
};
