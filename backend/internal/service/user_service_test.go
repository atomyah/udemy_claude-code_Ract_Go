package service

import (
	"context"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUserService_Follow_Success(t *testing.T) {
	followerID := uuid.New()
	target := &model.User{ID: uuid.New(), Handle: "hanako", DisplayName: "ハナコ"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "hanako").Return(target, nil)
	followRepo.On("Exists", mock.Anything, followerID, target.ID).Return(false, nil)
	followRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Follow")).Return(nil)

	svc := NewUserService(userRepo, followRepo)
	require.NoError(t, svc.Follow(context.Background(), followerID, "hanako"))

	created := followRepo.Calls[1].Arguments.Get(1).(*model.Follow)
	assert.Equal(t, followerID, created.FollowerID)
	assert.Equal(t, target.ID, created.FollowingID)
}

func TestUserService_Follow_SelfFollowIsRejected(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userRepo.On("FindByHandle", mock.Anything, "taro").Return(&model.User{ID: userID, Handle: "taro"}, nil)

	svc := NewUserService(userRepo, new(mockFollowRepo))
	err := svc.Follow(context.Background(), userID, "taro")

	assert.ErrorIs(t, err, ErrSelfFollow)
}

func TestUserService_Follow_AlreadyFollowing(t *testing.T) {
	followerID := uuid.New()
	target := &model.User{ID: uuid.New(), Handle: "hanako"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "hanako").Return(target, nil)
	followRepo.On("Exists", mock.Anything, followerID, target.ID).Return(true, nil)

	svc := NewUserService(userRepo, followRepo)
	err := svc.Follow(context.Background(), followerID, "hanako")

	assert.ErrorIs(t, err, ErrAlreadyFollowed)
	followRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestUserService_Follow_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("FindByHandle", mock.Anything, "unknown").Return(nil, repository.ErrNotFound)

	svc := NewUserService(userRepo, new(mockFollowRepo))
	err := svc.Follow(context.Background(), uuid.New(), "unknown")

	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestUserService_Unfollow_Success(t *testing.T) {
	followerID := uuid.New()
	target := &model.User{ID: uuid.New(), Handle: "hanako"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "hanako").Return(target, nil)
	followRepo.On("Exists", mock.Anything, followerID, target.ID).Return(true, nil)
	followRepo.On("Delete", mock.Anything, followerID, target.ID).Return(nil)

	svc := NewUserService(userRepo, followRepo)

	require.NoError(t, svc.Unfollow(context.Background(), followerID, "hanako"))
	followRepo.AssertExpectations(t)
}

func TestUserService_Unfollow_NotFollowing(t *testing.T) {
	followerID := uuid.New()
	target := &model.User{ID: uuid.New(), Handle: "hanako"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "hanako").Return(target, nil)
	followRepo.On("Exists", mock.Anything, followerID, target.ID).Return(false, nil)

	svc := NewUserService(userRepo, followRepo)
	err := svc.Unfollow(context.Background(), followerID, "hanako")

	assert.ErrorIs(t, err, ErrNotFollowing)
	followRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_GetProfile_IncludesCounts(t *testing.T) {
	viewerID := uuid.New()
	target := &model.User{ID: uuid.New(), Handle: "hanako", DisplayName: "ハナコ", Theme: "light"}

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByHandle", mock.Anything, "hanako").Return(target, nil)
	followRepo.On("CountFollowers", mock.Anything, target.ID).Return(3, nil)
	followRepo.On("CountFollowing", mock.Anything, target.ID).Return(5, nil)
	followRepo.On("Exists", mock.Anything, viewerID, target.ID).Return(true, nil)

	svc := NewUserService(userRepo, followRepo)
	resp, err := svc.GetProfile(context.Background(), "hanako", &viewerID)

	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.FollowersCount)
	assert.Equal(t, int64(5), resp.FollowingCount)
	assert.True(t, resp.IsFollowing)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	userID := uuid.New()
	user := &model.User{ID: userID, Handle: "taro", DisplayName: "タロウ", Theme: "light"}
	bio := "自己紹介です"

	userRepo := new(mockUserRepo)
	followRepo := new(mockFollowRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)
	followRepo.On("CountFollowers", mock.Anything, userID).Return(0, nil)
	followRepo.On("CountFollowing", mock.Anything, userID).Return(0, nil)
	followRepo.On("Exists", mock.Anything, userID, userID).Return(false, nil)

	svc := NewUserService(userRepo, followRepo)
	resp, err := svc.UpdateProfile(context.Background(), userID, &dto.UpdateProfileRequest{
		DisplayName: "タロウ改",
		Bio:         &bio,
	})

	require.NoError(t, err)
	assert.Equal(t, "タロウ改", resp.DisplayName)
	require.NotNil(t, resp.Bio)
	assert.Equal(t, bio, *resp.Bio)
}

func TestUserService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	userID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	h := string(hash)
	user := &model.User{ID: userID, PasswordHash: &h}

	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	svc := NewUserService(userRepo, new(mockFollowRepo))
	err = svc.ChangePassword(context.Background(), userID, &dto.ChangePasswordRequest{
		CurrentPassword: "wrong-password",
		NewPassword:     "newpassword123",
	})

	assert.ErrorIs(t, err, ErrInvalidPassword)
	userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUserService_ChangeEmail_EmailTaken(t *testing.T) {
	userID := uuid.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	h := string(hash)
	user := &model.User{ID: userID, Email: "old@example.com", PasswordHash: &h}

	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	userRepo.On("ExistsByEmail", mock.Anything, "taken@example.com").Return(true, nil)

	svc := NewUserService(userRepo, new(mockFollowRepo))
	err = svc.ChangeEmail(context.Background(), userID, &dto.ChangeEmailRequest{
		NewEmail:        "taken@example.com",
		CurrentPassword: "password123",
	})

	assert.ErrorIs(t, err, ErrEmailTaken)
	userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUserService_UpdateTheme_Success(t *testing.T) {
	userID := uuid.New()
	user := &model.User{ID: userID, Theme: "light"}

	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)

	svc := NewUserService(userRepo, new(mockFollowRepo))

	require.NoError(t, svc.UpdateTheme(context.Background(), userID, "dark"))
	assert.Equal(t, "dark", user.Theme)
}
