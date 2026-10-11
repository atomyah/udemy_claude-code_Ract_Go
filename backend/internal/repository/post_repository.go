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
	// FindStatsByIDs は複数投稿の集計値と閲覧者の状態を1クエリでまとめて取得する（一覧表示のN+1回避用）
	FindStatsByIDs(ctx context.Context, postIDs []uuid.UUID, viewerID *uuid.UUID) (map[uuid.UUID]PostStats, error)
}

// PostStats は投稿1件分の集計値と閲覧者の状態
type PostStats struct {
	PostID        uuid.UUID
	LikesCount    int64
	CommentsCount int64
	RepostsCount  int64
	IsLiked       bool
	IsBookmarked  bool
	IsReposted    bool
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

// postStatsQuery は投稿ごとの集計値を相関サブクエリで1行にまとめる。
// 各サブクエリは likes(post_id) / posts(reply_to) / posts(repost_of) / likes・bookmarks の(user_id, post_id) の
// インデックスで解決されるため、投稿数に比例してクエリ本数が増えない。
// 閲覧者がいない場合は @viewer に uuid.Nil を渡し、is_* はすべて false になる。
const postStatsQuery = `
SELECT p.id AS post_id,
  (SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS likes_count,
  (SELECT COUNT(*) FROM posts c WHERE c.reply_to = p.id AND c.is_deleted = false) AS comments_count,
  (SELECT COUNT(*) FROM posts r WHERE r.repost_of = p.id AND r.is_deleted = false) AS reposts_count,
  EXISTS (SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = @viewer) AS is_liked,
  EXISTS (SELECT 1 FROM bookmarks b WHERE b.post_id = p.id AND b.user_id = @viewer) AS is_bookmarked,
  EXISTS (SELECT 1 FROM posts r WHERE r.repost_of = p.id AND r.user_id = @viewer AND r.is_deleted = false) AS is_reposted
FROM posts p
WHERE p.id IN @ids`

func (r *postRepository) FindStatsByIDs(ctx context.Context, postIDs []uuid.UUID, viewerID *uuid.UUID) (map[uuid.UUID]PostStats, error) {
	result := make(map[uuid.UUID]PostStats, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	viewer := uuid.Nil
	if viewerID != nil {
		viewer = *viewerID
	}

	var rows []PostStats
	err := r.db.WithContext(ctx).
		Raw(postStatsQuery, map[string]interface{}{"ids": postIDs, "viewer": viewer}).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PostID] = row
	}
	return result, nil
}

// basePostQuery は削除済み投稿・停止ユーザーの投稿を除外した共通クエリを返す。
// 投稿者は停止ユーザー除外のためにJOINしているので、そのJOIN（InnerJoins）で同時に読み込み、
// Preload("User") の追加クエリを発行しない。
func (r *postRepository) basePostQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&model.Post{}).
		InnerJoins("User").
		Where(`posts.is_deleted = false AND "User".is_suspended = false`).
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
