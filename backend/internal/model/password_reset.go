package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// パスワードリセット申請のステータス
const (
	// ResetRequestPending はユーザーが申請し、管理者の対応待ちの状態
	ResetRequestPending = "pending"
	// ResetRequestIssued は管理者がリセットリンクを発行した状態（ユーザーへの送付は運営が手動で行う）
	ResetRequestIssued = "issued"
	// ResetRequestCompleted はユーザーがリンクからパスワードを再設定した状態
	ResetRequestCompleted = "completed"
)

// PasswordResetRequest はユーザーからのパスワードリセット申請
type PasswordResetRequest struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Status    string    `gorm:"size:20;not null;default:pending;index;check:chk_password_reset_requests_status,status IN ('pending','issued','completed')"`
	HandledBy *string   `gorm:"size:100"`
	HandledAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (r *PasswordResetRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// PasswordResetToken は管理者が発行したリセットリンクのトークン。トークン本体ではなく HMAC ハッシュを保存する。
// 有効期限付き・一回限り（used_at）で、同じユーザーに再発行すると古いトークンは失効（revoked_at）する
type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	RequestID *uuid.UUID `gorm:"type:uuid;index"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time
	RevokedAt *time.Time
	IssuedBy  string `gorm:"size:100;not null"`
	CreatedAt time.Time
	User      User                  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Request   *PasswordResetRequest `gorm:"foreignKey:RequestID;constraint:OnDelete:SET NULL"`
}

func (t *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
