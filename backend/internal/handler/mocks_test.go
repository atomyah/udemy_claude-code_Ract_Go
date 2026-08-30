package handler

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// サービス層・リポジトリ層のモック（ハンドラーの統合テスト用）
// ============================================================

type mockAuthSvc struct{ mock.Mock }

func (m *mockAuthSvc) Register(ctx context.Context, req *dto.RegisterRequest) (*service.AuthResult, error) {
	args := m.Called(ctx, req)
	return authResultOrNil(args.Get(0)), args.Error(1)
}

func (m *mockAuthSvc) Login(ctx context.Context, req *dto.LoginRequest) (*service.AuthResult, error) {
	args := m.Called(ctx, req)
	return authResultOrNil(args.Get(0)), args.Error(1)
}

func (m *mockAuthSvc) RefreshAccessToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RefreshResponse), args.Error(1)
}

func (m *mockAuthSvc) LoginWithGoogle(ctx context.Context, idToken string) (*service.AuthResult, error) {
	args := m.Called(ctx, idToken)
	return authResultOrNil(args.Get(0)), args.Error(1)
}

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

type mockCommentSvc struct{ mock.Mock }

func (m *mockCommentSvc) CreateComment(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error) {
	args := m.Called(ctx, userID, postID, content, files)
	return postResponseOrNil(args.Get(0)), args.Error(1)
}

type mockLikeSvc struct{ mock.Mock }

func (m *mockLikeSvc) Like(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error) {
	args := m.Called(ctx, userID, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LikeResponse), args.Error(1)
}

func (m *mockLikeSvc) Unlike(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error) {
	args := m.Called(ctx, userID, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LikeResponse), args.Error(1)
}

type mockRepostSvc struct{ mock.Mock }

func (m *mockRepostSvc) Repost(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.RepostResponse, error) {
	args := m.Called(ctx, userID, postID, content, files)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RepostResponse), args.Error(1)
}

func (m *mockRepostSvc) Unrepost(ctx context.Context, userID, postID uuid.UUID) (*dto.RepostResponse, error) {
	args := m.Called(ctx, userID, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RepostResponse), args.Error(1)
}

type mockBookmarkSvc struct{ mock.Mock }

func (m *mockBookmarkSvc) Bookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockBookmarkSvc) Unbookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return m.Called(ctx, userID, postID).Error(0)
}

func (m *mockBookmarkSvc) GetBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

type mockUserSvc struct{ mock.Mock }

func (m *mockUserSvc) GetProfile(ctx context.Context, handle string, viewerID *uuid.UUID) (*dto.UserResponse, error) {
	args := m.Called(ctx, handle, viewerID)
	return userResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserSvc) UpdateProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	return userResponseOrNil(args.Get(0)), args.Error(1)
}

func (m *mockUserSvc) UpdateTheme(ctx context.Context, userID uuid.UUID, theme string) error {
	return m.Called(ctx, userID, theme).Error(0)
}

func (m *mockUserSvc) Follow(ctx context.Context, followerID uuid.UUID, handle string) error {
	return m.Called(ctx, followerID, handle).Error(0)
}

func (m *mockUserSvc) Unfollow(ctx context.Context, followerID uuid.UUID, handle string) error {
	return m.Called(ctx, followerID, handle).Error(0)
}

func (m *mockUserSvc) GetFollowers(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	args := m.Called(ctx, handle, viewerID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserListResponse), args.Error(1)
}

func (m *mockUserSvc) GetFollowing(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	args := m.Called(ctx, handle, viewerID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserListResponse), args.Error(1)
}

func (m *mockUserSvc) ChangeEmail(ctx context.Context, userID uuid.UUID, req *dto.ChangeEmailRequest) error {
	return m.Called(ctx, userID, req).Error(0)
}

func (m *mockUserSvc) ChangePassword(ctx context.Context, userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	return m.Called(ctx, userID, req).Error(0)
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

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) FindByHandle(ctx context.Context, handle string) (*model.User, error) {
	args := m.Called(ctx, handle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) FindAll(ctx context.Context, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	args := m.Called(ctx, cursor, limit)
	var next *time.Time
	if args.Get(1) != nil {
		next = args.Get(1).(*time.Time)
	}
	return args.Get(0).([]model.User), next, args.Error(2)
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

// ============================================================
// ヘルパー
// ============================================================

func authResultOrNil(v interface{}) *service.AuthResult {
	if v == nil {
		return nil
	}
	return v.(*service.AuthResult)
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

func userResponseOrNil(v interface{}) *dto.UserResponse {
	if v == nil {
		return nil
	}
	return v.(*dto.UserResponse)
}
