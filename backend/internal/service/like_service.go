package service

import (
	"context"
	"errors"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
)

var ErrAlreadyLiked = errors.New("already liked")

// LikeService はいいねのトグルを担う
type LikeService interface {
	Like(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error)
	Unlike(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error)
}

type likeService struct {
	postRepo        repository.PostRepository
	likeRepo        repository.LikeRepository
	notificationSvc NotificationService
}

func NewLikeService(postRepo repository.PostRepository, likeRepo repository.LikeRepository, notificationSvc NotificationService) LikeService {
	return &likeService{postRepo: postRepo, likeRepo: likeRepo, notificationSvc: notificationSvc}
}

func (s *likeService) Like(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	alreadyLiked, err := s.likeRepo.IsLiked(ctx, userID, postID)
	if err != nil {
		return nil, err
	}
	if alreadyLiked {
		return nil, ErrAlreadyLiked
	}

	if err := s.likeRepo.Create(ctx, userID, postID); err != nil {
		return nil, err
	}

	if err := s.notificationSvc.Notify(ctx, post.UserID, userID, "like", &postID); err != nil {
		return nil, err
	}

	return s.buildResponse(ctx, postID, true)
}

func (s *likeService) Unlike(ctx context.Context, userID, postID uuid.UUID) (*dto.LikeResponse, error) {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		return nil, err
	}

	if err := s.likeRepo.Delete(ctx, userID, postID); err != nil {
		return nil, err
	}

	return s.buildResponse(ctx, postID, false)
}

func (s *likeService) buildResponse(ctx context.Context, postID uuid.UUID, isLiked bool) (*dto.LikeResponse, error) {
	count, err := s.likeRepo.CountByPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	return &dto.LikeResponse{PostID: postID.String(), LikesCount: count, IsLiked: isLiked}, nil
}
