package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Media struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	PostID uuid.UUID `gorm:"type:uuid;not null;index"`
	// StoragePath は Storage 上のオブジェクトパス（posts/{post_id}/{uuid}.jpg）。表示時に署名付きURLへ変換する
	StoragePath string `gorm:"size:500;not null;default:''"`
	// URL は公開URLで保存していた既存行のためだけに残す（新規行は空文字）
	URL       string `gorm:"size:500;not null;default:''"`
	Type      string `gorm:"size:10;not null"`
	SortOrder int    `gorm:"not null;default:0"`
	CreatedAt time.Time
}

func (m *Media) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
