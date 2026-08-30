package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error)
	Update(ctx context.Context, post *model.Post) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	GetExplore(ctx context.Context, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
	GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
	GetRepliesByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
	GetComments(ctx context.Context, postID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Post, error)
	CountComments(ctx context.Context, postID uuid.UUID) (int64, error)
	CountReposts(ctx context.Context, postID uuid.UUID) (int64, error)
	FindActiveRepost(ctx context.Context, userID, postID uuid.UUID) (*model.Post, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	var post model.Post
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Media").
		Preload("RepostOfPost").
		Preload("RepostOfPost.User").
		Preload("RepostOfPost.Media").
		Preload("ReplyToPost").
		Preload("ReplyToPost.User").
		Where("id = ? AND is_deleted = false", id).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &post, err
}

func (r *postRepository) Update(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Save(post).Error
}

func (r *postRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("is_deleted", true).Error
}

func (r *postRepository) GetExplore(ctx context.Context, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	query := r.basePostQuery(ctx)
	return r.paginate(query, cursor, limit)
}

// GetByUser は指定ユーザーの投稿（返信を除く）を取得する
func (r *postRepository) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	query := r.basePostQuery(ctx).Where("posts.user_id = ? AND posts.reply_to IS NULL", userID)
	return r.paginate(query, cursor, limit)
}

// GetRepliesByUser は指定ユーザーが他の投稿に対して行った返信のみを取得する
func (r *postRepository) GetRepliesByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	query := r.basePostQuery(ctx).Where("posts.user_id = ? AND posts.reply_to IS NOT NULL", userID)
	return r.paginate(query, cursor, limit)
}

func (r *postRepository) GetComments(ctx context.Context, postID uuid.UUID, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	query := r.basePostQuery(ctx).Where("posts.reply_to = ?", postID)
	return r.paginate(query, cursor, limit)
}

// FindByIDs は指定IDの投稿を取得する（順序は保証しない。呼び出し元で並べ替えること）
func (r *postRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Media").
		Preload("RepostOfPost").
		Preload("RepostOfPost.User").
		Preload("RepostOfPost.Media").
		Preload("ReplyToPost").
		Preload("ReplyToPost.User").
		Where("id IN ? AND is_deleted = false", ids).
		Find(&posts).Error
	return posts, err
}

func (r *postRepository) CountComments(ctx context.Context, postID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("reply_to = ? AND is_deleted = false", postID).
		Count(&count).Error
	return count, err
}

func (r *postRepository) CountReposts(ctx context.Context, postID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("repost_of = ? AND is_deleted = false", postID).
		Count(&count).Error
	return count, err
}

func (r *postRepository) FindActiveRepost(ctx context.Context, userID, postID uuid.UUID) (*model.Post, error) {
	var post model.Post
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND repost_of = ? AND is_deleted = false", userID, postID).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &post, err
}

// basePostQuery は削除済み投稿・停止ユーザーの投稿を除外した共通クエリを返す
func (r *postRepository) basePostQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&model.Post{}).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.is_deleted = false AND users.is_suspended = false").
		Preload("User").
		Preload("Media").
		Preload("RepostOfPost").
		Preload("RepostOfPost.User").
		Preload("RepostOfPost.Media").
		Preload("ReplyToPost").
		Preload("ReplyToPost.User")
}

func (r *postRepository) paginate(query *gorm.DB, cursor *time.Time, limit int) ([]model.Post, *time.Time, error) {
	if cursor != nil {
		query = query.Where("posts.created_at < ?", *cursor)
	}

	var posts []model.Post
	if err := query.Order("posts.created_at DESC").Limit(limit + 1).Find(&posts).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(posts) > limit {
		posts = posts[:limit]
		nextCursor = &posts[len(posts)-1].CreatedAt
	}
	return posts, nextCursor, nil
}
