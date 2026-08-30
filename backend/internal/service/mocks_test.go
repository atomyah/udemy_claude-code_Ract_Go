package service

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// リポジトリのモック（testify/mock）
// サービス層のユニットテストではDBを使わず、これらのモックを注入する
// ============================================================

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	return userOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	return userOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserRepo) FindByHandle(ctx context.Context, handle string) (*model.User, error) {
	args := m.Called(ctx, handle)
	return userOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserRepo) FindAll(ctx context.Context, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	args := m.Called(ctx, cursor, limit)
	return args.Get(0).([]model.User), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) ExistsByHandle(ctx context.Context, handle string) (bool, error) {
	args := m.Called(ctx, handle)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) Suspend(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockUserRepo) Unsuspend(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockPostRepo struct{ mock.Mock }

func (m *mockPostRepo) Create(ctx context.Context, post *model.Post) error {
	return m.Called(ctx, post).Error(0)
}

func (m *mockPostRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	args := m.Called(ctx, id)
	return postOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostRepo) Update(ctx context.Context, post *model.Post) error {
	return m.Called(ctx, post).Error(0)
}

func (m *mockPostRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockPostRepo) GetExplore(ctx context.Context, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockPostRepo) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockPostRepo) GetRepliesByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockPostRepo) GetComments(ctx context.Context, postID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, postID, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockPostRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Post, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]model.Post), args.Error(1)
}

func (m *mockPostRepo) CountComments(ctx context.Context, postID uuid.UUID) (int64, error) {
	args := m.Called(ctx, postID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockPostRepo) CountReposts(ctx context.Context, postID uuid.UUID) (int64, error) {
	args := m.Called(ctx, postID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockPostRepo) FindActiveRepost(ctx context.Context, userID, postID uuid.UUID) (*model.Post, error) {
	args := m.Called(ctx, userID, postID)
	return postOrNil(args.Get(0)), args.Error(1)
}

type mockLikeRepo struct{ mock.Mock }

func (m *mockLikeRepo) Create(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockLikeRepo) Delete(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockLikeRepo) CountByPost(ctx context.Context, postID uuid.UUID) (int64, error) {
	args := m.Called(ctx, postID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockLikeRepo) IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, postID)
	return args.Bool(0), args.Error(1)
}

type mockBookmarkRepo struct{ mock.Mock }

func (m *mockBookmarkRepo) Create(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockBookmarkRepo) Delete(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockBookmarkRepo) IsBookmarked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, postID)
	return args.Bool(0), args.Error(1)
}

func (m *mockBookmarkRepo) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

type mockFollowRepo struct{ mock.Mock }

func (m *mockFollowRepo) Create(ctx context.Context, follow *model.Follow) error {
	return m.Called(ctx, follow).Error(0)
}

func (m *mockFollowRepo) Delete(ctx context.Context, followerID, followingID uuid.UUID) error {
	return m.Called(ctx, followerID, followingID).Error(0)
}

func (m *mockFollowRepo) Exists(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	args := m.Called(ctx, followerID, followingID)
	return args.Bool(0), args.Error(1)
}

func (m *mockFollowRepo) CountFollowers(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockFollowRepo) CountFollowing(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockFollowRepo) GetFollowers(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.User), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockFollowRepo) GetFollowing(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.User), timeOrNil(args.Get(1)), args.Error(2)
}

type mockNotificationRepo struct{ mock.Mock }

func (m *mockNotificationRepo) Create(ctx context.Context, notification *model.Notification) error {
	return m.Called(ctx, notification).Error(0)
}

func (m *mockNotificationRepo) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Notification, *time.Time, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]model.Notification), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockNotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockNotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return int64(args.Int(0)), args.Error(1)
}

type mockMediaRepo struct{ mock.Mock }

func (m *mockMediaRepo) CreateBulk(ctx context.Context, media []model.Media) error {
	return m.Called(ctx, media).Error(0)
}

func (m *mockMediaRepo) FindByPostID(ctx context.Context, postID uuid.UUID) ([]model.Media, error) {
	args := m.Called(ctx, postID)
	return args.Get(0).([]model.Media), args.Error(1)
}

func (m *mockMediaRepo) FindByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.Media, error) {
	args := m.Called(ctx, postIDs)
	return args.Get(0).(map[uuid.UUID][]model.Media), args.Error(1)
}

type mockHashtagRepo struct{ mock.Mock }

func (m *mockHashtagRepo) FindOrCreate(ctx context.Context, name string) (*model.Hashtag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Hashtag), args.Error(1)
}

func (m *mockHashtagRepo) AttachToPost(ctx context.Context, postID uuid.UUID, hashtagIDs []uuid.UUID) error {
	return m.Called(ctx, postID, hashtagIDs).Error(0)
}

func (m *mockHashtagRepo) FindByName(ctx context.Context, name string) (*model.Hashtag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Hashtag), args.Error(1)
}

func (m *mockHashtagRepo) GetPostIDsByHashtag(ctx context.Context, hashtagID uuid.UUID, cursor *time.Time, limit int) ([]uuid.UUID, error) {
	args := m.Called(ctx, hashtagID, cursor, limit)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// ============================================================
// サービスのモック（他サービスに依存するサービスのテスト用）
// ============================================================

type mockNotificationSvc struct{ mock.Mock }

func (m *mockNotificationSvc) Notify(ctx context.Context, recipientID, actorID uuid.UUID, notifType string, postID *uuid.UUID) error {
	return m.Called(ctx, recipientID, actorID, notifType, postID).Error(0)
}

func (m *mockNotificationSvc) GetNotifications(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.NotificationListResponse, error) {
	args := m.Called(ctx, userID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.NotificationListResponse), args.Error(1)
}

func (m *mockNotificationSvc) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

// mockPostSvc はPostServiceのモック。comment/repost/bookmarkサービスのテストで使う
type mockPostSvc struct{ mock.Mock }

func (m *mockPostSvc) CreatePost(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error) {
	args := m.Called(ctx, userID, content, files)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) GetPost(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID) (*dto.PostResponse, error) {
	args := m.Called(ctx, id, viewerID)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) UpdatePost(ctx context.Context, id, userID uuid.UUID, content string) (*dto.PostResponse, error) {
	args := m.Called(ctx, id, userID, content)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) DeletePost(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}

func (m *mockPostSvc) GetHomeTimeline(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) GetExploreTimeline(ctx context.Context, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) GetUserPosts(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, handle, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) GetUserReplies(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, handle, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, postID, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) CreateWithRefs(ctx context.Context, userID uuid.UUID, content string, files []*multipart.FileHeader, replyTo, repostOf *uuid.UUID, maxImages int, allowVideo bool) (*dto.PostResponse, error) {
	args := m.Called(ctx, userID, content, files, replyTo, repostOf, maxImages, allowVideo)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockPostSvc) ToPostResponse(ctx context.Context, post *model.Post, viewerID *uuid.UUID) (*dto.PostResponse, error) {
	args := m.Called(ctx, post, viewerID)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

type mockStorageSvc struct{ mock.Mock }

func (m *mockStorageSvc) UploadImage(ctx context.Context, folder string, fh *multipart.FileHeader) (string, error) {
	args := m.Called(ctx, folder, fh)
	return args.String(0), args.Error(1)
}

func (m *mockStorageSvc) UploadPostMedia(ctx context.Context, folder string, fh *multipart.FileHeader) (string, string, error) {
	args := m.Called(ctx, folder, fh)
	return args.String(0), args.String(1), args.Error(2)
}

// ============================================================
// ヘルパー
// ============================================================

func userOrNil(v interface{}) *model.User {
	if v == nil {
		return nil
	}
	return v.(*model.User)
}

func postOrNil(v interface{}) *model.Post {
	if v == nil {
		return nil
	}
	return v.(*model.Post)
}

func timeOrNil(v interface{}) *time.Time {
	if v == nil {
		return nil
	}
	return v.(*time.Time)
}

func postResponseOrNil(v interface{}) *dto.PostResponse {
	if v == nil {
		return nil
	}
	return v.(*dto.PostResponse)
}

func postListOrNil(v interface{}) *dto.PostListResponse {
	if v == nil {
		return nil
	}
	return v.(*dto.PostListResponse)
}
