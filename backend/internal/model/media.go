package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Media struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index"`
	URL       string    `gorm:"size:500;not null"`
	Type      string    `gorm:"size:10;not null"`
	SortOrder int       `gorm:"not null;default:0"`
	CreatedAt time.Time
}

func (m *Media) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
