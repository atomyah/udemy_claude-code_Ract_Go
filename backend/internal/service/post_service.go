package service

import (
	"context"
	"errors"
	"mime/multipart"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrTooManyMedia    = errors.New("too many media files")
	ErrMixedMediaTypes = errors.New("cannot mix image and video")
)

// PostService は投稿のCRUD・タイムライン取得を担う
type PostService interface {
	CreatePost(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error)
	GetPost(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID) (*dto.PostResponse, error)
	UpdatePost(ctx context.Context, id, userID uuid.UUID, content string) (*dto.PostResponse, error)
	DeletePost(ctx context.Context, id, userID uuid.UUID) error
	GetHomeTimeline(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
	GetExploreTimeline(ctx context.Context, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
	GetUserPosts(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
	GetUserReplies(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
	GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)

	// CreateWithRefs は返信・リポストを含む投稿作成の共通ロジック。comment_service/repost_serviceから再利用される
	CreateWithRefs(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader, replyTo, repostOf *uuid.UUID, maxImages int, allowVideo bool) (*dto.PostResponse, error)
	// ToPostResponse はmodel.Postを集計値付きのdto.PostResponseに変換する
	ToPostResponse(ctx context.Context, post *model.Post, viewerID *uuid.UUID) (*dto.PostResponse, error)
}

type postService struct {
	postRepo     repository.PostRepository
	mediaRepo    repository.MediaRepository
	hashtagRepo  repository.HashtagRepository
	userRepo     repository.UserRepository
	likeRepo     repository.LikeRepository
	bookmarkRepo repository.BookmarkRepository
	storageSvc   StorageService
}

func NewPostService(
	postRepo repository.PostRepository,
	mediaRepo repository.MediaRepository,
	hashtagRepo repository.HashtagRepository,
	userRepo repository.UserRepository,
	likeRepo repository.LikeRepository,
	bookmarkRepo repository.BookmarkRepository,
	storageSvc StorageService,
) PostService {
	return &postService{
		postRepo:     postRepo,
		mediaRepo:    mediaRepo,
		hashtagRepo:  hashtagRepo,
		userRepo:     userRepo,
		likeRepo:     likeRepo,
		bookmarkRepo: bookmarkRepo,
		storageSvc:   storageSvc,
	}
}

func (s *postService) CreatePost(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error) {
	return s.CreateWithRefs(ctx, userID, content, files, nil, nil, 4, true)
}

func (s *postService) CreateWithRefs(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader, replyTo, repostOf *uuid.UUID, maxImages int, allowVideo bool) (*dto.PostResponse, error) {
	if err := validateMediaFiles(files, maxImages, allowVideo); err != nil {
		return nil, err
	}

	post := &model.Post{
		UserID:   userID,
		Content:  content,
		ReplyTo:  replyTo,
		RepostOf: repostOf,
	}
	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	if len(files) > 0 {
		media := make([]model.Media, 0, len(files))
		for i, fh := range files {
			url, mediaType, err := s.storageSvc.UploadPostMedia(ctx, "posts", fh)
			if err != nil {
				return nil, err
			}
			media = append(media, model.Media{PostID: post.ID, URL: url, Type: mediaType, SortOrder: i})
		}
		if err := s.mediaRepo.CreateBulk(ctx, media); err != nil {
			return nil, err
		}
		post.Media = media
	}

	if tags := utils.ExtractHashtags(content); len(tags) > 0 {
		hashtagIDs := make([]uuid.UUID, 0, len(tags))
		for _, tag := range tags {
			hashtag, err := s.hashtagRepo.FindOrCreate(ctx, tag)
			if err != nil {
				return nil, err
			}
			hashtagIDs = append(hashtagIDs, hashtag.ID)
		}
		if err := s.hashtagRepo.AttachToPost(ctx, post.ID, hashtagIDs); err != nil {
			return nil, err
		}
	}

	post, err := s.postRepo.FindByID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	return s.ToPostResponse(ctx, post, &userID)
}

func (s *postService) GetPost(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID) (*dto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.ToPostResponse(ctx, post, viewerID)
}

func (s *postService) UpdatePost(ctx context.Context, id, userID uuid.UUID, content string) (*dto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post.UserID != userID {
		return nil, ErrForbidden
	}

	post.Content = content
	post.IsEdited = true
	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, err
	}

	return s.ToPostResponse(ctx, post, &userID)
}

func (s *postService) DeletePost(ctx context.Context, id, userID uuid.UUID) error {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return ErrForbidden
	}
	return s.postRepo.SoftDelete(ctx, id)
}

// GetHomeTimeline はホームタイムラインを取得する。全ユーザーの投稿を新着順で返す（探索タイムラインと同一のクエリ）
func (s *postService) GetHomeTimeline(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.postRepo.GetExplore(ctx, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toPostListResponse(ctx, posts, next, &userID)
}

func (s *postService) GetExploreTimeline(ctx context.Context, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.postRepo.GetExplore(ctx, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toPostListResponse(ctx, posts, next, viewerID)
}

func (s *postService) GetUserPosts(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	user, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}

	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.postRepo.GetByUser(ctx, user.ID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toPostListResponse(ctx, posts, next, viewerID)
}

// GetUserReplies は指定ユーザーが他の投稿に行った返信の一覧を取得する
func (s *postService) GetUserReplies(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	user, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}

	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.postRepo.GetRepliesByUser(ctx, user.ID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toPostListResponse(ctx, posts, next, viewerID)
}

func (s *postService) GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.postRepo.GetComments(ctx, postID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toPostListResponse(ctx, posts, next, viewerID)
}

func (s *postService) toPostListResponse(ctx context.Context, posts []model.Post, next *time.Time, viewerID *uuid.UUID) (*dto.PostListResponse, error) {
	data := make([]dto.PostResponse, 0, len(posts))
	for i := range posts {
		resp, err := s.ToPostResponse(ctx, &posts[i], viewerID)
		if err != nil {
			return nil, err
		}
		data = append(data, *resp)
	}

	var nextCursor *string
	if next != nil {
		c := utils.FormatCursor(*next)
		nextCursor = &c
	}

	return &dto.PostListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}

func (s *postService) ToPostResponse(ctx context.Context, post *model.Post, viewerID *uuid.UUID) (*dto.PostResponse, error) {
	likesCount, err := s.likeRepo.CountByPost(ctx, post.ID)
	if err != nil {
		return nil, err
	}
	commentsCount, err := s.postRepo.CountComments(ctx, post.ID)
	if err != nil {
		return nil, err
	}
	repostsCount, err := s.postRepo.CountReposts(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	var isLiked, isBookmarked, isReposted bool
	if viewerID != nil {
		if isLiked, err = s.likeRepo.IsLiked(ctx, *viewerID, post.ID); err != nil {
			return nil, err
		}
		if isBookmarked, err = s.bookmarkRepo.IsBookmarked(ctx, *viewerID, post.ID); err != nil {
			return nil, err
		}
		if _, err := s.postRepo.FindActiveRepost(ctx, *viewerID, post.ID); err == nil {
			isReposted = true
		} else if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	}

	var replyTo *string
	var replyToUser *dto.UserInPost
	if post.ReplyTo != nil {
		s := post.ReplyTo.String()
		replyTo = &s
		if post.ReplyToPost != nil {
			u := toUserInPost(post.ReplyToPost.User)
			replyToUser = &u
		}
	}

	var repostOf *dto.PostSummary
	if post.RepostOfPost != nil {
		repostOf = toPostSummary(post.RepostOfPost)
	}

	return &dto.PostResponse{
		ID:            post.ID.String(),
		User:          toUserInPost(post.User),
		Content:       post.Content,
		Media:         toMediaResponses(post.Media),
		LikesCount:    likesCount,
		CommentsCount: commentsCount,
		RepostsCount:  repostsCount,
		IsLiked:       isLiked,
		IsBookmarked:  isBookmarked,
		IsReposted:    isReposted,
		IsEdited:      post.IsEdited,
		RepostOf:      repostOf,
		ReplyTo:       replyTo,
		ReplyToUser:   replyToUser,
		CreatedAt:     post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     post.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func toUserInPost(u model.User) dto.UserInPost {
	return dto.UserInPost{
		ID:          u.ID.String(),
		Handle:      u.Handle,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

func toMediaResponses(media []model.Media) []dto.MediaResponse {
	resp := make([]dto.MediaResponse, len(media))
	for i, m := range media {
		resp[i] = dto.MediaResponse{ID: m.ID.String(), URL: m.URL, Type: m.Type, SortOrder: m.SortOrder}
	}
	return resp
}

func toPostSummary(p *model.Post) *dto.PostSummary {
	return &dto.PostSummary{
		ID:        p.ID.String(),
		User:      toUserInPost(p.User),
		Content:   p.Content,
		Media:     toMediaResponses(p.Media),
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
}

// validateMediaFiles は投稿メディアの枚数・種別制約を検証する
func validateMediaFiles(files []*multipart.FileHeader, maxImages int, allowVideo bool) error {
	if len(files) == 0 {
		return nil
	}

	videoCount := 0
	for _, fh := range files {
		contentType := detectContentType(fh)
		switch {
		case allowedImageExtensions[contentType] != "":
			// 画像はOK
		case allowedVideoExtensions[contentType] != "":
			if !allowVideo {
				return ErrUnsupportedFileType
			}
			videoCount++
		default:
			return ErrUnsupportedFileType
		}
	}

	if videoCount > 0 {
		if videoCount > 1 || len(files) > 1 {
			return ErrMixedMediaTypes
		}
		return nil
	}

	if len(files) > maxImages {
		return ErrTooManyMedia
	}
	return nil
}
