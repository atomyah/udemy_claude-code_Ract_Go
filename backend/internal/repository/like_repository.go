package repository

import (
	"context"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LikeRepository interface {
	Create(ctx context.Context, userID, postID uuid.UUID) error
	Delete(ctx context.Context, userID, postID uuid.UUID) error
	CountByPost(ctx context.Context, postID uuid.UUID) (int64, error)
	IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error)
}

type likeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) Create(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).Create(&model.Like{UserID: userID, PostID: postID}).Error
}

func (r *likeRepository) Delete(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.Like{}).Error
}

func (r *likeRepository) CountByPost(ctx context.Context, postID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Like{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}

func (r *likeRepository) IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Like{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}
