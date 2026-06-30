package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	ActorID   uuid.UUID  `gorm:"type:uuid;not null"`
	Type      string     `gorm:"size:20;not null"`
	PostID    *uuid.UUID `gorm:"type:uuid;index"`
	IsRead    bool       `gorm:"not null;default:false"`
	CreatedAt time.Time

	User  User  `gorm:"foreignKey:UserID"`
	Actor User  `gorm:"foreignKey:ActorID"`
	Post  *Post `gorm:"foreignKey:PostID"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
