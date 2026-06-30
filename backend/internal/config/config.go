package config

import (
	"log"
	"os"
)

// Config はアプリケーション設定
type Config struct {
	DatabaseURL          string
	JWTSecret            string
	JWTRefreshSecret     string
	FirebaseCredentials  string
	CORSOrigins          string
	Port                 string
	Env                  string
}

// Load は環境変数から設定を読み込む
func Load() *Config {
	cfg := &Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://sns_user:sns_password@db:5432/sns_db?sslmode=disable"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTRefreshSecret:    getEnv("JWT_REFRESH_SECRET", ""),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS_JSON", "{}"),
		CORSOrigins:         getEnv("CORS_ORIGINS", "http://localhost:5173"),
		Port:                getEnv("PORT", "8080"),
		Env:                 getEnv("ENV", "development"),
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
