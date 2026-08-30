// Package main はテスト・開発環境向けのスキーマ適用コマンド。
// docker compose の api_test コンテナが go test を実行する前に自動で走らせる。
package main

import (
	"log"

	"github.com/atyahara/sns-backend/internal/config"
)

func main() {
	cfg := config.Load()
	if !cfg.ShouldAutoMigrate() {
		log.Fatalf("マイグレーションコマンドは development / test 環境でのみ実行できます (ENV=%s)", cfg.Env)
	}

	// NewDB は ShouldAutoMigrate() が true のとき AutoMigrate を実行する
	config.NewDB(cfg)
	log.Println("マイグレーション完了")
}
