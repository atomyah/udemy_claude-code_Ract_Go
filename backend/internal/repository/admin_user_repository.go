package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrStatusConflict は審査ステータスの変更元が想定と異なる（他の管理者が先に操作したなど）ことを表す
var ErrStatusConflict = errors.New("user status changed by another operation")

// AdminUserRepository は管理画面のユーザー審査（status）に関する DB 操作
type AdminUserRepository interface {
	// List はユーザーを登録日時の新しい順に返す。status が空なら全ステータス
	List(ctx context.Context, status string, cursor *time.Time, limit int) ([]model.User, *time.Time, error)
	// UpdateStatus は現在のステータスが from のいずれかのときだけ to に変更する（条件付き UPDATE で同時操作を防ぐ）。
	// 対象が from 以外のステータスなら ErrStatusConflict を返す
	UpdateStatus(ctx context.Context, id uuid.UUID, from []string, to, reviewedBy string, at time.Time) error
	// CountPosts はユーザーの投稿数（削除済みを除く）を返す
	CountPosts(ctx context.Context, userID uuid.UUID) (int64, error)
}

type adminUserRepository struct {
	db *gorm.DB
}

func NewAdminUserRepository(db *gorm.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

func (r *adminUserRepository) List(ctx context.Context, status string, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	query := r.db.WithContext(ctx).Model(&model.User{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	var users []model.User
	if err := query.Order("created_at DESC").Limit(limit + 1).Find(&users).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(users) > limit {
		users = users[:limit]
		nextCursor = &users[len(users)-1].CreatedAt
	}
	return users, nextCursor, nil
}

func (r *adminUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, from []string, to, reviewedBy string, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND status IN ?", id, from).
		Updates(map[string]interface{}{
			"status":      to,
			"reviewed_at": at,
			"reviewed_by": reviewedBy,
			"updated_at":  at,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrStatusConflict
	}
	return nil
}

func (r *adminUserRepository) CountPosts(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("user_id = ? AND is_deleted = false", userID).Count(&count).Error
	return count, err
}
