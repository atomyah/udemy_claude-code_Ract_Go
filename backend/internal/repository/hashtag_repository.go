package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HashtagRepository interface {
	FindOrCreate(ctx context.Context, name string) (*model.Hashtag, error)
	AttachToPost(ctx context.Context, postID uuid.UUID, hashtagIDs []uuid.UUID) error
	FindByName(ctx context.Context, name string) (*model.Hashtag, error)
	GetPostIDsByHashtag(ctx context.Context, hashtagID uuid.UUID, cursor *time.Time, limit int) ([]uuid.UUID, error)
}

type hashtagRepository struct {
	db *gorm.DB
}

func NewHashtagRepository(db *gorm.DB) HashtagRepository {
	return &hashtagRepository{db: db}
}

func (r *hashtagRepository) FindOrCreate(ctx context.Context, name string) (*model.Hashtag, error) {
	hashtag := &model.Hashtag{Name: name}
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).
		Create(hashtag)
	if result.Error != nil {
		return nil, result.Error
	}
	// BeforeCreateフックでIDは常に採番されるため、実際に挿入されたか（RowsAffected）で判定する。
	// 挿入されなかった場合は既存のハッシュタグを引き直す
	if result.RowsAffected > 0 {
		return hashtag, nil
	}
	return r.FindByName(ctx, name)
}

func (r *hashtagRepository) FindByName(ctx context.Context, name string) (*model.Hashtag, error) {
	var hashtag model.Hashtag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&hashtag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &hashtag, err
}

func (r *hashtagRepository) AttachToPost(ctx context.Context, postID uuid.UUID, hashtagIDs []uuid.UUID) error {
	if len(hashtagIDs) == 0 {
		return nil
	}
	links := make([]model.PostHashtag, len(hashtagIDs))
	for i, id := range hashtagIDs {
		links[i] = model.PostHashtag{PostID: postID, HashtagID: id}
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&links).Error
}

func (r *hashtagRepository) GetPostIDsByHashtag(ctx context.Context, hashtagID uuid.UUID, cursor *time.Time, limit int) ([]uuid.UUID, error) {
	query := r.db.WithContext(ctx).Model(&model.Post{}).
		Joins("JOIN post_hashtags ON post_hashtags.post_id = posts.id").
		Where("post_hashtags.hashtag_id = ? AND posts.is_deleted = false", hashtagID)
	if cursor != nil {
		query = query.Where("posts.created_at < ?", *cursor)
	}

	var ids []uuid.UUID
	err := query.Order("posts.created_at DESC").Limit(limit).Pluck("posts.id", &ids).Error
	return ids, err
}
