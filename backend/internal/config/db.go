package config

import (
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/querycount"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(cfg *Config) *gorm.DB {
	// SQLログ（全クエリ・DEBUG）は開発環境（およびSQL_LOG=trueのテスト環境）のみ出力する。
	// SQLエラー・遅いSQLは全環境で WARN として記録する（本番ではSQL文自体は出さない）
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.NewGormLogger(cfg.ShouldLogSQL()),
	})
	if err != nil {
		logger.L().Fatal().Err(err).Msg("データベース接続失敗")
	}

	if cfg.ShouldCountQueries() {
		if err = querycount.Register(db); err != nil {
			logger.L().Fatal().Err(err).Msg("クエリ数カウンターの登録失敗")
		}
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
			&model.RefreshToken{},
		); err != nil {
			logger.L().Fatal().Err(err).Msg("マイグレーション失敗")
		}
	}

	return db
}
