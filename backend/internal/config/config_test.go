package config

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)

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
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)
	t.Setenv("ENV", "test")
	t.Setenv("PORT", "9999")
	t.Setenv("CORS_ORIGINS", "http://localhost:5174")

	cfg := Load()

	assert.Equal(t, testJWTSecret, cfg.JWTSecret)
	assert.Equal(t, testRefreshSecret, cfg.JWTRefreshSecret)
	assert.Equal(t, "9999", cfg.Port)
	assert.True(t, cfg.IsTest())
	assert.Equal(t, []string{"http://localhost:5174"}, cfg.AllowedOrigins())
}

func TestLoad_RateLimitDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)
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
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)
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
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)
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

func TestLoad_CookieSecure(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)

	// 既定は全環境で Secure + SameSite=None
	t.Setenv("COOKIE_SECURE", "")
	for _, env := range []string{"development", "test", "production"} {
		t.Setenv("ENV", env)
		cfg := Load()
		assert.True(t, cfg.CookieSecure, env)
		assert.Equal(t, http.SameSiteNoneMode, cfg.CookieSameSite(), env)
	}

	// 開発環境だけ COOKIE_SECURE=false で Secure を外せる（SameSite=Lax になる）
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ENV", "development")
	cfg := Load()
	assert.False(t, cfg.CookieSecure)
	assert.Equal(t, http.SameSiteLaxMode, cfg.CookieSameSite())

	// 本番・テストでは設定しても無視する
	for _, env := range []string{"test", "production"} {
		t.Setenv("ENV", env)
		assert.True(t, Load().CookieSecure, env)
	}
}

// テスト用の署名鍵（ValidateSecrets を満たす 32 文字以上・互いに異なる値）
const (
	testJWTSecret     = "config-test-jwt-secret-0123456789abcdef"
	testRefreshSecret = "config-test-refresh-secret-0123456789abcdef"
)

func TestValidateSecrets_AcceptsLongDistinctRandomValues(t *testing.T) {
	assert.NoError(t, ValidateSecrets(testJWTSecret, testRefreshSecret))
	assert.NoError(t, ValidateSecrets(strings.Repeat("a", MinSecretLength), strings.Repeat("b", MinSecretLength)), "32 文字ちょうどは許可")
}

func TestValidateSecrets_RejectsUnsafeValues(t *testing.T) {
	cases := []struct {
		name, jwt, refresh, wantMsg string
	}{
		{"JWT_SECRET 未設定", "", testRefreshSecret, "JWT_SECRET が設定されていません"},
		{"JWT_REFRESH_SECRET 未設定", testJWTSecret, "", "JWT_REFRESH_SECRET が設定されていません"},
		{"JWT_SECRET が短い", strings.Repeat("a", MinSecretLength-1), testRefreshSecret, "JWT_SECRET は 32 文字以上"},
		{"JWT_REFRESH_SECRET が短い", testJWTSecret, "short-secret", "JWT_REFRESH_SECRET は 32 文字以上"},
		{".env.example の見本のまま", "change-me-to-a-random-32-chars-string", testRefreshSecret, "見本の値のまま"},
		{"見本（大文字・アンダースコア）", testJWTSecret, "CHANGE_ME_CHANGE_ME_CHANGE_ME_CHANGE_ME", "見本の値のまま"},
		{"同じ値", testJWTSecret, testJWTSecret, "異なる値"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSecrets(tc.jwt, tc.refresh)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

// 新規登録ユーザーの自動承認はテスト環境（ENV=test）でだけ有効になる
func TestLoad_AutoApproveUsersOnlyInTest(t *testing.T) {
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("JWT_REFRESH_SECRET", testRefreshSecret)
	t.Setenv("AUTO_APPROVE_USERS", "true")

	for env, want := range map[string]bool{"test": true, "development": false, "production": false} {
		t.Setenv("ENV", env)
		assert.Equal(t, want, Load().AutoApproveUsers, env)
	}

	t.Setenv("ENV", "test")
	t.Setenv("AUTO_APPROVE_USERS", "")
	assert.False(t, Load().AutoApproveUsers, "未設定なら自動承認しない")
}

func TestValidateAdminConfig(t *testing.T) {
	valid := func() *Config {
		return &Config{
			JWTSecret:          testJWTSecret,
			JWTRefreshSecret:   testRefreshSecret,
			AdminSessionSecret: "admin_session_secret_for_unit_test_32bytes",
			AdminRootUsername:  "root",
			AdminRootPassword:  "root-password-123",
		}
	}

	assert.NoError(t, ValidateAdminConfig(valid()), "正しい設定")
	assert.NoError(t, ValidateAdminConfig(&Config{}), "管理画面を使わない（署名鍵が空）ときは検証しない")

	cases := map[string]func(c *Config){
		"署名鍵が短い":          func(c *Config) { c.AdminSessionSecret = "short" },
		"署名鍵が JWT と同じ":    func(c *Config) { c.AdminSessionSecret = c.JWTSecret },
		"ルートのパスワードだけ未設定":  func(c *Config) { c.AdminRootPassword = "" },
		"ルートのパスワードが短い":    func(c *Config) { c.AdminRootPassword = "short" },
		"ルートのパスワードが見本のまま": func(c *Config) { c.AdminRootPassword = "CHANGE_ME_please_123" },
	}
	for name, mutate := range cases {
		c := valid()
		mutate(c)
		assert.Error(t, ValidateAdminConfig(c), name)
	}
}
