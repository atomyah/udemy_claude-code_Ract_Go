package repository

import (
	"context"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaRepository interface {
	CreateBulk(ctx context.Context, media []model.Media) error
	FindByPostID(ctx context.Context, postID uuid.UUID) ([]model.Media, error)
	FindByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.Media, error)
}

type mediaRepository struct {
	db *gorm.DB
}

func NewMediaRepository(db *gorm.DB) MediaRepository {
	return &mediaRepository{db: db}
}

func (r *mediaRepository) CreateBulk(ctx context.Context, media []model.Media) error {
	if len(media) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&media).Error
}

func (r *mediaRepository) FindByPostID(ctx context.Context, postID uuid.UUID) ([]model.Media, error) {
	var media []model.Media
	err := r.db.WithContext(ctx).Where("post_id = ?", postID).Order("sort_order ASC").Find(&media).Error
	return media, err
}

func (r *mediaRepository) FindByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]model.Media, error) {
	result := make(map[uuid.UUID][]model.Media)
	if len(postIDs) == 0 {
		return result, nil
	}

	var media []model.Media
	if err := r.db.WithContext(ctx).Where("post_id IN ?", postIDs).Order("sort_order ASC").Find(&media).Error; err != nil {
		return nil, err
	}

	for _, m := range media {
		result[m.PostID] = append(result[m.PostID], m)
	}
	return result, nil
}
