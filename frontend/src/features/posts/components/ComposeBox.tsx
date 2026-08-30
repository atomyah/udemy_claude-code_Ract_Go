import CloseIcon from '@mui/icons-material/Close';
import ImageIcon from '@mui/icons-material/Image';
import VideocamIcon from '@mui/icons-material/Videocam';
import { Avatar, Box, Button, IconButton, LinearProgress, Stack, TextField, Typography } from '@mui/material';
import { useEffect, useMemo, useRef, useState, type ChangeEvent, type RefObject } from 'react';
import { getApiErrorMessage } from '../../../utils/apiError';
import { useAuth } from '../../auth/AuthContext';
import type { CreatePostPayload } from '../api';

const MAX_LENGTH = 280;
const MAX_IMAGES = 4;
const MAX_IMAGE_SIZE = 5 * 1024 * 1024;
const MAX_VIDEO_SIZE = 100 * 1024 * 1024;
const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const VIDEO_TYPES = ['video/mp4', 'video/quicktime'];

interface ComposeBoxProps {
  placeholder: string;
  submitLabel: string;
  isPending: boolean;
  error: unknown;
  onSubmit: (payload: CreatePostPayload, reset: () => void) => void;
  /** 返信欄など、一覧に埋め込む用途で余白を詰めて表示する */
  compact?: boolean;
  inputRef?: RefObject<HTMLTextAreaElement | null>;
  /** E2Eテスト用のdata-testid接頭辞（例: post-form → post-form-input / post-form-submit） */
  testIdPrefix?: string;
}

export const ComposeBox = ({
  placeholder,
  submitLabel,
  isPending,
  error,
  onSubmit,
  compact = false,
  inputRef,
  testIdPrefix = 'compose',
}: ComposeBoxProps) => {
  const { currentUser } = useAuth();
  const [content, setContent] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [fileError, setFileError] = useState<string | null>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const videoInputRef = useRef<HTMLInputElement>(null);

  const previewUrls = useMemo(() => files.map((file) => URL.createObjectURL(file)), [files]);
  useEffect(() => () => previewUrls.forEach((url) => URL.revokeObjectURL(url)), [previewUrls]);

  const hasVideo = files.some((file) => file.type.startsWith('video/'));

  const handleImageSelect = (event: ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(event.target.files ?? []);
    event.target.value = '';
    if (selected.length === 0) return;
    if (hasVideo) {
      setFileError('画像と動画は同時に添付できません');
      return;
    }
    const invalid = selected.find((file) => !IMAGE_TYPES.includes(file.type) || file.size > MAX_IMAGE_SIZE);
    if (invalid) {
      setFileError('画像はJPEG/PNG/WebP形式、5MB以内にしてください');
      return;
    }
    setFileError(null);
    setFiles((prev) => [...prev, ...selected].slice(0, MAX_IMAGES));
  };

  const handleVideoSelect = (event: ChangeEvent<HTMLInputElement>) => {
    const selected = event.target.files?.[0];
    event.target.value = '';
    if (!selected) return;
    if (files.length > 0) {
      setFileError('画像と動画は同時に添付できません');
      return;
    }
    if (!VIDEO_TYPES.includes(selected.type) || selected.size > MAX_VIDEO_SIZE) {
      setFileError('動画はMP4/MOV形式、100MB以内にしてください');
      return;
    }
    setFileError(null);
    setFiles([selected]);
  };

  const removeFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index));
    setFileError(null);
  };

  const canSubmit = (content.trim().length > 0 || files.length > 0) && !isPending;

  const handleSubmit = () => {
    onSubmit({ content, media: files }, () => {
      setContent('');
      setFiles([]);
    });
  };

  return (
    <Box
      data-testid={testIdPrefix}
      sx={{ px: 2, py: compact ? 1 : 2, borderBottom: '1px solid', borderColor: 'divider' }}
    >
      <Stack direction="row" spacing={compact ? 1 : 1.5}>
        <Avatar
          src={currentUser?.avatar_url ?? undefined}
          sx={compact ? { width: 36, height: 36 } : undefined}
        >
          {currentUser?.display_name?.charAt(0)}
        </Avatar>
        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
          <TextField
            fullWidth
            multiline
            minRows={compact ? 1 : 2}
            placeholder={placeholder}
            value={content}
            onChange={(e) => setContent(e.target.value.slice(0, MAX_LENGTH))}
            variant="standard"
            inputRef={inputRef}
            slotProps={{
              input: { disableUnderline: true },
              htmlInput: { 'data-testid': `${testIdPrefix}-input` },
            }}
          />

          {files.length > 0 && (
            <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap' }}>
              {files.map((file, index) => (
                <Box key={index} sx={{ position: 'relative', width: 96, height: 96 }}>
                  {file.type.startsWith('video/') ? (
                    <Box
                      component="video"
                      src={previewUrls[index]}
                      sx={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: 1, bgcolor: 'black' }}
                    />
                  ) : (
                    <Box
                      component="img"
                      src={previewUrls[index]}
                      alt=""
                      sx={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: 1 }}
                    />
                  )}
                  <IconButton
                    size="small"
                    onClick={() => removeFile(index)}
                    sx={{ position: 'absolute', top: -8, right: -8, bgcolor: 'background.paper', boxShadow: 1 }}
                  >
                    <CloseIcon fontSize="small" />
                  </IconButton>
                </Box>
              ))}
            </Stack>
          )}

          {isPending && <LinearProgress sx={{ mt: 1 }} />}

          {Boolean(fileError || error) && (
            <Typography color="error" variant="body2" sx={{ mt: 1 }} data-testid={`${testIdPrefix}-error`}>
              {fileError ?? getApiErrorMessage(error, '送信に失敗しました')}
            </Typography>
          )}

          <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mt: compact ? 0 : 1 }}>
            <Stack direction="row">
              <IconButton
                aria-label="画像を添付"
                size={compact ? 'small' : 'medium'}
                onClick={() => imageInputRef.current?.click()}
                disabled={hasVideo || files.length >= MAX_IMAGES}
              >
                <ImageIcon fontSize={compact ? 'small' : 'medium'} />
              </IconButton>
              <IconButton
                aria-label="動画を添付"
                size={compact ? 'small' : 'medium'}
                onClick={() => videoInputRef.current?.click()}
                disabled={files.length > 0}
              >
                <VideocamIcon fontSize={compact ? 'small' : 'medium'} />
              </IconButton>
              <input
                ref={imageInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                multiple
                hidden
                onChange={handleImageSelect}
              />
              <input ref={videoInputRef} type="file" accept="video/mp4,video/quicktime" hidden onChange={handleVideoSelect} />
            </Stack>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <Typography variant="caption" color={content.length >= MAX_LENGTH ? 'error' : 'text.secondary'}>
                {content.length} / {MAX_LENGTH}
              </Typography>
              <Button
                variant="contained"
                size={compact ? 'small' : 'medium'}
                disabled={!canSubmit}
                onClick={handleSubmit}
                data-testid={`${testIdPrefix}-submit`}
              >
                {submitLabel}
              </Button>
            </Stack>
          </Stack>
        </Box>
      </Stack>
    </Box>
  );
};
