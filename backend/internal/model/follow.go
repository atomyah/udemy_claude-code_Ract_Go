package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Follow struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowerID  uuid.UUID `gorm:"type:uuid;not null;index"`
	FollowingID uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt   time.Time

	Follower  User `gorm:"foreignKey:FollowerID"`
	Following User `gorm:"foreignKey:FollowingID"`
}

func (f *Follow) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
