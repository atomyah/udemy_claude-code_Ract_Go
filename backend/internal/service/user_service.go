package service

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrSelfFollow      = errors.New("cannot follow yourself")
	ErrAlreadyFollowed = errors.New("already following")
	ErrNotFollowing    = errors.New("not following")
)

type UserService interface {
	GetProfile(ctx context.Context, handle string, viewerID *uuid.UUID) (*dto.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*dto.UserResponse, error)
	UpdateTheme(ctx context.Context, userID uuid.UUID, theme string) error
	Follow(ctx context.Context, followerID uuid.UUID, handle string) error
	Unfollow(ctx context.Context, followerID uuid.UUID, handle string) error
	GetFollowers(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error)
	GetFollowing(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error)
	ChangeEmail(ctx context.Context, userID uuid.UUID, req *dto.ChangeEmailRequest) error
	ChangePassword(ctx context.Context, userID uuid.UUID, req *dto.ChangePasswordRequest) error
}

type userService struct {
	userRepo   repository.UserRepository
	followRepo repository.FollowRepository
}

func NewUserService(userRepo repository.UserRepository, followRepo repository.FollowRepository) UserService {
	return &userService{userRepo: userRepo, followRepo: followRepo}
}

func (s *userService) GetProfile(ctx context.Context, handle string, viewerID *uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	resp, err := s.toUserResponseWithCounts(ctx, user, viewerID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	user.Bio = req.Bio
	user.Location = req.Location
	user.WebsiteURL = req.WebsiteURL
	if req.Birthday != nil && *req.Birthday != "" {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return nil, err
		}
		user.Birthday = &birthday
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.toUserResponseWithCounts(ctx, user, &userID)
}

func (s *userService) UpdateTheme(ctx context.Context, userID uuid.UUID, theme string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Theme = theme
	return s.userRepo.Update(ctx, user)
}

func (s *userService) Follow(ctx context.Context, followerID uuid.UUID, handle string) error {
	target, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return err
	}
	if target.ID == followerID {
		return ErrSelfFollow
	}

	exists, err := s.followRepo.Exists(ctx, followerID, target.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyFollowed
	}

	return s.followRepo.Create(ctx, &model.Follow{FollowerID: followerID, FollowingID: target.ID})
}

func (s *userService) Unfollow(ctx context.Context, followerID uuid.UUID, handle string) error {
	target, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return err
	}

	exists, err := s.followRepo.Exists(ctx, followerID, target.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFollowing
	}

	return s.followRepo.Delete(ctx, followerID, target.ID)
}

func (s *userService) GetFollowers(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	target, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	users, next, err := s.followRepo.GetFollowers(ctx, target.ID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toUserListResponse(ctx, users, next, viewerID)
}

func (s *userService) GetFollowing(ctx context.Context, handle string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	target, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	users, next, err := s.followRepo.GetFollowing(ctx, target.ID, cursorTime, limit)
	if err != nil {
		return nil, err
	}
	return s.toUserListResponse(ctx, users, next, viewerID)
}

func (s *userService) ChangeEmail(ctx context.Context, userID uuid.UUID, req *dto.ChangeEmailRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == nil {
		return ErrNoPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidPassword
	}

	if req.NewEmail != user.Email {
		exists, err := s.userRepo.ExistsByEmail(ctx, req.NewEmail)
		if err != nil {
			return err
		}
		if exists {
			return ErrEmailTaken
		}
	}

	user.Email = req.NewEmail
	return s.userRepo.Update(ctx, user)
}

func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == nil {
		return ErrNoPassword
	}
	// 現在のパスワードが正しいか確認する(パスワード検証)
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}
	newHash := string(hash)
	user.PasswordHash = &newHash
	return s.userRepo.Update(ctx, user)
}
// 現在のパスワードを CompareHashAndPassword で検証してから、新パスワードを再ハッシュして更新

// ┌──────────────────────────────┬─────────────────────┐
// │             用途             │        場所         │
// ├──────────────────────────────┼─────────────────────┤
// │ ログイン                     │ auth_service.go:115 │
// ├──────────────────────────────┼─────────────────────┤
// │ メールアドレス変更の本人確認 │ user_service.go:170 │
// ├──────────────────────────────┼─────────────────────┤
// │ パスワード変更の本人確認     │ user_service.go:196 │
// └──────────────────────────────┴─────────────────────┘

func (s *userService) toUserListResponse(ctx context.Context, users []model.User, next *time.Time, viewerID *uuid.UUID) (*dto.UserListResponse, error) {
	data := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		resp, err := s.toUserResponseWithCounts(ctx, &users[i], viewerID)
		if err != nil {
			return nil, err
		}
		data = append(data, *resp)
	}

	var nextCursor *string
	if next != nil {
		s := utils.FormatCursor(*next)
		nextCursor = &s
	}

	return &dto.UserListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}

func (s *userService) toUserResponseWithCounts(ctx context.Context, user *model.User, viewerID *uuid.UUID) (*dto.UserResponse, error) {
	resp := ToUserResponse(user)

	followersCount, err := s.followRepo.CountFollowers(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	followingCount, err := s.followRepo.CountFollowing(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	resp.FollowersCount = followersCount
	resp.FollowingCount = followingCount

	if viewerID != nil {
		isFollowing, err := s.followRepo.Exists(ctx, *viewerID, user.ID)
		if err != nil {
			return nil, err
		}
		resp.IsFollowing = isFollowing
	}

	return &resp, nil
}
