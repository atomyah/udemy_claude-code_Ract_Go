package service

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCommentService_CreateComment_Success(t *testing.T) {
	userID := uuid.New()
	authorID := uuid.New()
	postID := uuid.New()
	commentID := uuid.New()

	postRepo := new(mockPostRepo)
	postSvc := new(mockPostSvc)
	notifySvc := new(mockNotificationSvc)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: authorID}, nil)
	postSvc.On("CreateWithRefs", mock.Anything, userID, "返信します", []*multipart.FileHeader(nil), &postID, (*uuid.UUID)(nil), commentMaxImages, false).
		Return(&dto.PostResponse{ID: commentID.String(), Content: "返信します"}, nil)
	notifySvc.On("Notify", mock.Anything, authorID, userID, "comment", &postID).Return(nil)

	svc := NewCommentService(postRepo, postSvc, notifySvc)
	resp, err := svc.CreateComment(context.Background(), userID, postID, "返信します", nil)

	require.NoError(t, err)
	assert.Equal(t, commentID.String(), resp.ID)
	// コメントは動画不可・画像2枚までで作成される
	postSvc.AssertExpectations(t)
	notifySvc.AssertExpectations(t)
}

func TestCommentService_CreateComment_ParentNotFound(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postSvc := new(mockPostSvc)
	postRepo.On("FindByID", mock.Anything, postID).Return(nil, repository.ErrNotFound)

	svc := NewCommentService(postRepo, postSvc, new(mockNotificationSvc))
	_, err := svc.CreateComment(context.Background(), uuid.New(), postID, "返信します", nil)

	assert.ErrorIs(t, err, repository.ErrNotFound)
	postSvc.AssertNotCalled(t, "CreateWithRefs",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestBookmarkService_Bookmark_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	bookmarkRepo.On("IsBookmarked", mock.Anything, userID, postID).Return(false, nil)
	bookmarkRepo.On("Create", mock.Anything, userID, postID).Return(nil)

	svc := NewBookmarkService(postRepo, bookmarkRepo, new(mockPostSvc))

	require.NoError(t, svc.Bookmark(context.Background(), userID, postID))
	bookmarkRepo.AssertExpectations(t)
}

func TestBookmarkService_Bookmark_AlreadyBookmarked(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	bookmarkRepo.On("IsBookmarked", mock.Anything, userID, postID).Return(true, nil)

	svc := NewBookmarkService(postRepo, bookmarkRepo, new(mockPostSvc))
	err := svc.Bookmark(context.Background(), userID, postID)

	assert.ErrorIs(t, err, ErrAlreadyBookmarked)
	bookmarkRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestRepostService_Repost_AlreadyReposted(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	postSvc := new(mockPostSvc)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	postRepo.On("FindActiveRepost", mock.Anything, userID, postID).Return(&model.Post{ID: uuid.New()}, nil)

	svc := NewRepostService(postRepo, postSvc, new(mockNotificationSvc))
	_, err := svc.Repost(context.Background(), userID, postID, "", nil)

	assert.ErrorIs(t, err, ErrAlreadyReposted)
}

func TestRepostService_Unrepost_SoftDeletesRepost(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	repostID := uuid.New()

	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)
	postRepo.On("FindActiveRepost", mock.Anything, userID, postID).Return(&model.Post{ID: repostID}, nil)
	postRepo.On("SoftDelete", mock.Anything, repostID).Return(nil)
	postRepo.On("CountReposts", mock.Anything, postID).Return(0, nil)

	svc := NewRepostService(postRepo, new(mockPostSvc), new(mockNotificationSvc))
	resp, err := svc.Unrepost(context.Background(), userID, postID)

	require.NoError(t, err)
	assert.False(t, resp.IsReposted)
	assert.Equal(t, int64(0), resp.RepostsCount)
}

func TestAdminService_ForceDeletePost_SoftDeletes(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID}, nil)
	postRepo.On("SoftDelete", mock.Anything, postID).Return(nil)

	svc := NewAdminService(postRepo, new(mockUserRepo))

	require.NoError(t, svc.ForceDeletePost(context.Background(), postID))
	postRepo.AssertExpectations(t)
}

func TestAdminService_SuspendUser_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(&model.User{ID: userID}, nil)
	userRepo.On("Suspend", mock.Anything, userID).Return(nil)

	svc := NewAdminService(new(mockPostRepo), userRepo)

	require.NoError(t, svc.SuspendUser(context.Background(), userID))
	userRepo.AssertExpectations(t)
}

func TestAdminService_SuspendUser_NotFound(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(nil, repository.ErrNotFound)

	svc := NewAdminService(new(mockPostRepo), userRepo)
	err := svc.SuspendUser(context.Background(), userID)

	assert.ErrorIs(t, err, repository.ErrNotFound)
	userRepo.AssertNotCalled(t, "Suspend", mock.Anything, mock.Anything)
}

func TestAdminService_UnsuspendUser_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(&model.User{ID: userID, IsSuspended: true}, nil)
	userRepo.On("Unsuspend", mock.Anything, userID).Return(nil)

	svc := NewAdminService(new(mockPostRepo), userRepo)

	require.NoError(t, svc.UnsuspendUser(context.Background(), userID))
	userRepo.AssertExpectations(t)
}
