package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrRefreshTokenNotActive はローテーション対象のトークンが既に使用済みまたは失効済みであることを表す
var ErrRefreshTokenNotActive = errors.New("refresh token already used or revoked")

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	// Rotate は oldID を使用済みにし、同じトランザクションで next を保存する。
	// oldID が既に使用済み・失効済みなら ErrRefreshTokenNotActive を返す（同時リクエストでも二重に交換されない）
	Rotate(ctx context.Context, oldID uuid.UUID, next *model.RefreshToken, now time.Time) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, now time.Time) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Omit("User").Create(token).Error
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) Rotate(ctx context.Context, oldID uuid.UUID, next *model.RefreshToken, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 未使用・未失効のときだけ使用済みにする（条件付き UPDATE で同時実行時の二重交換を防ぐ）
		res := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", oldID).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRefreshTokenNotActive
		}
		return tx.Omit("User").Create(next).Error
	})
}

func (r *refreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

func (r *refreshTokenRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}
