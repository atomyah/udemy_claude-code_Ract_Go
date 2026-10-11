package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func sampleAuthResult() *service.AuthResult {
	return &service.AuthResult{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		User:         dto.UserResponse{ID: uuid.NewString(), Handle: "taro", DisplayName: "タロウ"},
	}
}

func TestAuthHandler_Register_Success(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Register", mock.Anything, mock.AnythingOfType("*dto.RegisterRequest")).Return(sampleAuthResult(), nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusCreated, rec.Code)
	var body dto.AuthResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "taro", body.User.Handle)

	// トークンはボディではなく HttpOnly Cookie で返す（localStorage に保存させない）
	assertAuthCookies(t, rec, "access-token", "refresh-token")
	assertNoTokenInBody(t, rec)
}

func TestAuthHandler_Register_MissingEmail_ReturnsJapaneseMessage(t *testing.T) {
	svc := new(mockAuthSvc)
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "メールアドレスを入力してください", body.Message)
	svc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestAuthHandler_Register_InvalidEmailFormat_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"not-an-email","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "メールアドレスの形式が正しくありません", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_ShortPassword_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"short","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "パスワードは8文字以上で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_ShortHandle_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"password123","handle":"ab","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "ユーザーIDは3文字以上で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_BrokenJSON_Returns400(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register", `{"email":`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "INVALID_REQUEST", body.Code)
	assert.Equal(t, "リクエストの形式が不正です", body.Message)
}

func TestAuthHandler_Register_EmailTaken_Returns409(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Register", mock.Anything, mock.Anything).Return(nil, service.ErrEmailTaken)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "EMAIL_TAKEN", body.Code)
	assert.Equal(t, "このメールアドレスは既に使用されています", body.Message)
}

func TestAuthHandler_Register_HandleTaken_Returns409(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Register", mock.Anything, mock.Anything).Return(nil, service.ErrHandleTaken)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, newTestAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "このハンドルは既に使用されています", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest")).Return(sampleAuthResult(), nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"password123"}`)

	require.NoError(t, newTestAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.AuthResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "taro", body.User.Handle)
	assertAuthCookies(t, rec, "access-token", "refresh-token")
}

func TestAuthHandler_Login_InvalidCredentials_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, service.ErrInvalidCredentials)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"wrong-password"}`)

	require.NoError(t, newTestAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "INVALID_CREDENTIALS", body.Code)
	assert.Equal(t, "メールアドレスまたはパスワードが正しくありません", body.Message)
}

func TestAuthHandler_Login_SuspendedAccount_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, service.ErrAccountSuspended)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"password123"}`)

	require.NoError(t, newTestAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "ACCOUNT_SUSPENDED", body.Code)
	assert.Equal(t, "このアカウントは停止されています", body.Message)
}

func TestAuthHandler_Login_MissingPassword_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login", `{"email":"taro@example.com","password":""}`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Login(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "パスワードを入力してください", decodeErrorResponse(t, rec).Message)
}

// newTestAuthHandler は本番と同じ Cookie 属性（Secure + SameSite=None）の AuthHandler を作る
func newTestAuthHandler(svc service.AuthService) *AuthHandler {
	return NewAuthHandler(svc, AuthCookieConfig{Secure: true, SameSite: http.SameSiteNoneMode})
}

// assertAuthCookies はアクセストークン・リフレッシュトークンが HttpOnly / Secure / SameSite=None の Cookie で返ることを検証する
func assertAuthCookies(t *testing.T, rec *httptest.ResponseRecorder, accessToken, refreshToken string) {
	t.Helper()

	access := findCookie(rec, "access_token")
	require.NotNil(t, access, "access_token Cookie がない")
	assert.Equal(t, accessToken, access.Value)
	assert.True(t, access.HttpOnly, "JavaScript から読めないよう HttpOnly にする")
	assert.True(t, access.Secure)
	assert.Equal(t, http.SameSiteNoneMode, access.SameSite)
	assert.Equal(t, "/api/v1", access.Path)
	assert.Equal(t, 3600, access.MaxAge, "アクセストークンは1時間")

	refresh := findCookie(rec, "refresh_token")
	require.NotNil(t, refresh, "refresh_token Cookie がない")
	assert.Equal(t, refreshToken, refresh.Value)
	assert.True(t, refresh.HttpOnly, "JavaScript から読めないよう HttpOnly にする")
	assert.True(t, refresh.Secure)
	assert.Equal(t, http.SameSiteNoneMode, refresh.SameSite)
	assert.Equal(t, "/api/v1/auth", refresh.Path, "リフレッシュトークンは認証APIにだけ送る")
	assert.Equal(t, 7*24*3600, refresh.MaxAge, "リフレッシュトークンは7日")

	// Set-Cookie ヘッダーの文字列としても属性が付いていること
	for _, h := range rec.Header().Values("Set-Cookie") {
		assert.Contains(t, h, "HttpOnly")
		assert.Contains(t, h, "Secure")
		assert.Contains(t, h, "SameSite=None")
	}
}

// assertNoTokenInBody はレスポンスボディにトークンが含まれないことを検証する（JavaScript から読めないようにする）
func assertNoTokenInBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	body := rec.Body.String()
	assert.NotContains(t, body, "access_token")
	assert.NotContains(t, body, "refresh_token")
	assert.NotContains(t, body, "access-token")
	assert.NotContains(t, body, "refresh-token")
}

// assertAuthCookiesCleared は認証 Cookie が削除（Max-Age<0 で上書き）されたことを検証する
func assertAuthCookiesCleared(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	for name, path := range map[string]string{"access_token": "/api/v1", "refresh_token": "/api/v1/auth"} {
		cookie := findCookie(rec, name)
		require.NotNil(t, cookie, "%s Cookie の削除指示がない", name)
		assert.Equal(t, "", cookie.Value)
		assert.Less(t, cookie.MaxAge, 0)
		assert.Equal(t, path, cookie.Path, "発行時と同じ Path でないと削除されない")
		assert.True(t, cookie.HttpOnly)
	}
}

func TestAuthHandler_Login_SetsHttpOnlyCookies(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest")).Return(sampleAuthResult(), nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"password123"}`)

	require.NoError(t, newTestAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assertAuthCookies(t, rec, "access-token", "refresh-token")
	assertNoTokenInBody(t, rec)
}

func TestAuthHandler_DevelopmentWithoutSecure_UsesSameSiteLax(t *testing.T) {
	// 開発環境で COOKIE_SECURE=false にしたとき（Safari など）は SameSite=Lax になる
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.Anything).Return(sampleAuthResult(), nil)
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"password123"}`)

	h := NewAuthHandler(svc, AuthCookieConfig{Secure: false, SameSite: http.SameSiteLaxMode})
	require.NoError(t, h.Login(c))

	access := findCookie(rec, "access_token")
	require.NotNil(t, access)
	assert.True(t, access.HttpOnly, "Secure を外しても HttpOnly は必ず付ける")
	assert.False(t, access.Secure)
	assert.Equal(t, http.SameSiteLaxMode, access.SameSite)
}

func TestAuthHandler_Logout_RevokesTokenAndClearsCookies(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Logout", mock.Anything, "refresh-token").Return(nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/logout", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token"})

	require.NoError(t, newTestAuthHandler(svc).Logout(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	svc.AssertExpectations(t)
	assertAuthCookiesCleared(t, rec)
}

func TestAuthHandler_Logout_WithoutCookie_StillClearsCookies(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Logout", mock.Anything, "").Return(nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/logout", "")

	require.NoError(t, newTestAuthHandler(svc).Logout(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assertAuthCookiesCleared(t, rec)
}

func TestAuthHandler_Logout_ServiceError_Returns500(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Logout", mock.Anything, "refresh-token").Return(errors.New("db down"))

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/logout", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token"})

	require.NoError(t, newTestAuthHandler(svc).Logout(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "サーバーエラーが発生しました", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_RevokeAll_RevokesAndClearsCookies(t *testing.T) {
	userID := uuid.New()
	svc := new(mockAuthSvc)
	svc.On("RevokeAll", mock.Anything, userID).Return(nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/revoke-all", "")
	withAuth(c, userID)

	require.NoError(t, newTestAuthHandler(svc).RevokeAll(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	svc.AssertExpectations(t)
	assertAuthCookiesCleared(t, rec)
}

func TestAuthHandler_RevokeAll_Unauthenticated_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/revoke-all", "")

	require.NoError(t, newTestAuthHandler(svc).RevokeAll(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	svc.AssertNotCalled(t, "RevokeAll", mock.Anything, mock.Anything)
}

func TestAuthHandler_Refresh_MissingCookie_Returns401(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).Refresh(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "Refresh Tokenが見つかりません", body.Message)
}

func TestAuthHandler_Refresh_Success_RotatesCookies(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Refresh", mock.Anything, "old-refresh-token").Return(&service.AuthResult{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		User:         dto.UserResponse{ID: uuid.NewString(), Handle: "taro"},
	}, nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-refresh-token"})

	require.NoError(t, newTestAuthHandler(svc).Refresh(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assertAuthCookies(t, rec, "new-access-token", "new-refresh-token")
	var body dto.AuthResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "taro", body.User.Handle)
	assert.NotContains(t, rec.Body.String(), "new-access-token")
	assert.NotContains(t, rec.Body.String(), "new-refresh-token")
}

func TestAuthHandler_Refresh_Errors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{"期限切れ", service.ErrExpiredToken, "EXPIRED_TOKEN", "ログインの有効期限が切れました。再度ログインしてください"},
		{"不明なトークン", service.ErrInvalidToken, "INVALID_TOKEN", "ログイン情報が無効です。再度ログインしてください"},
		{"失効済み", service.ErrRevokedToken, "INVALID_TOKEN", "ログイン情報が無効です。再度ログインしてください"},
		{"使用済みトークンの再利用", service.ErrTokenReused, "INVALID_TOKEN", "ログイン情報が無効です。再度ログインしてください"},
		{"アカウント停止", service.ErrAccountSuspended, "ACCOUNT_SUSPENDED", "このアカウントは停止されています"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthSvc)
			svc.On("Refresh", mock.Anything, "token").Return(nil, tt.err)
			c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")
			c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})

			require.NoError(t, newTestAuthHandler(svc).Refresh(c))

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			body := decodeErrorResponse(t, rec)
			assert.Equal(t, tt.code, body.Code)
			assert.Equal(t, tt.message, body.Message)
			// 使えないトークンの Cookie は削除して、以後のリフレッシュを止める
			assertAuthCookiesCleared(t, rec)
		})
	}
}

func TestAuthHandler_Refresh_InternalError_Returns500(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Refresh", mock.Anything, "token").Return(nil, errors.New("db down"))
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "token"})

	require.NoError(t, newTestAuthHandler(svc).Refresh(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Nil(t, findCookie(rec, "refresh_token"), "一時的なエラーでは Cookie を消さない")
}

func TestAuthHandler_GoogleLogin_MissingIDToken_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/google", `{"id_token":""}`)

	require.NoError(t, newTestAuthHandler(new(mockAuthSvc)).GoogleLogin(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "IDトークンを入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_GoogleLogin_InvalidIDToken_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("LoginWithGoogle", mock.Anything, "dummy").Return(nil, service.ErrInvalidIDToken)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/google", `{"id_token":"dummy"}`)

	require.NoError(t, newTestAuthHandler(svc).GoogleLogin(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "Google IDトークンが無効です", decodeErrorResponse(t, rec).Message)
}

// echoのインターフェース互換性を保証する（コンパイル時チェック）
var _ echo.HandlerFunc = (&AuthHandler{}).Register

// ============================================================
// 境界値テスト（登録バリデーション）
// ============================================================

func TestAuthHandler_Register_BoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		handle     string
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "パスワード8文字ちょうどは登録できる",
			password:   strings.Repeat("a", 8),
			handle:     "taro",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "パスワード7文字は拒否される",
			password:   strings.Repeat("a", 7),
			handle:     "taro",
			wantStatus: http.StatusBadRequest,
			wantMsg:    "パスワードは8文字以上で入力してください",
		},
		{
			name:       "パスワード72文字ちょうどは登録できる",
			password:   strings.Repeat("a", 72),
			handle:     "taro",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "パスワード73文字は拒否される",
			password:   strings.Repeat("a", 73),
			handle:     "taro",
			wantStatus: http.StatusBadRequest,
			wantMsg:    "パスワードは72文字以内で入力してください",
		},
		{
			name:       "ハンドル3文字ちょうどは登録できる",
			password:   "password123",
			handle:     "abc",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "ハンドル50文字ちょうどは登録できる",
			password:   "password123",
			handle:     strings.Repeat("a", 50),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "ハンドル51文字は拒否される",
			password:   "password123",
			handle:     strings.Repeat("a", 51),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "ユーザーIDは50文字以内で入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthSvc)
			svc.On("Register", mock.Anything, mock.Anything).Return(sampleAuthResult(), nil)

			body := fmt.Sprintf(`{"email":"taro@example.com","password":%q,"handle":%q,"display_name":"タロウ"}`,
				tt.password, tt.handle)
			c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register", body)

			require.NoError(t, newTestAuthHandler(svc).Register(c))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantMsg != "" {
				assert.Equal(t, tt.wantMsg, decodeErrorResponse(t, rec).Message)
				svc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
			}
		})
	}
}

// TestAuthHandler_Register_MultibytePasswordExceedingBcryptLimit は
// 「72文字以内だがbcryptの72バイト上限を超える」パスワードが
// 500ではなく400と日本語メッセージで返ることを検証する。
func TestAuthHandler_Register_MultibytePasswordExceedingBcryptLimit(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Register", mock.Anything, mock.Anything).Return(nil, service.ErrPasswordTooLong)

	// ひらがな30文字 = 90バイト（文字数バリデーションは通過する）
	password := strings.Repeat("ぱ", 30)
	require.LessOrEqual(t, len([]rune(password)), 72)
	require.Greater(t, len(password), 72)

	body := fmt.Sprintf(`{"email":"taro@example.com","password":%q,"handle":"taro","display_name":"タロウ"}`, password)
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register", body)

	require.NoError(t, newTestAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	errBody := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", errBody.Code)
	assert.Contains(t, errBody.Message, "パスワードが長すぎます")
}
