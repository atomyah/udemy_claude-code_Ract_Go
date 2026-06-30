package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Content   string     `gorm:"size:280;not null"`
	IsEdited  bool       `gorm:"not null;default:false"`
	IsDeleted bool       `gorm:"not null;default:false"`
	RepostOf  *uuid.UUID `gorm:"type:uuid;index"`
	ReplyTo   *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt time.Time  `gorm:"index:idx_posts_created_at,sort:desc"`
	UpdatedAt time.Time

	User         User    `gorm:"foreignKey:UserID"`
	Media        []Media `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	RepostOfPost *Post   `gorm:"foreignKey:RepostOf"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
