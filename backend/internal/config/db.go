package config

import (
	"log"

	"github.com/atyahara/sns-backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(cfg *Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("データベース接続失敗: %v", err)
	}

	if cfg.IsDevelopment() {
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
