package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken はリフレッシュトークンの発行記録。トークン本体ではなくハッシュを保存する。
// FamilyID は 1 回のログインから始まるローテーションの系列を表し、
// 使用済みトークンの再利用（漏洩の兆候）を検知したら同じ系列をまとめて失効させる
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	// UsedAt はローテーションで新しいトークンに交換された日時（使用済み）
	UsedAt *time.Time
	// RevokedAt はログアウト・全デバイスログアウト・再利用検知などで失効した日時
	RevokedAt *time.Time
	CreatedAt time.Time
	User      User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
