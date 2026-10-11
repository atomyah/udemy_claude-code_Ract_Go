// Package main はテスト・開発環境向けのスキーマ適用コマンド。
// docker compose の api_test コンテナが go test を実行する前に自動で走らせる。
package main

import (
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/logger"
)

func main() {
	cfg := config.Load()
	if !cfg.ShouldAutoMigrate() {
		logger.L().Fatal().Str("env", cfg.Env).Msg("マイグレーションコマンドは development / test 環境でのみ実行できます")
	}

	// NewDB は ShouldAutoMigrate() が true のとき AutoMigrate を実行する
	config.NewDB(cfg)
	logger.L().Info().Msg("マイグレーション完了")
}
