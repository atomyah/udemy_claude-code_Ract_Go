package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Admin は管理画面を利用する管理者ユーザー（ルート管理者が作成する）。
// ルート管理者は環境変数（ADMIN_ROOT_USERNAME / ADMIN_ROOT_PASSWORD）で定義し、このテーブルには保存しない。
// 一般ユーザー（users）とは完全に別のアカウントで、SNS アプリにはログインできない
type Admin struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username     string    `gorm:"size:50;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	// Role は権限の種類。現在は "admin"（ユーザー審査・パスワードリセット・閲覧）のみ。
	// 将来、閲覧専用（"viewer"）などを追加できるよう列だけ用意している
	Role     string `gorm:"size:20;not null;default:admin;check:chk_admins_role,role IN ('admin')"`
	IsActive bool   `gorm:"not null;default:true"`
	// CreatedBy は作成した管理者（例: "root:admin"）
	CreatedBy   string `gorm:"size:100;not null"`
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AdminRoleAdmin は通常管理者のロール
const AdminRoleAdmin = "admin"

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
