package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-for-middleware-unit-test"

// makeToken はテスト用のアクセストークン（user_id と exp のみ）を生成する。
// userID に UUID 以外の文字列も渡せるよう authtoken.IssueAccessToken を使わずに組み立てる
func makeToken(t *testing.T, secret, userID string, expiresAt time.Time) string {
	t.Helper()

	claims := &authtoken.AccessClaims{
		UserID:           userID,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// newContext は Authorization ヘッダー付きのechoコンテキストを作る
func newContext(authHeader string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	if authHeader != "" {
		req.Header.Set(echo.HeaderAuthorization, authHeader)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// okHandler は認証通過時に200を返すダミーハンドラー
func okHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// decodeError はエラーレスポンスボディをデコードする
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestJWTAuth_ValidToken_SetsUserID(t *testing.T) {
	userID := uuid.New()
	token := makeToken(t, testSecret, userID.String(), time.Now().Add(time.Hour))
	c, rec := newContext("Bearer " + token)

	var capturedID uuid.UUID
	handler := JWTAuth(testSecret)(func(c echo.Context) error {
		capturedID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedID)
}

func TestJWTAuth_MissingToken_Returns401(t *testing.T) {
	c, rec := newContext("")

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body["code"])
	assert.Equal(t, "認証トークンが必要です", body["message"])
}

func TestJWTAuth_MalformedAuthorizationHeader_Returns401(t *testing.T) {
	c, rec := newContext("Token abcdef")

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "MISSING_TOKEN", decodeError(t, rec)["code"])
}

func TestJWTAuth_InvalidSignature_Returns401(t *testing.T) {
	token := makeToken(t, "another-secret", uuid.NewString(), time.Now().Add(15*time.Minute))
	c, rec := newContext("Bearer " + token)

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "INVALID_TOKEN", body["code"])
	assert.Equal(t, "無効なトークンです", body["message"])
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	token := makeToken(t, testSecret, uuid.NewString(), time.Now().Add(-1*time.Minute))
	c, rec := newContext("Bearer " + token)

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "EXPIRED_TOKEN", body["code"])
	assert.Equal(t, "トークンの有効期限が切れています", body["message"])
}

func TestJWTAuth_SubjectIsNotUUID_Returns401(t *testing.T) {
	token := makeToken(t, testSecret, "not-a-uuid", time.Now().Add(15*time.Minute))
	c, rec := newContext("Bearer " + token)

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_TOKEN", decodeError(t, rec)["code"])
}

func TestOptionalJWTAuth_NoToken_PassesThroughWithoutUserID(t *testing.T) {
	c, rec := newContext("")

	var hasUserID bool
	handler := OptionalJWTAuth(testSecret)(func(c echo.Context) error {
		_, hasUserID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUserID)
}

func TestOptionalJWTAuth_InvalidToken_PassesThroughWithoutUserID(t *testing.T) {
	c, rec := newContext("Bearer invalid.token.value")

	var hasUserID bool
	handler := OptionalJWTAuth(testSecret)(func(c echo.Context) error {
		_, hasUserID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUserID)
}

func TestOptionalJWTAuth_ValidToken_SetsUserID(t *testing.T) {
	userID := uuid.New()
	token := makeToken(t, testSecret, userID.String(), time.Now().Add(15*time.Minute))
	c, rec := newContext("Bearer " + token)

	var capturedID uuid.UUID
	handler := OptionalJWTAuth(testSecret)(func(c echo.Context) error {
		capturedID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedID)
}

func TestOptionalJWTAuth_SubjectIsNotUUID_PassesThroughWithoutUserID(t *testing.T) {
	token := makeToken(t, testSecret, "not-a-uuid", time.Now().Add(15*time.Minute))
	c, rec := newContext("Bearer " + token)

	var hasUserID bool
	handler := OptionalJWTAuth(testSecret)(func(c echo.Context) error {
		_, hasUserID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUserID)
}

// newCookieContext は access_token Cookie 付きのechoコンテキストを作る（ブラウザからのリクエスト）
func newCookieContext(token string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.AddCookie(&http.Cookie{Name: authtoken.AccessTokenCookie, Value: token})
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestJWTAuth_TokenInCookie_SetsUserID(t *testing.T) {
	userID := uuid.New()
	c, rec := newCookieContext(makeToken(t, testSecret, userID.String(), time.Now().Add(time.Hour)))

	var capturedID uuid.UUID
	handler := JWTAuth(testSecret)(func(c echo.Context) error {
		capturedID = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedID)
}

func TestJWTAuth_ExpiredTokenInCookie_ReturnsExpiredToken(t *testing.T) {
	c, rec := newCookieContext(makeToken(t, testSecret, uuid.NewString(), time.Now().Add(-time.Second)))

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "EXPIRED_TOKEN", decodeError(t, rec)["code"], "フロントエンドはこの401を受けてリフレッシュする")
}

func TestJWTAuth_TokenWithoutExp_Returns401(t *testing.T) {
	// exp のないトークンは無期限になってしまうため受け付けない
	claims := &authtoken.AccessClaims{UserID: uuid.NewString()}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	require.NoError(t, err)
	c, rec := newCookieContext(token)

	require.NoError(t, JWTAuth(testSecret)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_TOKEN", decodeError(t, rec)["code"])
}

func TestOptionalJWTAuth_TokenInCookie_SetsUserID(t *testing.T) {
	userID := uuid.New()
	c, rec := newCookieContext(makeToken(t, testSecret, userID.String(), time.Now().Add(time.Hour)))

	var capturedID uuid.UUID
	handler := OptionalJWTAuth(testSecret)(func(c echo.Context) error {
		capturedID, _ = c.Get("userID").(uuid.UUID)
		return okHandler(c)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedID)
}
