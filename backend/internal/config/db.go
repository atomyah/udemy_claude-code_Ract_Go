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
		if err = addUserStatusColumn(db); err != nil {
			logger.L().Fatal().Err(err).Msg("マイグレーション失敗")
		}
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
			&model.Admin{},
			&model.PasswordResetRequest{},
			&model.PasswordResetToken{},
		); err != nil {
			logger.L().Fatal().Err(err).Msg("マイグレーション失敗")
		}
	}

	return db
}

// addUserStatusColumn は既存の users テーブルに status 列がなければ、既存ユーザーを approved として追加する。
// AutoMigrate に任せると列の既定値（pending）で追加され、開発中のユーザーが全員ログインできなくなるため先に追加する
// （migrations/008_admin_panel.up.sql と同じ手順）
func addUserStatusColumn(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable(&model.User{}) || m.HasColumn(&model.User{}, "Status") {
		return nil
	}
	if err := db.Exec("ALTER TABLE users ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'approved'").Error; err != nil {
		return err
	}
	return db.Exec("ALTER TABLE users ALTER COLUMN status SET DEFAULT 'pending'").Error
}
