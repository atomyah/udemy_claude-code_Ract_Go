package service

import (
	"context"
	"errors"
	"mime/multipart"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
)

var ErrAlreadyReposted = errors.New("already reposted")

const repostMaxImages = 2

// RepostService はリポストの作成・取消を担う
type RepostService interface {
	Repost(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.RepostResponse, error)
	Unrepost(ctx context.Context, userID, postID uuid.UUID) (*dto.RepostResponse, error)
}

type repostService struct {
	postRepo        repository.PostRepository
	postSvc         PostService
	notificationSvc NotificationService
}

func NewRepostService(postRepo repository.PostRepository, postSvc PostService, notificationSvc NotificationService) RepostService {
	return &repostService{postRepo: postRepo, postSvc: postSvc, notificationSvc: notificationSvc}
}

func (s *repostService) Repost(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.RepostResponse, error) {
	original, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if _, err := s.postRepo.FindActiveRepost(ctx, userID, postID); err == nil {
		return nil, ErrAlreadyReposted
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if _, err := s.postSvc.CreateWithRefs(ctx, userID, content, files, nil, &postID, repostMaxImages, false); err != nil {
		return nil, err
	}

	if err := s.notificationSvc.Notify(ctx, original.UserID, userID, "repost", &postID); err != nil {
		return nil, err
	}

	return s.buildResponse(ctx, postID, true)
}

func (s *repostService) Unrepost(ctx context.Context, userID, postID uuid.UUID) (*dto.RepostResponse, error) {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		return nil, err
	}

	repost, err := s.postRepo.FindActiveRepost(ctx, userID, postID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return s.buildResponse(ctx, postID, false)
		}
		return nil, err
	}

	if err := s.postRepo.SoftDelete(ctx, repost.ID); err != nil {
		return nil, err
	}

	return s.buildResponse(ctx, postID, false)
}

func (s *repostService) buildResponse(ctx context.Context, postID uuid.UUID, isReposted bool) (*dto.RepostResponse, error) {
	count, err := s.postRepo.CountReposts(ctx, postID)
	if err != nil {
		return nil, err
	}
	return &dto.RepostResponse{PostID: postID.String(), RepostsCount: count, IsReposted: isReposted}, nil
}
