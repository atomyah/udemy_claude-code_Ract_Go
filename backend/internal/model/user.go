package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash *string   `gorm:"size:255"`
	Handle       string    `gorm:"size:50;uniqueIndex;not null"`
	DisplayName  string    `gorm:"size:50;not null"`
	AvatarURL    *string   `gorm:"size:500"`
	BannerURL    *string   `gorm:"size:500"`
	Bio          *string   `gorm:"size:160"`
	Location     *string   `gorm:"size:30"`
	WebsiteURL   *string   `gorm:"size:100"`
	Birthday     *time.Time
	Theme        string `gorm:"size:10;not null;default:light"`
	Role         string `gorm:"size:10;not null;default:user"`
	IsSuspended  bool   `gorm:"not null;default:false"`
	// Status は登録審査の状態（pending / approved / rejected）。approved のユーザーだけが通常機能を使える
	Status string `gorm:"size:10;not null;default:pending;index;check:chk_users_status,status IN ('pending','approved','rejected')"`
	// ReviewedAt / ReviewedBy は管理画面で承認・拒否した日時と管理者（例: "root:admin" / "admin:yamada"）
	ReviewedAt *time.Time
	ReviewedBy *string `gorm:"size:100"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ユーザーの登録審査ステータス
const (
	UserStatusPending  = "pending"
	UserStatusApproved = "approved"
	UserStatusRejected = "rejected"
)

// IsApproved は管理者に承認済み（通常機能を利用できる）かどうかを返す
func (u *User) IsApproved() bool {
	return u.Status == UserStatusApproved
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
