package repository

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowRepository interface {
	Create(ctx context.Context, follow *model.Follow) error
	Delete(ctx context.Context, followerID, followingID uuid.UUID) error
	Exists(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
	CountFollowers(ctx context.Context, userID uuid.UUID) (int64, error)
	CountFollowing(ctx context.Context, userID uuid.UUID) (int64, error)
	GetFollowers(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error)
	GetFollowing(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error)
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Create(ctx context.Context, follow *model.Follow) error {
	return r.db.WithContext(ctx).Create(follow).Error
}

func (r *followRepository) Delete(ctx context.Context, followerID, followingID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&model.Follow{}).Error
}

func (r *followRepository) Exists(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

func (r *followRepository) CountFollowers(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("following_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *followRepository) CountFollowing(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("follower_id = ?", userID).Count(&count).Error
	return count, err
}

// GetFollowers はuserIDをフォローしているユーザー一覧をフォロー作成日時のcursorページネーションで返す
func (r *followRepository) GetFollowers(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	query := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("users.*, follows.created_at AS follow_created_at").
		Joins("JOIN follows ON follows.follower_id = users.id").
		Where("follows.following_id = ?", userID)
	return r.paginateByFollowCreatedAt(query, cursor, limit)
}

// GetFollowing はuserIDがフォローしているユーザー一覧をフォロー作成日時のcursorページネーションで返す
func (r *followRepository) GetFollowing(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	query := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("users.*, follows.created_at AS follow_created_at").
		Joins("JOIN follows ON follows.following_id = users.id").
		Where("follows.follower_id = ?", userID)
	return r.paginateByFollowCreatedAt(query, cursor, limit)
}

type userWithFollowCreatedAt struct {
	model.User
	FollowCreatedAt time.Time
}

func (r *followRepository) paginateByFollowCreatedAt(query *gorm.DB, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	if cursor != nil {
		query = query.Where("follows.created_at < ?", *cursor)
	}

	var rows []userWithFollowCreatedAt
	if err := query.Order("follows.created_at DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(rows) > limit {
		rows = rows[:limit]
		nextCursor = &rows[len(rows)-1].FollowCreatedAt
	}

	users := make([]model.User, len(rows))
	for i, row := range rows {
		users[i] = row.User
	}
	return users, nextCursor, nil
}
