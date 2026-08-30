package repository

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *model.Notification) error
	GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Notification, *time.Time, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *model.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepository) GetByUser(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]model.Notification, *time.Time, error) {
	query := r.db.WithContext(ctx).
		Preload("Actor").
		Preload("Post").
		Preload("Post.User").
		Preload("Post.Media").
		Where("user_id = ?", userID)
	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	var notifications []model.Notification
	if err := query.Order("created_at DESC").Limit(limit + 1).Find(&notifications).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(notifications) > limit {
		notifications = notifications[:limit]
		nextCursor = &notifications[len(notifications)-1].CreatedAt
	}
	return notifications, nextCursor, nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}
