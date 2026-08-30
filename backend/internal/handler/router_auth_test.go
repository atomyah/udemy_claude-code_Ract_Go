package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appMiddleware "github.com/atyahara/sns-backend/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const routerTestSecret = "router-test-secret"

// newProtectedRouter はJWT認証ミドルウェアを適用した保護ルートを組み立てる
func newProtectedRouter(t *testing.T) *echo.Echo {
	t.Helper()

	e := echo.New()
	v1 := e.Group("/api/v1")
	jwtMiddleware := appMiddleware.JWTAuth(routerTestSecret)

	postSvc := new(mockPostSvc)
	postHandler := NewPostHandler(postSvc, new(mockCommentSvc))
	userHandler := newUserHandler(new(mockUserRepo), new(mockUserSvc))

	v1.GET("/posts/home", postHandler.GetHome, jwtMiddleware)
	v1.POST("/posts", postHandler.CreatePost, jwtMiddleware)
	v1.GET("/users/me", userHandler.GetMe, jwtMiddleware)
	v1.GET("/notifications", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, jwtMiddleware)

	return e
}

// TestProtectedEndpoints_WithoutToken_Return401 は
// 認証が必要なエンドポイントにトークンなしでアクセスした場合の挙動を検証する
func TestProtectedEndpoints_WithoutToken_Return401(t *testing.T) {
	e := newProtectedRouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/posts/home"},
		{http.MethodPost, "/api/v1/posts"},
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodGet, "/api/v1/notifications"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			body := decodeErrorResponse(t, rec)
			assert.Equal(t, "MISSING_TOKEN", body.Code)
			assert.Equal(t, "認証トークンが必要です", body.Message)
		})
	}
}

func TestProtectedEndpoint_WithExpiredToken_Returns401(t *testing.T) {
	e := newProtectedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+routerToken(t, uuid.New().String(), time.Now().Add(-time.Minute)))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "EXPIRED_TOKEN", body.Code)
	assert.Equal(t, "トークンの有効期限が切れています", body.Message)
}

func TestProtectedEndpoint_WithValidToken_PassesAuthentication(t *testing.T) {
	e := newProtectedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+routerToken(t, uuid.New().String(), time.Now().Add(15*time.Minute)))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func routerToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"sub": sub, "role": "user", "exp": exp.Unix(), "iat": time.Now().Unix()}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(routerTestSecret))
	require.NoError(t, err)
	return signed
}
