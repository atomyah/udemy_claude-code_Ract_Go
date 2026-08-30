package service

import (
	"context"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
)

// AdminService は管理者専用の投稿強制削除・ユーザー停止・ユーザー一覧機能を担う
type AdminService interface {
	ForceDeletePost(ctx context.Context, postID uuid.UUID) error
	SuspendUser(ctx context.Context, userID uuid.UUID) error
	UnsuspendUser(ctx context.Context, userID uuid.UUID) error
	ListUsers(ctx context.Context, cursor string, limit int) (*dto.AdminUserListResponse, error)
}

type adminService struct {
	postRepo repository.PostRepository
	userRepo repository.UserRepository
}

func NewAdminService(postRepo repository.PostRepository, userRepo repository.UserRepository) AdminService {
	return &adminService{postRepo: postRepo, userRepo: userRepo}
}

func (s *adminService) ForceDeletePost(ctx context.Context, postID uuid.UUID) error {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		return err
	}
	return s.postRepo.SoftDelete(ctx, postID)
}

func (s *adminService) SuspendUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := s.userRepo.FindByID(ctx, userID); err != nil {
		return err
	}
	return s.userRepo.Suspend(ctx, userID)
}

func (s *adminService) UnsuspendUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := s.userRepo.FindByID(ctx, userID); err != nil {
		return err
	}
	return s.userRepo.Unsuspend(ctx, userID)
}

func (s *adminService) ListUsers(ctx context.Context, cursor string, limit int) (*dto.AdminUserListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	users, next, err := s.userRepo.FindAll(ctx, cursorTime, limit)
	if err != nil {
		return nil, err
	}

	data := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		data = append(data, toAdminUserResponse(&u))
	}

	var nextCursor *string
	if next != nil {
		s := utils.FormatCursor(*next)
		nextCursor = &s
	}

	return &dto.AdminUserListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}

func toAdminUserResponse(u *model.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		Handle:      u.Handle,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		IsSuspended: u.IsSuspended,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
