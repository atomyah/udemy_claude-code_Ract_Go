package repository

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"gorm.io/gorm"
)

type SearchRepository interface {
	SearchUsers(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.User, *time.Time, error)
	SearchPosts(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
}

type searchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) SearchRepository {
	return &searchRepository{db: db}
}

// SearchUsers はhandle・display_nameの部分一致でユーザーを検索する（停止ユーザーは除外）
func (r *searchRepository) SearchUsers(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	like := "%" + query + "%"
	q := r.db.WithContext(ctx).
		Where("is_suspended = false AND (handle ILIKE ? OR display_name ILIKE ?)", like, like)
	if cursor != nil {
		q = q.Where("created_at < ?", *cursor)
	}

	var users []model.User
	if err := q.Order("created_at DESC").Limit(limit + 1).Find(&users).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(users) > limit {
		users = users[:limit]
		nextCursor = &users[len(users)-1].CreatedAt
	}
	return users, nextCursor, nil
}

// SearchPosts は投稿本文の部分一致で検索する（削除済み・停止ユーザーの投稿は除外）
func (r *searchRepository) SearchPosts(ctx context.Context, query string, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	like := "%" + query + "%"
	q := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.is_deleted = false AND users.is_suspended = false AND posts.content ILIKE ?", like).
		Preload("User").
		Preload("Media").
		Preload("RepostOfPost").
		Preload("RepostOfPost.User").
		Preload("RepostOfPost.Media").
		Preload("ReplyToPost").
		Preload("ReplyToPost.User")
	if cursor != nil {
		q = q.Where("posts.created_at < ?", *cursor)
	}

	var posts []model.Post
	if err := q.Order("posts.created_at DESC").Limit(limit + 1).Find(&posts).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(posts) > limit {
		posts = posts[:limit]
		nextCursor = &posts[len(posts)-1].CreatedAt
	}
	return posts, nextCursor, nil
}
