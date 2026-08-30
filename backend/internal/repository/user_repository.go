package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByHandle(ctx context.Context, handle string) (*model.User, error)
	FindAll(ctx context.Context, cursor *time.Time, limit int) ([]model.User, *time.Time, error)
	Update(ctx context.Context, user *model.User) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByHandle(ctx context.Context, handle string) (bool, error)
	Suspend(ctx context.Context, id uuid.UUID) error
	Unsuspend(ctx context.Context, id uuid.UUID) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepository) FindByHandle(ctx context.Context, handle string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("handle = ?", handle).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

// FindAll は全ユーザーを作成日時のcursorページネーションで返す（管理者用）
func (r *userRepository) FindAll(ctx context.Context, cursor *time.Time, limit int) ([]model.User, *time.Time, error) {
	query := r.db.WithContext(ctx).Model(&model.User{})
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

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) ExistsByHandle(ctx context.Context, handle string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("handle = ?", handle).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) Suspend(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("is_suspended", true).Error
}

func (r *userRepository) Unsuspend(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("is_suspended", false).Error
}
