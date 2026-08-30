import BookmarkIcon from '@mui/icons-material/Bookmark';
import BookmarkBorderIcon from '@mui/icons-material/BookmarkBorder';
import ChatBubbleOutlineIcon from '@mui/icons-material/ChatBubbleOutlineOutlined';
import FavoriteIcon from '@mui/icons-material/Favorite';
import FavoriteBorderIcon from '@mui/icons-material/FavoriteBorder';
import MoreHorizIcon from '@mui/icons-material/MoreHoriz';
import RepeatIcon from '@mui/icons-material/Repeat';
import ShareIcon from '@mui/icons-material/Share';
import { Avatar, Box, Card, CardContent, IconButton, Menu, MenuItem, Stack, Typography } from '@mui/material';
import { type MouseEvent, useState } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import type { PostResponse, PostSummary } from '../../api/types';
import { useAuth } from '../../features/auth/AuthContext';
import { PostEditDialog } from '../../features/posts/components/PostEditDialog';
import { QuoteRepostDialog } from '../../features/posts/components/QuoteRepostDialog';
import { useBookmark } from '../../features/posts/hooks/useBookmark';
import { useDeletePost } from '../../features/posts/hooks/useDeletePost';
import { useLike } from '../../features/posts/hooks/useLike';
import { useRepost } from '../../features/posts/hooks/useRepost';
import { UserHoverCard } from '../../features/profile/components/UserHoverCard';
import { formatRelativeTime } from '../../utils/formatDate';
import { LinkifiedText } from '../LinkifiedText';
import { ActionButton } from './ActionButton';
import { MediaGrid } from './MediaGrid';

interface PostCardProps {
  post: PostResponse;
  /** 返信ボタンの挙動を上書きする（投稿詳細ページでは返信欄にフォーカスする） */
  onCommentClick?: () => void;
}

const QuotedPost = ({ quoted }: { quoted: PostSummary }) => {
  const navigate = useNavigate();
  const user = quoted.user;

  return (
    <Box
      onClick={(e: MouseEvent) => {
        e.stopPropagation();
        navigate(`/posts/${quoted.id ?? ''}`);
      }}
      sx={{
        mt: 1,
        p: 1.5,
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 2,
        cursor: 'pointer',
      }}
    >
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Avatar src={user?.avatar_url ?? undefined} sx={{ width: 20, height: 20 }}>
          {(user?.display_name ?? '?').charAt(0)}
        </Avatar>
        <Typography variant="body2" sx={{ fontWeight: 700 }}>
          {user?.display_name}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          @{user?.handle}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          ・{formatRelativeTime(quoted.created_at ?? '')}
        </Typography>
      </Stack>
      <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', mt: 0.5 }}>
        <LinkifiedText text={quoted.content ?? ''} />
      </Typography>
      {(quoted.media ?? []).length > 0 && <MediaGrid media={quoted.media ?? []} />}
    </Box>
  );
};

export const PostCard = ({ post, onCommentClick }: PostCardProps) => {
  const { currentUser } = useAuth();
  const navigate = useNavigate();
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [repostMenuAnchor, setRepostMenuAnchor] = useState<HTMLElement | null>(null);
  const [quoteOpen, setQuoteOpen] = useState(false);

  const postId = post.id ?? '';
  const user = post.user;

  const likeMutation = useLike(postId);
  const repostMutation = useRepost(postId);
  const bookmarkMutation = useBookmark(postId);
  const deletePostMutation = useDeletePost();

  const isOwn = currentUser?.id !== undefined && currentUser.id === user?.id;

  const goToDetail = () => navigate(`/posts/${postId}`);

  const stop = (event: MouseEvent, action: () => void) => {
    event.stopPropagation();
    action();
  };

  return (
    <Card
      variant="outlined"
      onClick={goToDetail}
      data-testid="post-card"
      data-post-id={postId}
      sx={{ borderRadius: 0, borderLeft: 0, borderRight: 0, borderTop: 0, cursor: 'pointer' }}
    >
      <CardContent>
        <Stack direction="row" spacing={1.5}>
          <UserHoverCard handle={user?.handle ?? ''} sx={{ alignSelf: 'flex-start' }}>
            <Avatar
              component={RouterLink}
              to={`/${user?.handle ?? ''}`}
              src={user?.avatar_url ?? undefined}
              onClick={(e) => stop(e, () => {})}
            >
              {(user?.display_name ?? '?').charAt(0)}
            </Avatar>
          </UserHoverCard>
          <Box sx={{ flexGrow: 1, minWidth: 0 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline', flexWrap: 'wrap' }}>
              <UserHoverCard handle={user?.handle ?? ''}>
                <Typography
                  component={RouterLink}
                  to={`/${user?.handle ?? ''}`}
                  onClick={(e) => stop(e, () => {})}
                  variant="subtitle2"
                  sx={{ fontWeight: 700, color: 'text.primary', textDecoration: 'none' }}
                >
                  {user?.display_name}
                </Typography>
              </UserHoverCard>
              <Typography variant="body2" color="text.secondary">
                @{user?.handle}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                ・{formatRelativeTime(post.created_at ?? '')}
              </Typography>
              {post.is_edited && (
                <Typography variant="caption" color="text.secondary">
                  （編集済み）
                </Typography>
              )}
              <Box sx={{ flexGrow: 1 }} />
              <IconButton
                size="small"
                aria-label="その他のオプション"
                data-testid="post-menu-button"
                onClick={(e) => stop(e, () => setMenuAnchor(e.currentTarget))}
              >
                <MoreHorizIcon fontSize="small" />
              </IconButton>
            </Stack>

            {post.reply_to_user && (
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
                返信先:{' '}
                <Box
                  component={RouterLink}
                  to={`/${post.reply_to_user.handle ?? ''}`}
                  onClick={(e: MouseEvent) => stop(e, () => {})}
                  sx={{ color: 'primary.main', textDecoration: 'none' }}
                >
                  @{post.reply_to_user.handle}
                </Box>
                さん
              </Typography>
            )}

            {(post.content ?? '').length > 0 && (
              <Typography
                variant="body1"
                data-testid="post-content"
                sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', mt: 0.5 }}
              >
                <LinkifiedText text={post.content ?? ''} />
              </Typography>
            )}

            {(post.media ?? []).length > 0 && <MediaGrid media={post.media ?? []} />}
            {post.repost_of && <QuotedPost quoted={post.repost_of} />}

            <Stack
              direction="row"
              sx={{ mt: 1, maxWidth: 425, justifyContent: 'space-between' }}
              onClick={(e) => e.stopPropagation()}
            >
              <ActionButton
                icon={<ChatBubbleOutlineIcon fontSize="small" />}
                count={post.comments_count ?? 0}
                label="コメント"
                testIdPrefix="post-comment"
                onClick={onCommentClick ?? goToDetail}
              />
              <ActionButton
                icon={<RepeatIcon fontSize="small" />}
                count={post.reposts_count ?? 0}
                active={post.is_reposted}
                label="リポスト"
                testIdPrefix="post-repost"
                onClick={(e) =>
                  post.is_reposted
                    ? repostMutation.mutate({ isReposted: true })
                    : setRepostMenuAnchor(e.currentTarget as HTMLElement)
                }
              />
              <ActionButton
                icon={post.is_liked ? <FavoriteIcon fontSize="small" /> : <FavoriteBorderIcon fontSize="small" />}
                count={post.likes_count ?? 0}
                active={post.is_liked}
                label="いいね"
                testIdPrefix="post-like"
                onClick={() => likeMutation.mutate(post.is_liked ?? false)}
              />
              <IconButton
                size="small"
                aria-label="ブックマーク"
                data-testid="post-bookmark-button"
                color={post.is_bookmarked ? 'primary' : 'default'}
                onClick={() => bookmarkMutation.mutate(post.is_bookmarked ?? false)}
              >
                {post.is_bookmarked ? <BookmarkIcon fontSize="small" /> : <BookmarkBorderIcon fontSize="small" />}
              </IconButton>
              <IconButton
                size="small"
                aria-label="共有"
                onClick={() => {
                  const url = `${window.location.origin}/posts/${postId}`;
                  if (navigator.share) {
                    navigator.share({ url }).catch(() => {});
                  } else {
                    navigator.clipboard.writeText(url).catch(() => {});
                  }
                }}
              >
                <ShareIcon fontSize="small" />
              </IconButton>
            </Stack>
          </Box>
        </Stack>
      </CardContent>

      <Menu anchorEl={menuAnchor} open={Boolean(menuAnchor)} onClose={() => setMenuAnchor(null)}>
        {isOwn
          ? [
              <MenuItem
                key="edit"
                data-testid="post-menu-edit"
                onClick={(e) =>
                  stop(e, () => {
                    setEditOpen(true);
                    setMenuAnchor(null);
                  })
                }
              >
                編集
              </MenuItem>,
              <MenuItem
                key="delete"
                data-testid="post-menu-delete"
                onClick={(e) =>
                  stop(e, () => {
                    deletePostMutation.mutate(postId);
                    setMenuAnchor(null);
                  })
                }
              >
                削除
              </MenuItem>,
            ]
          : [
              <MenuItem key="hide" onClick={(e) => stop(e, () => setMenuAnchor(null))}>
                非表示
              </MenuItem>,
              <MenuItem key="report" onClick={(e) => stop(e, () => setMenuAnchor(null))}>
                報告
              </MenuItem>,
            ]}
      </Menu>

      {isOwn && (
        <Box onClick={(e) => e.stopPropagation()}>
          <PostEditDialog post={post} open={editOpen} onClose={() => setEditOpen(false)} />
        </Box>
      )}

      <Menu
        anchorEl={repostMenuAnchor}
        open={Boolean(repostMenuAnchor)}
        onClose={() => setRepostMenuAnchor(null)}
        onClick={(e) => e.stopPropagation()}
      >
        <MenuItem
          onClick={() => {
            repostMutation.mutate({ isReposted: false });
            setRepostMenuAnchor(null);
          }}
        >
          リポスト
        </MenuItem>
        <MenuItem
          onClick={() => {
            setQuoteOpen(true);
            setRepostMenuAnchor(null);
          }}
        >
          引用リポスト
        </MenuItem>
      </Menu>

      <Box onClick={(e) => e.stopPropagation()}>
        <QuoteRepostDialog post={post} open={quoteOpen} onClose={() => setQuoteOpen(false)} />
      </Box>
    </Card>
  );
};
