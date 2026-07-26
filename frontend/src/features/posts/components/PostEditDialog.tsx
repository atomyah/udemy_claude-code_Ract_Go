import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField, Typography } from '@mui/material';
import { useState } from 'react';
import type { PostResponse } from '../../../api/types';
import { getApiErrorMessage } from '../../../utils/apiError';
import { useUpdatePost } from '../hooks/useUpdatePost';

interface PostEditDialogProps {
  post: PostResponse;
  open: boolean;
  onClose: () => void;
}

const MAX_LENGTH = 280;

export const PostEditDialog = ({ post, open, onClose }: PostEditDialogProps) => {
  const [content, setContent] = useState(post.content ?? '');
  const { mutate, isPending, error } = useUpdatePost(post.id ?? '');

  const handleSave = () => {
    mutate({ content }, { onSuccess: onClose });
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm" onClick={(e) => e.stopPropagation()}>
      <DialogTitle>投稿を編集</DialogTitle>
      <DialogContent>
        <TextField
          autoFocus
          multiline
          minRows={3}
          fullWidth
          value={content}
          onChange={(e) => setContent(e.target.value.slice(0, MAX_LENGTH))}
          helperText={`${content.length} / ${MAX_LENGTH}`}
        />
        {error && (
          <Typography color="error" variant="body2" sx={{ mt: 1 }}>
            {getApiErrorMessage(error, '投稿の更新に失敗しました')}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>キャンセル</Button>
        <Button onClick={handleSave} variant="contained" disabled={content.trim().length === 0 || isPending}>
          保存
        </Button>
      </DialogActions>
    </Dialog>
  );
};
