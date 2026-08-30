package service

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
)

// SearchService はユーザー・投稿・ハッシュタグの検索を担う
type SearchService interface {
	SearchUsers(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error)
	SearchPosts(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
	GetPostsByHashtag(ctx context.Context, tag string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error)
}

type searchService struct {
	searchRepo  repository.SearchRepository
	hashtagRepo repository.HashtagRepository
	postRepo    repository.PostRepository
	followRepo  repository.FollowRepository
	postSvc     PostService
}

func NewSearchService(
	searchRepo repository.SearchRepository,
	hashtagRepo repository.HashtagRepository,
	postRepo repository.PostRepository,
	followRepo repository.FollowRepository,
	postSvc PostService,
) SearchService {
	return &searchService{searchRepo: searchRepo, hashtagRepo: hashtagRepo, postRepo: postRepo, followRepo: followRepo, postSvc: postSvc}
}

func (s *searchService) SearchUsers(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.UserListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	users, next, err := s.searchRepo.SearchUsers(ctx, query, cursorTime, limit)
	if err != nil {
		return nil, err
	}

	data := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp := ToUserResponse(&u)

		followersCount, err := s.followRepo.CountFollowers(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		followingCount, err := s.followRepo.CountFollowing(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		resp.FollowersCount = followersCount
		resp.FollowingCount = followingCount

		if viewerID != nil {
			isFollowing, err := s.followRepo.Exists(ctx, *viewerID, u.ID)
			if err != nil {
				return nil, err
			}
			resp.IsFollowing = isFollowing
		}
		data[i] = resp
	}

	var nextCursor *string
	if next != nil {
		c := utils.FormatCursor(*next)
		nextCursor = &c
	}

	return &dto.UserListResponse{Data: data, NextCursor: nextCursor, HasMore: next != nil}, nil
}

func (s *searchService) SearchPosts(ctx context.Context, query string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	posts, next, err := s.searchRepo.SearchPosts(ctx, query, cursorTime, limit)
	if err != nil {
		return nil, err
	}

	return s.toPostListResponse(ctx, posts, next, viewerID)
}

func (s *searchService) GetPostsByHashtag(ctx context.Context, tag string, viewerID *uuid.UUID, cursor string, limit int) (*dto.PostListResponse, error) {
	hashtag, err := s.hashtagRepo.FindByName(ctx, tag)
	if err != nil {
		return nil, err
	}

	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	ids, err := s.hashtagRepo.GetPostIDsByHashtag(ctx, hashtag.ID, cursorTime, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}

	posts, err := s.postRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	orderedPosts := reorderPosts(posts, ids)

	var nextCursor *time.Time
	if hasMore && len(orderedPosts) > 0 {
		t := orderedPosts[len(orderedPosts)-1].CreatedAt
		nextCursor = &t
	}

	return s.toPostListResponse(ctx, orderedPosts, nextCursor, viewerID)
}

func (s *searchService) toPostListResponse(ctx context.Context, posts []model.Post, next *time.Time, viewerID *uuid.UUID) (*dto.PostListResponse, error) {
	data := make([]dto.PostResponse, 0, len(posts))
	for i := range posts {
		resp, err := s.postSvc.ToPostResponse(ctx, &posts[i], viewerID)
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

// reorderPosts はFindByIDsで取得した投稿をids順に並べ替える
func reorderPosts(posts []model.Post, ids []uuid.UUID) []model.Post {
	byID := make(map[uuid.UUID]model.Post, len(posts))
	for _, p := range posts {
		byID[p.ID] = p
	}
	ordered := make([]model.Post, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered
}
