package config

import (
	"log"

	"github.com/atyahara/sns-backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(cfg *Config) *gorm.DB {
	// SQLログは開発環境のみ全件出力する（テスト・本番では警告以上に絞る）
	logLevel := logger.Warn
	if cfg.IsDevelopment() {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("データベース接続失敗: %v", err)
	}

	if cfg.ShouldAutoMigrate() {
		if err = db.AutoMigrate(
			&model.User{},
			&model.Post{},
			&model.Media{},
			&model.Like{},
			&model.Bookmark{},
			&model.Follow{},
			&model.Hashtag{},
			&model.PostHashtag{},
			&model.Notification{},
		); err != nil {
			log.Fatalf("マイグレーション失敗: %v", err)
		}
	}

	return db
}
