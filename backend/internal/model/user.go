package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email        string     `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash *string    `gorm:"size:255"`
	Handle       string     `gorm:"size:50;uniqueIndex;not null"`
	DisplayName  string     `gorm:"size:50;not null"`
	AvatarURL    *string    `gorm:"size:500"`
	BannerURL    *string    `gorm:"size:500"`
	Bio          *string    `gorm:"size:160"`
	Location     *string    `gorm:"size:30"`
	WebsiteURL   *string    `gorm:"size:100"`
	Birthday     *time.Time
	Theme        string     `gorm:"size:10;not null;default:light"`
	Role         string     `gorm:"size:10;not null;default:user"`
	IsSuspended  bool       `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
