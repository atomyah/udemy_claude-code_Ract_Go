package service

import (
	"context"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
)

// NotificationService は通知の作成・取得を担う
type NotificationService interface {
	// Notify は通知を作成する。actorIDとrecipientIDが同一の場合は何もしない（自分自身への通知は作らない）
	Notify(ctx context.Context, recipientID, actorID uuid.UUID, notifType string, postID *uuid.UUID) error
	GetNotifications(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.NotificationListResponse, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type notificationService struct {
	notificationRepo repository.NotificationRepository
}

func NewNotificationService(notificationRepo repository.NotificationRepository) NotificationService {
	return &notificationService{notificationRepo: notificationRepo}
}

func (s *notificationService) Notify(ctx context.Context, recipientID, actorID uuid.UUID, notifType string, postID *uuid.UUID) error {
	if recipientID == actorID {
		return nil
	}
	return s.notificationRepo.Create(ctx, &model.Notification{
		UserID:  recipientID,
		ActorID: actorID,
		Type:    notifType,
		PostID:  postID,
	})
}

func (s *notificationService) GetNotifications(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*dto.NotificationListResponse, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, err
	}

	notifications, next, err := s.notificationRepo.GetByUser(ctx, userID, cursorTime, limit)
	if err != nil {
		return nil, err
	}

	unreadCount, err := s.notificationRepo.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}

	data := make([]dto.NotificationResponse, len(notifications))
	for i, n := range notifications {
		var post *dto.PostSummary
		if n.Post != nil {
			post = toPostSummary(n.Post)
		}
		data[i] = dto.NotificationResponse{
			ID:        n.ID.String(),
			Type:      n.Type,
			Actor:     toUserInPost(n.Actor),
			Post:      post,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		}
	}

	var nextCursor *string
	if next != nil {
		c := utils.FormatCursor(*next)
		nextCursor = &c
	}

	return &dto.NotificationListResponse{
		Data:        data,
		NextCursor:  nextCursor,
		HasMore:     next != nil,
		UnreadCount: unreadCount,
	}, nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.notificationRepo.MarkAllRead(ctx, userID)
}
