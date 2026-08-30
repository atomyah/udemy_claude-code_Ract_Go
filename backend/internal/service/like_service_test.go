package service

import (
	"context"
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLikeService_Like_Success(t *testing.T) {
	userID := uuid.New()
	authorID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	notifySvc := new(mockNotificationSvc)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: authorID}, nil)
	likeRepo.On("IsLiked", mock.Anything, userID, postID).Return(false, nil)
	likeRepo.On("Create", mock.Anything, userID, postID).Return(nil)
	likeRepo.On("CountByPost", mock.Anything, postID).Return(1, nil)
	notifySvc.On("Notify", mock.Anything, authorID, userID, "like", &postID).Return(nil)

	svc := NewLikeService(postRepo, likeRepo, notifySvc)
	resp, err := svc.Like(context.Background(), userID, postID)

	require.NoError(t, err)
	assert.True(t, resp.IsLiked)
	assert.Equal(t, int64(1), resp.LikesCount)
	// 投稿者へ通知が作成される
	notifySvc.AssertExpectations(t)
}

func TestLikeService_Like_AlreadyLiked(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	likeRepo.On("IsLiked", mock.Anything, userID, postID).Return(true, nil)

	svc := NewLikeService(postRepo, likeRepo, new(mockNotificationSvc))
	_, err := svc.Like(context.Background(), userID, postID)

	assert.ErrorIs(t, err, ErrAlreadyLiked)
	likeRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestLikeService_Like_PostNotFound(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(nil, repository.ErrNotFound)

	svc := NewLikeService(postRepo, new(mockLikeRepo), new(mockNotificationSvc))
	_, err := svc.Like(context.Background(), uuid.New(), postID)

	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestLikeService_Unlike_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	likeRepo.On("Delete", mock.Anything, userID, postID).Return(nil)
	likeRepo.On("CountByPost", mock.Anything, postID).Return(0, nil)

	svc := NewLikeService(postRepo, likeRepo, new(mockNotificationSvc))
	resp, err := svc.Unlike(context.Background(), userID, postID)

	require.NoError(t, err)
	assert.False(t, resp.IsLiked)
	assert.Equal(t, int64(0), resp.LikesCount)
}

func TestNotificationService_Notify_SkipsSelfNotification(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	notificationRepo := new(mockNotificationRepo)
	svc := NewNotificationService(notificationRepo)

	// 自分の投稿への操作では通知を作らない
	err := svc.Notify(context.Background(), userID, userID, "like", &postID)

	require.NoError(t, err)
	notificationRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestNotificationService_Notify_CreatesNotification(t *testing.T) {
	recipientID := uuid.New()
	actorID := uuid.New()
	postID := uuid.New()

	notificationRepo := new(mockNotificationRepo)
	notificationRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Notification")).Return(nil)

	svc := NewNotificationService(notificationRepo)
	require.NoError(t, svc.Notify(context.Background(), recipientID, actorID, "comment", &postID))

	created := notificationRepo.Calls[0].Arguments.Get(1).(*model.Notification)
	assert.Equal(t, recipientID, created.UserID)
	assert.Equal(t, actorID, created.ActorID)
	assert.Equal(t, "comment", created.Type)
	require.NotNil(t, created.PostID)
	assert.Equal(t, postID, *created.PostID)
}

func TestNotificationService_MarkAllRead(t *testing.T) {
	userID := uuid.New()
	notificationRepo := new(mockNotificationRepo)
	notificationRepo.On("MarkAllRead", mock.Anything, userID).Return(nil)

	svc := NewNotificationService(notificationRepo)

	require.NoError(t, svc.MarkAllRead(context.Background(), userID))
	notificationRepo.AssertExpectations(t)
}
