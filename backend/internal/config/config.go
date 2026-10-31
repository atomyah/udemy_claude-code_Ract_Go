package config

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/atyahara/sns-backend/internal/logger"
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
	// LogLevel はログレベル（debug / info / warn / error）。空のときは開発環境 debug、それ以外 info（logger.ParseLevel）。
	// テスト環境で SQL_LOG=true のときは SQL ログ（DEBUG）を出すため debug になる
	LogLevel string
	// AuthRateLimit は認証系API（ログイン・登録など）の IP あたり・API あたりの上限（回/分）
	AuthRateLimit int
	// GeneralRateLimit は一般APIのユーザー（未認証は IP）あたり・API あたりの上限（回/分）
	GeneralRateLimit int
	// TrustedProxyHops は X-Forwarded-For の右端から数えて信頼するプロキシの段数。
	// 0 のときは X-Forwarded-For を見ず、TCP 接続元アドレスをクライアントIPとして扱う
	TrustedProxyHops int
	// CookieSecure は認証 Cookie に Secure 属性を付けるかどうか。
	// 本番・テストは常に true（SameSite=None）。開発環境だけ COOKIE_SECURE=false で無効化できる（SameSite=Lax になる）
	CookieSecure bool
	// AutoApproveUsers は新規登録ユーザーを承認待ち（pending）にせず approved にするかどうか。
	// E2Eテスト用で、ENV=test のときだけ AUTO_APPROVE_USERS=true で有効になる（開発・本番では常に false）
	AutoApproveUsers bool
	// AdminRootUsername / AdminRootPassword は管理画面のルート管理者のログイン情報（admins テーブルには保存しない）。
	// どちらかが空のときはルート管理者でログインできない
	AdminRootUsername string
	AdminRootPassword string
	// AdminSessionSecret は管理画面のセッション Cookie の署名鍵。空のときは管理画面（/admin）を公開しない
	AdminSessionSecret string
	// FrontendBaseURL はパスワードリセットリンクの生成に使うフロントエンドのURL（例: https://example.web.app）
	FrontendBaseURL string
	// PasswordResetTokenTTL はパスワードリセットリンクの有効期限
	PasswordResetTokenTTL time.Duration
}

// レートリミットの既定値（非機能要件: 認証系 5 回/分、一般 60 回/分）
const (
	DefaultAuthRateLimit    = 5
	DefaultGeneralRateLimit = 60
)

// パスワードリセットリンクの有効期限（分）の既定値・上限
const (
	DefaultPasswordResetTTLMinutes = 24 * 60
	MaxPasswordResetTTLMinutes     = 7 * 24 * 60
)

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
		LogLevel:              getEnv("LOG_LEVEL", ""),
		AuthRateLimit:         DefaultAuthRateLimit,
		GeneralRateLimit:      DefaultGeneralRateLimit,
	}

	// レート上限の緩和はテスト環境のみ許可する（E2Eテストは同一IPから登録・ログインを繰り返すため）。
	// 開発・本番では環境変数を設定しても非機能要件どおりの値を使う
	if cfg.IsTest() {
		cfg.AuthRateLimit = getEnvInt("RATE_LIMIT_AUTH_PER_MINUTE", DefaultAuthRateLimit, 1)
		cfg.GeneralRateLimit = getEnvInt("RATE_LIMIT_GENERAL_PER_MINUTE", DefaultGeneralRateLimit, 1)
	}

	// Cloud Run（本番）では Google Front End が実際の接続元IPを X-Forwarded-For の末尾に追記するため 1 段を信頼する。
	// 前段に外部ロードバランサーを置く場合は TRUSTED_PROXY_HOPS=2 にする。
	// 開発・テストはプロキシを経由しないため既定 0（X-Forwarded-For は偽装できるので見ない）
	defaultHops := 0
	if cfg.IsProduction() {
		defaultHops = 1
	}
	cfg.TrustedProxyHops = getEnvInt("TRUSTED_PROXY_HOPS", defaultHops, 0)

	// Secure Cookie は http://localhost でも Chrome / Edge / Firefox では使えるため既定は全環境で true。
	// localhost を安全なオリジンとして扱わないブラウザ（Safari など）で開発するときだけ COOKIE_SECURE=false にする。
	// フロント（localhost:5173）と API（localhost:8080）はポート違いの同一サイトなので SameSite=Lax でも Cookie は送られる
	cfg.CookieSecure = !(cfg.IsDevelopment() && getEnv("COOKIE_SECURE", "true") == "false")

	// 新規登録ユーザーの自動承認は E2E テスト用。テスト環境以外では環境変数を設定しても必ず pending にする
	cfg.AutoApproveUsers = cfg.IsTest() && getEnv("AUTO_APPROVE_USERS", "false") == "true"

	cfg.AdminRootUsername = strings.TrimSpace(getEnv("ADMIN_ROOT_USERNAME", ""))
	cfg.AdminRootPassword = getEnv("ADMIN_ROOT_PASSWORD", "")
	cfg.AdminSessionSecret = getEnv("ADMIN_SESSION_SECRET", "")
	cfg.FrontendBaseURL = strings.TrimRight(getEnv("FRONTEND_BASE_URL", firstOrEmpty(cfg.AllowedOrigins())), "/")
	ttlMinutes := getEnvInt("PASSWORD_RESET_TOKEN_TTL_MINUTES", DefaultPasswordResetTTLMinutes, 10)
	if ttlMinutes > MaxPasswordResetTTLMinutes {
		ttlMinutes = MaxPasswordResetTTLMinutes
	}
	cfg.PasswordResetTokenTTL = time.Duration(ttlMinutes) * time.Minute

	if cfg.LogLevel == "" && cfg.IsTest() && cfg.SQLLog {
		cfg.LogLevel = "debug"
	}
	// 以降のログ（サーバー・マイグレーションコマンドとも）を環境に合ったレベルの JSON で出す
	logger.Init(cfg.Env, cfg.LogLevel)

	if err := ValidateSecrets(cfg.JWTSecret, cfg.JWTRefreshSecret); err != nil {
		logger.L().Fatal().Err(err).Msg("JWT シークレットの設定が不正です")
	}
	if err := ValidateAdminConfig(cfg); err != nil {
		logger.L().Fatal().Err(err).Msg("管理画面の設定が不正です")
	}
	if cfg.AutoApproveUsers {
		logger.L().Warn().Msg("AUTO_APPROVE_USERS is enabled: new users are approved without admin review (test only)")
	}

	return cfg
}

// MinSecretLength は JWT_SECRET / JWT_REFRESH_SECRET に必要な最小の長さ（HS256 の鍵として 256 ビット以上）
const MinSecretLength = 32

// placeholderSecretMarkers を含む値は .env.example などの見本をそのまま使ったものとみなして拒否する
var placeholderSecretMarkers = []string{"change-me", "change_me", "changeme"}

// ValidateSecrets は JWT の署名鍵が安全に使える値かを検証する。
// 未設定・短すぎる・見本の値のまま・アクセス用とリフレッシュ用が同じ値のときはエラーを返す。
// 値は `openssl rand -base64 48` で生成する。
func ValidateSecrets(jwtSecret, refreshSecret string) error {
	for _, s := range []struct{ name, value string }{
		{"JWT_SECRET", jwtSecret},
		{"JWT_REFRESH_SECRET", refreshSecret},
	} {
		if s.value == "" {
			return fmt.Errorf("%s が設定されていません", s.name)
		}
		if len(s.value) < MinSecretLength {
			return fmt.Errorf("%s は %d 文字以上のランダムな文字列にしてください（openssl rand -base64 48 で生成）", s.name, MinSecretLength)
		}
		lower := strings.ToLower(s.value)
		for _, marker := range placeholderSecretMarkers {
			if strings.Contains(lower, marker) {
				return fmt.Errorf("%s が見本の値のままです。openssl rand -base64 48 で生成した値にしてください", s.name)
			}
		}
	}
	if jwtSecret == refreshSecret {
		return errors.New("JWT_SECRET と JWT_REFRESH_SECRET には異なる値を設定してください")
	}
	return nil
}

// MinAdminRootPasswordLength はルート管理者パスワードに必要な最小の長さ
const MinAdminRootPasswordLength = 12

// ValidateAdminConfig は管理画面の設定を検証する。
// 管理画面を使わない（ADMIN_SESSION_SECRET が空）ときは何も検証しない。
// 使う場合は署名鍵が 32 文字以上・JWT の鍵と別の値であること、ルート管理者のパスワードが十分な長さであることを求める
func ValidateAdminConfig(cfg *Config) error {
	if cfg.AdminSessionSecret == "" {
		return nil
	}
	if len(cfg.AdminSessionSecret) < MinSecretLength {
		return fmt.Errorf("ADMIN_SESSION_SECRET は %d 文字以上のランダムな文字列にしてください（openssl rand -base64 48 で生成）", MinSecretLength)
	}
	if isPlaceholder(cfg.AdminSessionSecret) {
		return errors.New("ADMIN_SESSION_SECRET が見本の値のままです")
	}
	if cfg.AdminSessionSecret == cfg.JWTSecret || cfg.AdminSessionSecret == cfg.JWTRefreshSecret {
		return errors.New("ADMIN_SESSION_SECRET には JWT_SECRET / JWT_REFRESH_SECRET と異なる値を設定してください")
	}
	if (cfg.AdminRootUsername == "") != (cfg.AdminRootPassword == "") {
		return errors.New("ADMIN_ROOT_USERNAME と ADMIN_ROOT_PASSWORD は両方設定してください")
	}
	if cfg.AdminRootPassword != "" {
		if len(cfg.AdminRootPassword) < MinAdminRootPasswordLength {
			return fmt.Errorf("ADMIN_ROOT_PASSWORD は %d 文字以上にしてください", MinAdminRootPasswordLength)
		}
		if isPlaceholder(cfg.AdminRootPassword) {
			return errors.New("ADMIN_ROOT_PASSWORD が見本の値のままです")
		}
	}
	return nil
}

func isPlaceholder(v string) bool {
	lower := strings.ToLower(v)
	for _, marker := range placeholderSecretMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// AdminPanelEnabled は管理画面（/admin）を公開するかどうかを返す（ADMIN_SESSION_SECRET が設定されているとき）
func (c *Config) AdminPanelEnabled() bool {
	return c.AdminSessionSecret != ""
}

// RootAdminEnabled はルート管理者でログインできるかどうかを返す
func (c *Config) RootAdminEnabled() bool {
	return c.AdminRootUsername != "" && c.AdminRootPassword != ""
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

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt は環境変数を整数として読む。未設定・数値でない・min 未満のときは fallback を返す
func getEnvInt(key string, fallback, min int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min {
		logger.L().Warn().Str("key", key).Int("default", fallback).Msg("invalid integer environment variable, using default")
		return fallback
	}
	return n
}

// CookieSameSite は認証 Cookie の SameSite 属性を返す。
// フロントエンドとAPIが別オリジン（本番は別ドメイン）のため None にする。None は Secure 必須なので、
// 開発環境で Secure を外したときだけ Lax にする
func (c *Config) CookieSameSite() http.SameSite {
	if c.CookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
