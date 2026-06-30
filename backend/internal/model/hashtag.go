package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Hashtag struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:100;uniqueIndex;not null"`
	CreatedAt time.Time
}

type PostHashtag struct {
	PostID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	HashtagID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (h *Hashtag) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}
