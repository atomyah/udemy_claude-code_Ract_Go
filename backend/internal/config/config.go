package config

import (
	"log"
	"os"
	"strings"
)

// Config はアプリケーション設定
type Config struct {
	DatabaseURL           string
	JWTSecret             string
	JWTRefreshSecret      string
	FirebaseCredentials   string
	FirebaseStorageBucket string
	CORSOrigins           string
	Port                  string
	Env                   string
	// SQLLog はテスト環境でもSQLログ（全クエリ）を出力するかどうか（環境変数 SQL_LOG=true）
	SQLLog bool
}

// Load は環境変数から設定を読み込む
func Load() *Config {
	cfg := &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://sns_user:sns_password@db:5432/sns_db?sslmode=disable"),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		JWTRefreshSecret:      getEnv("JWT_REFRESH_SECRET", ""),
		FirebaseCredentials:   getEnv("FIREBASE_CREDENTIALS_JSON", "{}"),
		FirebaseStorageBucket: getEnv("FIREBASE_STORAGE_BUCKET", ""),
		CORSOrigins:           getEnv("CORS_ORIGINS", "http://localhost:5173"),
		Port:                  getEnv("PORT", "8080"),
		Env:                   getEnv("ENV", "development"),
		SQLLog:                getEnv("SQL_LOG", "false") == "true",
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if cfg.JWTRefreshSecret == "" {
		log.Fatal("JWT_REFRESH_SECRET is required")
	}

	return cfg
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsTest はテスト環境（ENV=test）かどうかを返す。
// テスト環境ではSwagger UIを公開せず、AutoMigrateだけ開発環境と同じ扱いにする。
func (c *Config) IsTest() bool {
	return c.Env == "test"
}

// IsProduction は本番環境（ENV=production）かどうかを返す
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// ShouldAutoMigrate はGORMのAutoMigrateを実行してよい環境かどうかを返す。
// 本番環境ではマイグレーションファイル（golang-migrate）を使うため常にfalseになる。
func (c *Config) ShouldAutoMigrate() bool {
	return c.IsDevelopment() || c.IsTest()
}

// ShouldLogSQL はGORMのSQLログ（全クエリ）を出力するかどうかを返す。
// 開発環境では常に出力し、テスト環境では SQL_LOG=true のときだけ出力する（性能計測用）。
// 本番環境ではクエリ内容にユーザーデータが含まれるため常に出力しない。
func (c *Config) ShouldLogSQL() bool {
	return c.IsDevelopment() || (c.IsTest() && c.SQLLog)
}

// ShouldCountQueries はリクエストごとのSQLクエリ数を数えて X-DB-Query-Count ヘッダーで返すかどうかを返す。
// N+1検出・性能計測用のため開発・テスト環境のみ有効にする。
func (c *Config) ShouldCountQueries() bool {
	return c.IsDevelopment() || c.IsTest()
}

// AllowedOrigins はCORSで許可するオリジンをカンマ区切りから分割して返す
func (c *Config) AllowedOrigins() []string {
	origins := strings.Split(c.CORSOrigins, ",")
	trimmed := make([]string, 0, len(origins))
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			trimmed = append(trimmed, o)
		}
	}
	return trimmed
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
