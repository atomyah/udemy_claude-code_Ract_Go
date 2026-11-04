package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminRepository は管理画面の管理者ユーザー（admins テーブル）を扱う
type AdminRepository interface {
	Create(ctx context.Context, admin *model.Admin) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Admin, error)
	// FindByUsername はユーザー名（大文字・小文字を区別しない）で管理者を探す
	FindByUsername(ctx context.Context, username string) (*model.Admin, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	// List は管理者を作成日時の新しい順に返す（cursor ページネーション）
	List(ctx context.Context, cursor *time.Time, limit int) ([]model.Admin, *time.Time, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Create(ctx context.Context, admin *model.Admin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *adminRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &admin, err
}

func (r *adminRepository) FindByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.WithContext(ctx).Where("LOWER(username) = ?", strings.ToLower(username)).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &admin, err
}

func (r *adminRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Admin{}).Where("LOWER(username) = ?", strings.ToLower(username)).Count(&count).Error
	return count > 0, err
}

func (r *adminRepository) List(ctx context.Context, cursor *time.Time, limit int) ([]model.Admin, *time.Time, error) {
	query := r.db.WithContext(ctx).Model(&model.Admin{})
	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	var admins []model.Admin
	if err := query.Order("created_at DESC").Limit(limit + 1).Find(&admins).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(admins) > limit {
		admins = admins[:limit]
		nextCursor = &admins[len(admins)-1].CreatedAt
	}
	return admins, nextCursor, nil
}

func (r *adminRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Admin{}).Where("id = ?", id).Update("last_login_at", at).Error
}
