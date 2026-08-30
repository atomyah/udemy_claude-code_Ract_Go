package service

import (
	"context"
	"errors"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
)

var ErrAlreadyBookmarked = errors.New("already bookmarked")

// BookmarkService はブックマークの追加・解除・一覧取得を担う
type BookmarkService interface {
	Bookmark(ctx context.Context, userID, postID uuid.UUID) error
	Unbookmark(ctx context.Context, userID, postID uuid.UUID) error
	GetBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
}

type bookmarkService struct {
	postRepo     repository.PostRepository
	bookmarkRepo repository.BookmarkRepository
	postSvc      PostService
}

func NewBookmarkService(postRepo repository.PostRepository, bookmarkRepo repository.BookmarkRepository, postSvc PostService) BookmarkService {
	return &bookmarkService{postRepo: postRepo, bookmarkRepo: bookmarkRepo, postSvc: postSvc}
}

func (s *bookmarkService) Bookmark(ctx context.Context, userID, postID uuid.UUID) error {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		return err
	}

	already, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadyBookmarked
	}

	return s.bookmarkRepo.Create(ctx, userID, postID)
}

func (s *bookmarkService) Unbookmark(ctx context.Context, userID, postID uuid.UUID) error {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		return err
	}
	return s.bookmarkRepo.Delete(ctx, userID, postID)
}

func (s *bookmarkService) GetBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.bookmarkRepo.GetByUser(ctx, userID, cursorTime, limit)
	if err != nil {
		return nil, err
	}

	data := make([]dto.PostResponse, 0, len(posts))
	for i := range posts {
		resp, err := s.postSvc.ToPostResponse(ctx, &posts[i], &userID)
		if err != nil {
			return nil, err
		}
		data = append(data, *resp)
	}

	var nextCursor *string
	if next != nil {
		c := utils.FormatCursor(*next)
		nextCursor = &c
	}

	return &dto.PostListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}
