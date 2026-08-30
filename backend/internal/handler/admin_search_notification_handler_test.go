package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================
// 管理者・検索・通知サービスのモック
// ============================================================

type mockAdminSvc struct{ mock.Mock }

func (m *mockAdminSvc) ForceDeletePost(ctx context.Context, postID uuid.UUID) error {
	return m.Called(ctx, postID).Error(0)
}

func (m *mockAdminSvc) SuspendUser(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockAdminSvc) UnsuspendUser(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockAdminSvc) ListUsers(ctx context.Context, cursor string, limit int) (*dto.AdminUserListResponse, error) {
	args := m.Called(ctx, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AdminUserListResponse), args.Error(1)
}

type mockSearchSvc struct{ mock.Mock }

func (m *mockSearchSvc) SearchUsers(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	args := m.Called(ctx, query, viewerID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UserListResponse), args.Error(1)
}

func (m *mockSearchSvc) SearchPosts(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, query, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

func (m *mockSearchSvc) GetPostsByHashtag(ctx context.Context, tag string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	args := m.Called(ctx, tag, viewerID, cursor, limit)
	return postListOrNil(args.Get(0)), args.Error(1)
}

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

// ============================================================
// 管理者ハンドラー
// ============================================================

func TestAdminHandler_AdminDeletePost_Success(t *testing.T) {
	postID := uuid.New()
	svc := new(mockAdminSvc)
	svc.On("ForceDeletePost", mock.Anything, postID).Return(nil)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/admin/posts/"+postID.String(), "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewAdminHandler(svc).AdminDeletePost(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAdminHandler_AdminDeletePost_NotFound(t *testing.T) {
	postID := uuid.New()
	svc := new(mockAdminSvc)
	svc.On("ForceDeletePost", mock.Anything, postID).Return(repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/admin/posts/"+postID.String(), "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewAdminHandler(svc).AdminDeletePost(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "投稿が見つかりません", decodeErrorResponse(t, rec).Message)
}

func TestAdminHandler_AdminDeletePost_InvalidID(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/admin/posts/xxx", "")
	withParam(c, []string{"id"}, []string{"xxx"})

	require.NoError(t, NewAdminHandler(new(mockAdminSvc)).AdminDeletePost(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminHandler_SuspendUser_Success(t *testing.T) {
	userID := uuid.New()
	svc := new(mockAdminSvc)
	svc.On("SuspendUser", mock.Anything, userID).Return(nil)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/admin/users/"+userID.String()+"/suspend", "")
	withParam(c, []string{"id"}, []string{userID.String()})

	require.NoError(t, NewAdminHandler(svc).SuspendUser(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAdminHandler_SuspendUser_NotFound(t *testing.T) {
	userID := uuid.New()
	svc := new(mockAdminSvc)
	svc.On("SuspendUser", mock.Anything, userID).Return(repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/admin/users/"+userID.String()+"/suspend", "")
	withParam(c, []string{"id"}, []string{userID.String()})

	require.NoError(t, NewAdminHandler(svc).SuspendUser(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "ユーザーが見つかりません", decodeErrorResponse(t, rec).Message)
}

func TestAdminHandler_UnsuspendUser_Success(t *testing.T) {
	userID := uuid.New()
	svc := new(mockAdminSvc)
	svc.On("UnsuspendUser", mock.Anything, userID).Return(nil)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/admin/users/"+userID.String()+"/suspend", "")
	withParam(c, []string{"id"}, []string{userID.String()})

	require.NoError(t, NewAdminHandler(svc).UnsuspendUser(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAdminHandler_UnsuspendUser_InvalidID(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/admin/users/xxx/suspend", "")
	withParam(c, []string{"id"}, []string{"xxx"})

	require.NoError(t, NewAdminHandler(new(mockAdminSvc)).UnsuspendUser(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "USER_NOT_FOUND", decodeErrorResponse(t, rec).Code)
}

func TestAdminHandler_ListUsers_Success(t *testing.T) {
	svc := new(mockAdminSvc)
	svc.On("ListUsers", mock.Anything, "", 20).
		Return(&dto.AdminUserListResponse{Data: []dto.AdminUserResponse{{Handle: "taro"}}}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/admin/users", "")

	require.NoError(t, NewAdminHandler(svc).ListUsers(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.AdminUserListResponse
	decodeJSON(t, rec, &body)
	require.Len(t, body.Data, 1)
	assert.Equal(t, "taro", body.Data[0].Handle)
}

// ============================================================
// 検索ハンドラー
// ============================================================

func TestSearchHandler_SearchUsers_QueryTooShort(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/users?q=a", "")

	require.NoError(t, NewSearchHandler(new(mockSearchSvc)).SearchUsers(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "検索クエリは2文字以上で指定してください", body.Message)
}

func TestSearchHandler_SearchUsers_Success(t *testing.T) {
	svc := new(mockSearchSvc)
	svc.On("SearchUsers", mock.Anything, "taro", (*uuid.UUID)(nil), "", 20).
		Return(&dto.UserListResponse{Data: []dto.UserResponse{{Handle: "taro"}}}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/users?q=taro", "")

	require.NoError(t, NewSearchHandler(svc).SearchUsers(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.UserListResponse
	decodeJSON(t, rec, &body)
	require.Len(t, body.Data, 1)
}

func TestSearchHandler_SearchPosts_EmptyQuery(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/posts", "")

	require.NoError(t, NewSearchHandler(new(mockSearchSvc)).SearchPosts(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "検索クエリを指定してください", decodeErrorResponse(t, rec).Message)
}

func TestSearchHandler_SearchPosts_Success(t *testing.T) {
	svc := new(mockSearchSvc)
	svc.On("SearchPosts", mock.Anything, "テスト", (*uuid.UUID)(nil), "", 20).
		Return(&dto.PostListResponse{Data: []dto.PostResponse{{Content: "テスト投稿"}}}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/posts?q=%E3%83%86%E3%82%B9%E3%83%88", "")

	require.NoError(t, NewSearchHandler(svc).SearchPosts(c))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSearchHandler_GetHashtagPosts_NotFound(t *testing.T) {
	svc := new(mockSearchSvc)
	svc.On("GetPostsByHashtag", mock.Anything, "unknown", (*uuid.UUID)(nil), "", 20).
		Return(nil, repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/hashtags/unknown", "")
	withParam(c, []string{"tag"}, []string{"unknown"})

	require.NoError(t, NewSearchHandler(svc).GetHashtagPosts(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "HASHTAG_NOT_FOUND", body.Code)
	assert.Equal(t, "ハッシュタグが見つかりません", body.Message)
}

func TestSearchHandler_GetHashtagPosts_Success(t *testing.T) {
	svc := new(mockSearchSvc)
	svc.On("GetPostsByHashtag", mock.Anything, "golang", (*uuid.UUID)(nil), "", 20).
		Return(&dto.PostListResponse{Data: []dto.PostResponse{{Content: "#golang"}}}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/search/hashtags/golang", "")
	withParam(c, []string{"tag"}, []string{"golang"})

	require.NoError(t, NewSearchHandler(svc).GetHashtagPosts(c))

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ============================================================
// 通知ハンドラー
// ============================================================

func TestNotificationHandler_GetNotifications_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/notifications", "")

	require.NoError(t, NewNotificationHandler(new(mockNotificationSvc)).GetNotifications(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "認証が必要です", body.Message)
}

func TestNotificationHandler_GetNotifications_Success(t *testing.T) {
	userID := uuid.New()
	svc := new(mockNotificationSvc)
	svc.On("GetNotifications", mock.Anything, userID, "", 20).
		Return(&dto.NotificationListResponse{
			Data:        []dto.NotificationResponse{{Type: "like"}},
			UnreadCount: 1,
		}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/notifications", "")
	withAuth(c, userID)

	require.NoError(t, NewNotificationHandler(svc).GetNotifications(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.NotificationListResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, int64(1), body.UnreadCount)
}

func TestNotificationHandler_MarkAllRead_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/notifications/read", "")

	require.NoError(t, NewNotificationHandler(new(mockNotificationSvc)).MarkAllRead(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestNotificationHandler_MarkAllRead_Success(t *testing.T) {
	userID := uuid.New()
	svc := new(mockNotificationSvc)
	svc.On("MarkAllRead", mock.Anything, userID).Return(nil)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/notifications/read", "")
	withAuth(c, userID)

	require.NoError(t, NewNotificationHandler(svc).MarkAllRead(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	svc.AssertExpectations(t)
}
