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
