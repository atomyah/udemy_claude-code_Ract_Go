package service

import (
	"context"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSearchRepo struct{ mock.Mock }

func (m *mockSearchRepo) SearchUsers(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	args := m.Called(ctx, query, cursor, limit)
	return args.Get(0).([]model.User), timeOrNil(args.Get(1)), args.Error(2)
}

func (m *mockSearchRepo) SearchPosts(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	args := m.Called(ctx, query, cursor, limit)
	return args.Get(0).([]model.Post), timeOrNil(args.Get(1)), args.Error(2)
}

func TestSearchService_SearchUsers_IncludesFollowState(t *testing.T) {
	viewerID := uuid.New()
	target := model.User{ID: uuid.New(), Handle: "taro", DisplayName: "タロウ", Theme: "light"}

	searchRepo := new(mockSearchRepo)
	followRepo := new(mockFollowRepo)
	searchRepo.On("SearchUsers", mock.Anything, "taro", (*time.Time)(nil), 20).Return([]model.User{target}, nil, nil)
	followRepo.On("CountFollowers", mock.Anything, target.ID).Return(2, nil)
	followRepo.On("CountFollowing", mock.Anything, target.ID).Return(1, nil)
	followRepo.On("Exists", mock.Anything, viewerID, target.ID).Return(true, nil)

	svc := NewSearchService(searchRepo, new(mockHashtagRepo), new(mockPostRepo), followRepo, new(mockPostSvc))
	resp, err := svc.SearchUsers(context.Background(), "taro", &viewerID, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "taro", resp.Data[0].Handle)
	assert.Equal(t, int64(2), resp.Data[0].FollowersCount)
	assert.True(t, resp.Data[0].IsFollowing)
	assert.False(t, resp.HasMore)
}

func TestSearchService_SearchPosts_Success(t *testing.T) {
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: uuid.New(), Content: "テスト投稿"}

	searchRepo := new(mockSearchRepo)
	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	searchRepo.On("SearchPosts", mock.Anything, "テスト", (*time.Time)(nil), 20).Return([]model.Post{post}, nil, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	postSvc := newPostService(postRepo, likeRepo, bookmarkRepo)
	svc := NewSearchService(searchRepo, new(mockHashtagRepo), postRepo, new(mockFollowRepo), postSvc)
	resp, err := svc.SearchPosts(context.Background(), "テスト", nil, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "テスト投稿", resp.Data[0].Content)
}

func TestSearchService_GetPostsByHashtag_NotFound(t *testing.T) {
	hashtagRepo := new(mockHashtagRepo)
	hashtagRepo.On("FindByName", mock.Anything, "unknown").Return(nil, repository.ErrNotFound)

	svc := NewSearchService(new(mockSearchRepo), hashtagRepo, new(mockPostRepo), new(mockFollowRepo), new(mockPostSvc))
	_, err := svc.GetPostsByHashtag(context.Background(), "unknown", nil, "", 20)

	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestSearchService_GetPostsByHashtag_Success(t *testing.T) {
	hashtag := &model.Hashtag{ID: uuid.New(), Name: "golang"}
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: uuid.New(), Content: "#golang たのしい"}

	hashtagRepo := new(mockHashtagRepo)
	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	hashtagRepo.On("FindByName", mock.Anything, "golang").Return(hashtag, nil)
	hashtagRepo.On("GetPostIDsByHashtag", mock.Anything, hashtag.ID, (*time.Time)(nil), 21).Return([]uuid.UUID{postID}, nil)
	postRepo.On("FindByIDs", mock.Anything, []uuid.UUID{postID}).Return([]model.Post{post}, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	postSvc := newPostService(postRepo, likeRepo, bookmarkRepo)
	svc := NewSearchService(new(mockSearchRepo), hashtagRepo, postRepo, new(mockFollowRepo), postSvc)
	resp, err := svc.GetPostsByHashtag(context.Background(), "golang", nil, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "#golang たのしい", resp.Data[0].Content)
}

func TestNotificationService_GetNotifications_Success(t *testing.T) {
	userID := uuid.New()
	actorID := uuid.New()
	notification := model.Notification{
		ID:      uuid.New(),
		UserID:  userID,
		ActorID: actorID,
		Type:    "follow",
		Actor:   model.User{ID: actorID, Handle: "hanako", DisplayName: "ハナコ"},
	}

	notificationRepo := new(mockNotificationRepo)
	notificationRepo.On("GetByUser", mock.Anything, userID, (*time.Time)(nil), 20).
		Return([]model.Notification{notification}, nil, nil)
	notificationRepo.On("CountUnread", mock.Anything, userID).Return(1, nil)

	svc := NewNotificationService(notificationRepo)
	resp, err := svc.GetNotifications(context.Background(), userID, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "follow", resp.Data[0].Type)
	assert.Equal(t, "hanako", resp.Data[0].Actor.Handle)
	assert.Equal(t, int64(1), resp.UnreadCount)
	assert.False(t, resp.HasMore)
}

func TestBookmarkService_GetBookmarks_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: uuid.New(), Content: "ブックマーク済み"}

	bookmarkRepo := new(mockBookmarkRepo)
	postSvc := new(mockPostSvc)
	bookmarkRepo.On("GetByUser", mock.Anything, userID, (*time.Time)(nil), 20).Return([]model.Post{post}, nil, nil)
	postSvc.On("ToPostResponse", mock.Anything, mock.AnythingOfType("*model.Post"), &userID).
		Return(&dto.PostResponse{ID: postID.String(), Content: "ブックマーク済み"}, nil)

	svc := NewBookmarkService(new(mockPostRepo), bookmarkRepo, postSvc)
	resp, err := svc.GetBookmarks(context.Background(), userID, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "ブックマーク済み", resp.Data[0].Content)
}

func TestAdminService_ListUsers_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("FindAll", mock.Anything, (*time.Time)(nil), 20).
		Return([]model.User{{ID: uuid.New(), Email: "taro@example.com", Handle: "taro", Role: "user"}}, nil, nil)

	svc := NewAdminService(new(mockPostRepo), userRepo)
	resp, err := svc.ListUsers(context.Background(), "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "taro@example.com", resp.Data[0].Email)
	assert.False(t, resp.HasMore)
}

func TestUserService_GetFollowersAndFollowing(t *testing.T) {
	target := &model.User{ID: uuid.New(), Handle: "taro"}
	follower := model.User{ID: uuid.New(), Handle: "hanako", Theme: "light"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "taro").Return(target, nil)
	followRepo.On("GetFollowers", mock.Anything, target.ID, (*time.Time)(nil), 20).Return([]model.User{follower}, nil, nil)
	followRepo.On("GetFollowing", mock.Anything, target.ID, (*time.Time)(nil), 20).Return([]model.User{}, nil, nil)
	followRepo.On("CountFollowers", mock.Anything, follower.ID).Return(0, nil)
	followRepo.On("CountFollowing", mock.Anything, follower.ID).Return(0, nil)

	svc := NewUserService(userRepo, followRepo)

	followers, err := svc.GetFollowers(context.Background(), "taro", nil, "", 20)
	require.NoError(t, err)
	require.Len(t, followers.Data, 1)
	assert.Equal(t, "hanako", followers.Data[0].Handle)

	following, err := svc.GetFollowing(context.Background(), "taro", nil, "", 20)
	require.NoError(t, err)
	assert.Empty(t, following.Data)
}

func TestPostService_GetUserPostsAndComments(t *testing.T) {
	handle := "taro"
	user := &model.User{ID: uuid.New(), Handle: handle}
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: user.ID, Content: "ユーザーの投稿"}

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	userRepo := new(mockUserRepo)
	userRepo.On("FindByHandle", mock.Anything, handle).Return(user, nil)
	postRepo.On("GetByUser", mock.Anything, user.ID, (*time.Time)(nil), 20).Return([]model.Post{post}, nil, nil)
	postRepo.On("GetRepliesByUser", mock.Anything, user.ID, (*time.Time)(nil), 20).Return([]model.Post{}, nil, nil)
	postRepo.On("GetComments", mock.Anything, postID, (*time.Time)(nil), 20).Return([]model.Post{}, nil, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	svc := NewPostService(postRepo, new(mockMediaRepo), new(mockHashtagRepo), userRepo, likeRepo, bookmarkRepo, new(mockStorageSvc))

	posts, err := svc.GetUserPosts(context.Background(), handle, nil, "", 20)
	require.NoError(t, err)
	require.Len(t, posts.Data, 1)

	replies, err := svc.GetUserReplies(context.Background(), handle, nil, "", 20)
	require.NoError(t, err)
	assert.Empty(t, replies.Data)

	comments, err := svc.GetComments(context.Background(), postID, nil, "", 20)
	require.NoError(t, err)
	assert.Empty(t, comments.Data)
}

func TestPostService_GetHomeTimeline_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: uuid.New(), Content: "ホームの投稿"}

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	postRepo.On("GetExplore", mock.Anything, (*time.Time)(nil), 20).Return([]model.Post{post}, nil, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	svc := newPostService(postRepo, likeRepo, bookmarkRepo)
	resp, err := svc.GetHomeTimeline(context.Background(), userID, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "ホームの投稿", resp.Data[0].Content)
}
