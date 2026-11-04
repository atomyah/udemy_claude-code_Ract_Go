package service

import (
	"context"
	"errors"
	"mime/multipart"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/logger"
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
	// ToPostResponses は複数の投稿をまとめて変換する。集計値は件数によらず1クエリで取得する（N+1回避）
	ToPostResponses(ctx context.Context, posts []model.Post, viewerID *uuid.UUID) ([]dto.PostResponse, error)
}

type postService struct {
	postRepo        repository.PostRepository
	hashtagRepo     repository.HashtagRepository
	userRepo        repository.UserRepository
	likeRepo        repository.LikeRepository
	bookmarkRepo    repository.BookmarkRepository
	storageSvc      StorageService
	notificationSvc NotificationService
}

func NewPostService(
	postRepo repository.PostRepository,
	hashtagRepo repository.HashtagRepository,
	userRepo repository.UserRepository,
	likeRepo repository.LikeRepository,
	bookmarkRepo repository.BookmarkRepository,
	storageSvc StorageService,
	notificationSvc NotificationService,
) PostService {
	return &postService{
		postRepo:        postRepo,
		hashtagRepo:     hashtagRepo,
		userRepo:        userRepo,
		likeRepo:        likeRepo,
		bookmarkRepo:    bookmarkRepo,
		storageSvc:      storageSvc,
		notificationSvc: notificationSvc,
	}
}

func (s *postService) CreatePost(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error) {
	return s.CreateWithRefs(ctx, userID, content, files, nil, nil, 4, true)
}

func (s *postService) CreateWithRefs(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader, replyTo, repostOf *uuid.UUID, maxImages int, allowVideo bool) (*dto.PostResponse, error) {
	if err := validateMediaFiles(files, maxImages, allowVideo); err != nil {
		return nil, err
	}

	// 保存先パスに使うため投稿IDを先に決める。アップロードに失敗したら投稿は作らない
	postID := uuid.New()
	media, err := s.uploadPostMedia(ctx, postID, files)
	if err != nil {
		return nil, err
	}

	// 投稿とメディアは GORM の関連付け保存で同じトランザクションに入る
	post := &model.Post{
		ID:       postID,
		UserID:   userID,
		Content:  content,
		ReplyTo:  replyTo,
		RepostOf: repostOf,
		Media:    media,
	}
	if err := s.postRepo.Create(ctx, post); err != nil {
		s.deleteUploadedMedia(ctx, media)
		return nil, err
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

	if err := s.notifyMentions(ctx, post.ID, userID, content); err != nil {
		return nil, err
	}

	post, err = s.postRepo.FindByID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	return s.ToPostResponse(ctx, post, &userID)
}

// uploadPostMedia はファイルを順番にアップロードする。途中で失敗したらアップロード済みの分を削除する
func (s *postService) uploadPostMedia(ctx context.Context, postID uuid.UUID, files []*multipart.FileHeader) ([]model.Media, error) {
	media := make([]model.Media, 0, len(files))
	for i, fh := range files {
		objectPath, mediaType, err := s.storageSvc.UploadPostMedia(ctx, postID, fh)
		if err != nil {
			s.deleteUploadedMedia(ctx, media)
			return nil, err
		}
		media = append(media, model.Media{PostID: postID, StoragePath: objectPath, Type: mediaType, SortOrder: i})
	}
	return media, nil
}

func (s *postService) deleteUploadedMedia(ctx context.Context, media []model.Media) {
	if len(media) == 0 {
		return
	}
	objectPaths := make([]string, len(media))
	for i, m := range media {
		objectPaths[i] = m.StoragePath
	}
	s.storageSvc.DeleteObjects(ctx, objectPaths)
}

// notifyMentions は本文中の@ハンドルにメンション通知を作成する。
// 存在しないハンドルは無視し、自分自身への通知はNotificationService側でスキップされる。
func (s *postService) notifyMentions(ctx context.Context, postID, actorID uuid.UUID, content string) error {
	for _, handle := range utils.ExtractMentions(content) {
		user, err := s.userRepo.FindByHandle(ctx, handle)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err := s.notificationSvc.Notify(ctx, user.ID, actorID, "mention", &postID); err != nil {
			return err
		}
	}
	return nil
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
	data, err := s.ToPostResponses(ctx, posts, viewerID)
	if err != nil {
		return nil, err
	}

	var nextCursor *string
	if next != nil {
		c := utils.FormatCursor(*next)
		nextCursor = &c
	}

	return &dto.PostListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}

func (s *postService) ToPostResponse(ctx context.Context, post *model.Post, viewerID *uuid.UUID) (*dto.PostResponse, error) {
	resps, err := s.ToPostResponses(ctx, []model.Post{*post}, viewerID)
	if err != nil {
		return nil, err
	}
	return &resps[0], nil
}

func (s *postService) ToPostResponses(ctx context.Context, posts []model.Post, viewerID *uuid.UUID) ([]dto.PostResponse, error) {
	data := make([]dto.PostResponse, 0, len(posts))
	if len(posts) == 0 {
		return data, nil
	}

	ids := make([]uuid.UUID, len(posts))
	for i := range posts {
		ids[i] = posts[i].ID
	}
	stats, err := s.postRepo.FindStatsByIDs(ctx, ids, viewerID)
	if err != nil {
		return nil, err
	}

	for i := range posts {
		data = append(data, buildPostResponse(ctx, s.storageSvc, &posts[i], stats[posts[i].ID]))
	}
	return data, nil
}

// buildPostResponse は投稿と集計値からレスポンスを組み立てる（DBアクセスなし）
func buildPostResponse(ctx context.Context, signer MediaURLSigner, post *model.Post, stats repository.PostStats) dto.PostResponse {
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
		repostOf = toPostSummary(ctx, signer, post.RepostOfPost)
	}

	return dto.PostResponse{
		ID:            post.ID.String(),
		User:          toUserInPost(post.User),
		Content:       post.Content,
		Media:         toMediaResponses(ctx, signer, post.Media),
		LikesCount:    stats.LikesCount,
		CommentsCount: stats.CommentsCount,
		RepostsCount:  stats.RepostsCount,
		IsLiked:       stats.IsLiked,
		IsBookmarked:  stats.IsBookmarked,
		IsReposted:    stats.IsReposted,
		IsEdited:      post.IsEdited,
		RepostOf:      repostOf,
		ReplyTo:       replyTo,
		ReplyToUser:   replyToUser,
		CreatedAt:     post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     post.UpdatedAt.Format(time.RFC3339),
	}
}

func toUserInPost(u model.User) dto.UserInPost {
	return dto.UserInPost{
		ID:          u.ID.String(),
		Handle:      u.Handle,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

// toMediaResponses はメディアを閲覧URL付きのレスポンスに変換する。
// storage_path を持つ行は署名付きURLを発行し、公開URLで保存した既存行は url をそのまま返す。
// 署名はローカル処理のためメディア数だけ通信が発生することはない。
// 署名に失敗したメディアは一覧全体を失敗させず、ログに残してレスポンスから外す
func toMediaResponses(ctx context.Context, signer MediaURLSigner, media []model.Media) []dto.MediaResponse {
	resp := make([]dto.MediaResponse, 0, len(media))
	for _, m := range media {
		url := m.URL
		if m.StoragePath != "" {
			signed, err := signer.SignedURL(m.StoragePath)
			if err != nil {
				logger.FromContext(ctx).Warn().Err(err).Str("media_id", m.ID.String()).Msg("failed to sign media url")
				continue
			}
			url = signed
		}
		resp = append(resp, dto.MediaResponse{ID: m.ID.String(), URL: url, Type: m.Type, SortOrder: m.SortOrder})
	}
	return resp
}

func toPostSummary(ctx context.Context, signer MediaURLSigner, p *model.Post) *dto.PostSummary {
	return &dto.PostSummary{
		ID:        p.ID.String(),
		User:      toUserInPost(p.User),
		Content:   p.Content,
		Media:     toMediaResponses(ctx, signer, p.Media),
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
}

// validateMediaFiles は投稿メディアの形式・サイズ・枚数を検証する。
// 1 件でも不正なら 1 件もアップロードしないよう、アップロードの前に全ファイルを確認する
func validateMediaFiles(files []*multipart.FileHeader, maxImages int, allowVideo bool) error {
	if len(files) == 0 {
		return nil
	}

	videoCount := 0
	for _, fh := range files {
		mediaType, maxSize, _ := postMediaRule(detectContentType(fh))
		switch mediaType {
		case mediaTypeImage:
			// 画像はOK
		case mediaTypeVideo:
			if !allowVideo {
				return ErrUnsupportedFileType
			}
			videoCount++
		default:
			return ErrUnsupportedFileType
		}
		if fh.Size > maxSize {
			return ErrFileTooLarge
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
