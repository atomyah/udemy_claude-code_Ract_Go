package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_EnvironmentFlags(t *testing.T) {
	tests := []struct {
		env             string
		isDevelopment   bool
		isTest          bool
		isProduction    bool
		shouldMigrate   bool
		swaggerExpected bool
	}{
		{env: "development", isDevelopment: true, shouldMigrate: true, swaggerExpected: true},
		{env: "test", isTest: true, shouldMigrate: true},
		{env: "production", isProduction: true},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}

			assert.Equal(t, tt.isDevelopment, cfg.IsDevelopment())
			assert.Equal(t, tt.isTest, cfg.IsTest())
			assert.Equal(t, tt.isProduction, cfg.IsProduction())
			// AutoMigrateは開発・テスト環境のみ。本番はマイグレーションファイルを使う
			assert.Equal(t, tt.shouldMigrate, cfg.ShouldAutoMigrate())
			// Swagger UIは開発環境のみ公開する（テスト・本番では非公開）
			assert.Equal(t, tt.swaggerExpected, cfg.IsDevelopment())
		})
	}
}

func TestConfig_SQLLogAndQueryCountFlags(t *testing.T) {
	tests := []struct {
		name             string
		env              string
		sqlLog           bool
		wantLogSQL       bool
		wantCountQueries bool
	}{
		{name: "開発環境は常にSQLログを出す", env: "development", wantLogSQL: true, wantCountQueries: true},
		{name: "テスト環境はSQL_LOG未指定ならSQLログを出さない", env: "test", wantCountQueries: true},
		{name: "テスト環境はSQL_LOG=trueでSQLログを出す", env: "test", sqlLog: true, wantLogSQL: true, wantCountQueries: true},
		{name: "本番環境はSQL_LOG=trueでもSQLログ・クエリ数を出さない", env: "production", sqlLog: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Env: tt.env, SQLLog: tt.sqlLog}

			assert.Equal(t, tt.wantLogSQL, cfg.ShouldLogSQL())
			assert.Equal(t, tt.wantCountQueries, cfg.ShouldCountQueries())
		})
	}
}

func TestLoad_ReadsSQLLog(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")

	t.Setenv("SQL_LOG", "true")
	assert.True(t, Load().SQLLog)

	t.Setenv("SQL_LOG", "")
	assert.False(t, Load().SQLLog)
}

func TestConfig_AllowedOrigins(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "単一オリジン", input: "http://localhost:5173", want: []string{"http://localhost:5173"}},
		{
			name:  "カンマ区切りの複数オリジン",
			input: "http://localhost:5173,http://localhost:5174",
			want:  []string{"http://localhost:5173", "http://localhost:5174"},
		},
		{name: "空白を含む場合はトリムされる", input: " http://a.example.com , http://b.example.com ", want: []string{"http://a.example.com", "http://b.example.com"}},
		{name: "空文字は除外される", input: "http://localhost:5173,,", want: []string{"http://localhost:5173"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{CORSOrigins: tt.input}
			assert.Equal(t, tt.want, cfg.AllowedOrigins())
		})
	}
}

func TestGetEnv_FallbackWhenUnset(t *testing.T) {
	assert.Equal(t, "fallback", getEnv("SNS_TEST_UNDEFINED_ENV_KEY", "fallback"))

	t.Setenv("SNS_TEST_DEFINED_ENV_KEY", "value")
	assert.Equal(t, "value", getEnv("SNS_TEST_DEFINED_ENV_KEY", "fallback"))
}

func TestLoad_ReadsEnvironmentVariables(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("ENV", "test")
	t.Setenv("PORT", "9999")
	t.Setenv("CORS_ORIGINS", "http://localhost:5174")

	cfg := Load()

	assert.Equal(t, "secret", cfg.JWTSecret)
	assert.Equal(t, "refresh-secret", cfg.JWTRefreshSecret)
	assert.Equal(t, "9999", cfg.Port)
	assert.True(t, cfg.IsTest())
	assert.Equal(t, []string{"http://localhost:5174"}, cfg.AllowedOrigins())
}

func TestLoad_RateLimitDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("RATE_LIMIT_AUTH_PER_MINUTE", "")
	t.Setenv("RATE_LIMIT_GENERAL_PER_MINUTE", "")
	t.Setenv("TRUSTED_PROXY_HOPS", "")

	for _, env := range []string{"development", "test", "production"} {
		t.Setenv("ENV", env)
		cfg := Load()
		assert.Equal(t, 5, cfg.AuthRateLimit, env)
		assert.Equal(t, 60, cfg.GeneralRateLimit, env)
	}
}

func TestLoad_RateLimitOverrideOnlyInTest(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("RATE_LIMIT_AUTH_PER_MINUTE", "1000")
	t.Setenv("RATE_LIMIT_GENERAL_PER_MINUTE", "10000")

	t.Setenv("ENV", "test")
	cfg := Load()
	assert.Equal(t, 1000, cfg.AuthRateLimit)
	assert.Equal(t, 10000, cfg.GeneralRateLimit)

	// 開発・本番では環境変数で緩和できない
	for _, env := range []string{"development", "production"} {
		t.Setenv("ENV", env)
		cfg := Load()
		assert.Equal(t, DefaultAuthRateLimit, cfg.AuthRateLimit, env)
		assert.Equal(t, DefaultGeneralRateLimit, cfg.GeneralRateLimit, env)
	}

	// 不正値は既定値に戻す
	t.Setenv("ENV", "test")
	t.Setenv("RATE_LIMIT_AUTH_PER_MINUTE", "0")
	t.Setenv("RATE_LIMIT_GENERAL_PER_MINUTE", "abc")
	cfg = Load()
	assert.Equal(t, DefaultAuthRateLimit, cfg.AuthRateLimit)
	assert.Equal(t, DefaultGeneralRateLimit, cfg.GeneralRateLimit)
}

func TestLoad_TrustedProxyHops(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("TRUSTED_PROXY_HOPS", "")

	// 本番（Cloud Run）は 1 段、開発・テストは X-Forwarded-For を信頼しない
	t.Setenv("ENV", "production")
	assert.Equal(t, 1, Load().TrustedProxyHops)
	t.Setenv("ENV", "development")
	assert.Equal(t, 0, Load().TrustedProxyHops)
	t.Setenv("ENV", "test")
	assert.Equal(t, 0, Load().TrustedProxyHops)

	t.Setenv("ENV", "production")
	t.Setenv("TRUSTED_PROXY_HOPS", "2")
	assert.Equal(t, 2, Load().TrustedProxyHops)
	t.Setenv("TRUSTED_PROXY_HOPS", "-1")
	assert.Equal(t, 1, Load().TrustedProxyHops, "負の値は既定値に戻す")
}
