package handler

import (
	"net/http"
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

	require.NoError(t, NewAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusCreated, rec.Code)
	var body dto.AuthResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "access-token", body.AccessToken)
	assert.Equal(t, "taro", body.User.Handle)

	// リフレッシュトークンはHttpOnly Cookieで返す（localStorageに保存させない）
	cookie := findCookie(rec, "refresh_token")
	require.NotNil(t, cookie)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, "refresh-token", cookie.Value)
}

func TestAuthHandler_Register_MissingEmail_ReturnsJapaneseMessage(t *testing.T) {
	svc := new(mockAuthSvc)
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, NewAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "メールアドレスを入力してください", body.Message)
	svc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestAuthHandler_Register_InvalidEmailFormat_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"not-an-email","password":"password123","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "メールアドレスの形式が正しくありません", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_ShortPassword_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"short","handle":"taro","display_name":"タロウ"}`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "パスワードは8文字以上で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_ShortHandle_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register",
		`{"email":"taro@example.com","password":"password123","handle":"ab","display_name":"タロウ"}`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Register(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "ユーザーIDは3文字以上で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Register_BrokenJSON_Returns400(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/register", `{"email":`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Register(c))

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

	require.NoError(t, NewAuthHandler(svc).Register(c))

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

	require.NoError(t, NewAuthHandler(svc).Register(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "このハンドルは既に使用されています", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.AnythingOfType("*dto.LoginRequest")).Return(sampleAuthResult(), nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"password123"}`)

	require.NoError(t, NewAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.AuthResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "access-token", body.AccessToken)
}

func TestAuthHandler_Login_InvalidCredentials_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, service.ErrInvalidCredentials)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"taro@example.com","password":"wrong-password"}`)

	require.NoError(t, NewAuthHandler(svc).Login(c))

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

	require.NoError(t, NewAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "ACCOUNT_SUSPENDED", body.Code)
	assert.Equal(t, "このアカウントは停止されています", body.Message)
}

func TestAuthHandler_Login_MissingPassword_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/login", `{"email":"taro@example.com","password":""}`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Login(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "パスワードを入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_Logout_ClearsRefreshTokenCookie(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/logout", "")

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Logout(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	cookie := findCookie(rec, "refresh_token")
	require.NotNil(t, cookie)
	assert.Equal(t, "", cookie.Value)
	assert.True(t, cookie.MaxAge < 0)
}

func TestAuthHandler_Refresh_MissingCookie_Returns401(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).Refresh(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "Refresh Tokenが見つかりません", body.Message)
}

func TestAuthHandler_Refresh_ExpiredToken_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("RefreshAccessToken", mock.Anything, "expired").Return(nil, service.ErrExpiredToken)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired"})

	require.NoError(t, NewAuthHandler(svc).Refresh(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "EXPIRED_TOKEN", body.Code)
	assert.Equal(t, "Refresh Tokenの有効期限が切れています", body.Message)
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("RefreshAccessToken", mock.Anything, "valid").Return(&dto.RefreshResponse{AccessToken: "new-access-token"}, nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/refresh", "")
	c.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid"})

	require.NoError(t, NewAuthHandler(svc).Refresh(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.RefreshResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "new-access-token", body.AccessToken)
}

func TestAuthHandler_GoogleLogin_MissingIDToken_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/google", `{"id_token":""}`)

	require.NoError(t, NewAuthHandler(new(mockAuthSvc)).GoogleLogin(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "IDトークンを入力してください", decodeErrorResponse(t, rec).Message)
}

func TestAuthHandler_GoogleLogin_InvalidIDToken_Returns401(t *testing.T) {
	svc := new(mockAuthSvc)
	svc.On("LoginWithGoogle", mock.Anything, "dummy").Return(nil, service.ErrInvalidIDToken)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/auth/google", `{"id_token":"dummy"}`)

	require.NoError(t, NewAuthHandler(svc).GoogleLogin(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "Google IDトークンが無効です", decodeErrorResponse(t, rec).Message)
}

// echoのインターフェース互換性を保証する（コンパイル時チェック）
var _ echo.HandlerFunc = (&AuthHandler{}).Register
