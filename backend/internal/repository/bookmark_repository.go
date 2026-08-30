package repository

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookmarkRepository interface {
	Create(ctx context.Context, userID, postID uuid.UUID) error
	Delete(ctx context.Context, userID, postID uuid.UUID) error
	IsBookmarked(ctx context.Context, userID, postID uuid.UUID) (bool, error)
	GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
}

type bookmarkRepository struct {
	db *gorm.DB
}

func NewBookmarkRepository(db *gorm.DB) BookmarkRepository {
	return &bookmarkRepository{db: db}
}

func (r *bookmarkRepository) Create(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).Create(&model.Bookmark{UserID: userID, PostID: postID}).Error
}

func (r *bookmarkRepository) Delete(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.Bookmark{}).Error
}

func (r *bookmarkRepository) IsBookmarked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Bookmark{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}

// GetByUser はuserIDがブックマークした投稿一覧をブックマーク作成日時のcursorページネーションで返す
func (r *bookmarkRepository) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Select("posts.*, bookmarks.created_at AS bookmark_created_at").
		Joins("JOIN bookmarks ON bookmarks.post_id = posts.id").
		Joins("JOIN users ON users.id = posts.user_id").
		Where("bookmarks.user_id = ? AND posts.is_deleted = false AND users.is_suspended = false", userID).
		Preload("User").
		Preload("Media").
		Preload("RepostOfPost").
		Preload("RepostOfPost.User").
		Preload("RepostOfPost.Media").
		Preload("ReplyToPost").
		Preload("ReplyToPost.User")

	if cursor != nil {
		query = query.Where("bookmarks.created_at < ?", *cursor)
	}

	type postWithBookmarkCreatedAt struct {
		model.Post
		BookmarkCreatedAt time.Time
	}

	var rows []postWithBookmarkCreatedAt
	if err := query.Order("bookmarks.created_at DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(rows) > limit {
		rows = rows[:limit]
		nextCursor = &rows[len(rows)-1].BookmarkCreatedAt
	}

	posts := make([]model.Post, len(rows))
	for i, row := range rows {
		posts[i] = row.Post
	}
	return posts, nextCursor, nil
}
