package service

import (
	"context"
	"mime/multipart"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
)

const commentMaxImages = 2

// CommentService はコメント（返信）投稿を担う
type CommentService interface {
	CreateComment(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error)
}

type commentService struct {
	postRepo        repository.PostRepository
	postSvc         PostService
	notificationSvc NotificationService
}

func NewCommentService(postRepo repository.PostRepository, postSvc PostService, notificationSvc NotificationService) CommentService {
	return &commentService{postRepo: postRepo, postSvc: postSvc, notificationSvc: notificationSvc}
}

func (s *commentService) CreateComment(ctx context.Context, userID, postID uuid.UUID, content string, files []*multipart.FileHeader) (*dto.PostResponse, error) {
	parent, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	resp, err := s.postSvc.CreateWithRefs(ctx, userID, content, files, &postID, nil, commentMaxImages, false)
	if err != nil {
		return nil, err
	}

	if err := s.notificationSvc.Notify(ctx, parent.UserID, userID, "comment", &postID); err != nil {
		return nil, err
	}

	return resp, nil
}
