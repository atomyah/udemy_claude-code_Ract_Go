package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrResetTokenNotActive はリセットトークンが存在しない・使用済み・失効済み・期限切れであることを表す
var ErrResetTokenNotActive = errors.New("password reset token is not active")

// PasswordResetRepository はパスワードリセット申請とリセットトークンを扱う
type PasswordResetRepository interface {
	// CreateRequestIfNone は対応中（pending / issued）の申請がなければ新しい申請を作る。
	// 既にあれば作らずに false を返す（同じユーザーの申請が一覧に重複しないようにする）
	CreateRequestIfNone(ctx context.Context, userID uuid.UUID) (bool, error)
	// ListRequests は申請を新しい順に返す（ユーザー情報付き）。status が空なら全ステータス
	ListRequests(ctx context.Context, status string, cursor *time.Time, limit int) ([]model.PasswordResetRequest, *time.Time, error)
	FindRequestByID(ctx context.Context, id uuid.UUID) (*model.PasswordResetRequest, error)
	// ListRequestsByUser はユーザーの申請を新しい順に最大 limit 件返す（ユーザー詳細画面用）
	ListRequestsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.PasswordResetRequest, error)
	// IssueToken はユーザーの未使用トークンを失効させてから新しいトークンを保存し、申請を issued にする（1 トランザクション）
	IssueToken(ctx context.Context, token *model.PasswordResetToken, handledBy string, now time.Time) error
	// ConsumeToken は有効なトークンを使用済みにし、同じトランザクションでパスワードを更新・申請を完了・
	// リフレッシュトークンを全失効させる。トークンが無効なら ErrResetTokenNotActive を返す
	ConsumeToken(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) (uuid.UUID, error)
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

// openResetStatuses は対応中（未完了）の申請ステータス
var openResetStatuses = []string{model.ResetRequestPending, model.ResetRequestIssued}

func (r *passwordResetRepository) CreateRequestIfNone(ctx context.Context, userID uuid.UUID) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 同じユーザーの同時申請で重複しないよう、ユーザー行をロックしてから確認する
		if err := tx.Exec("SELECT id FROM users WHERE id = ? FOR UPDATE", userID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.PasswordResetRequest{}).
			Where("user_id = ? AND status IN ?", userID, openResetStatuses).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		created = true
		return tx.Omit("User").Create(&model.PasswordResetRequest{UserID: userID, Status: model.ResetRequestPending}).Error
	})
	return created, err
}

func (r *passwordResetRepository) ListRequests(ctx context.Context, status string, cursor *time.Time, limit int) ([]model.PasswordResetRequest, *time.Time, error) {
	query := r.db.WithContext(ctx).Model(&model.PasswordResetRequest{}).Preload("User")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	var requests []model.PasswordResetRequest
	if err := query.Order("created_at DESC").Limit(limit + 1).Find(&requests).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *time.Time
	if len(requests) > limit {
		requests = requests[:limit]
		nextCursor = &requests[len(requests)-1].CreatedAt
	}
	return requests, nextCursor, nil
}

func (r *passwordResetRepository) FindRequestByID(ctx context.Context, id uuid.UUID) (*model.PasswordResetRequest, error) {
	var req model.PasswordResetRequest
	err := r.db.WithContext(ctx).Preload("User").Where("id = ?", id).First(&req).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &req, err
}

func (r *passwordResetRepository) ListRequestsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.PasswordResetRequest, error) {
	var requests []model.PasswordResetRequest
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&requests).Error
	return requests, err
}

func (r *passwordResetRepository) IssueToken(ctx context.Context, token *model.PasswordResetToken, handledBy string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 以前に発行したリンクは使えないようにする（最新のリンクだけが有効）
		if err := tx.Model(&model.PasswordResetToken{}).
			Where("user_id = ? AND used_at IS NULL AND revoked_at IS NULL", token.UserID).
			Update("revoked_at", now).Error; err != nil {
			return err
		}
		if err := tx.Omit("User", "Request").Create(token).Error; err != nil {
			return err
		}
		if token.RequestID == nil {
			return nil
		}
		return tx.Model(&model.PasswordResetRequest{}).
			Where("id = ? AND status IN ?", *token.RequestID, openResetStatuses).
			Updates(map[string]interface{}{
				"status":     model.ResetRequestIssued,
				"handled_by": handledBy,
				"handled_at": now,
				"updated_at": now,
			}).Error
	})
}

func (r *passwordResetRepository) ConsumeToken(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 一回限り: 未使用・未失効・期限内のトークンだけを条件付き UPDATE で使用済みにする（同時利用でも 1 回しか成功しない）
		res := tx.Model(&model.PasswordResetToken{}).
			Where("token_hash = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrResetTokenNotActive
		}

		var token model.PasswordResetToken
		if err := tx.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
			return err
		}
		userID = token.UserID

		if err := tx.Model(&model.User{}).Where("id = ?", token.UserID).
			Updates(map[string]interface{}{"password_hash": newPasswordHash, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.PasswordResetRequest{}).
			Where("user_id = ? AND status IN ?", token.UserID, openResetStatuses).
			Updates(map[string]interface{}{"status": model.ResetRequestCompleted, "updated_at": now}).Error; err != nil {
			return err
		}
		// パスワードを変えたので、ほかの端末のログインも無効にする
		return tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", token.UserID).
			Update("revoked_at", now).Error
	})
	return userID, err
}
