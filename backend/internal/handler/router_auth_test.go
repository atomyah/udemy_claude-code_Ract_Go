package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	appMiddleware "github.com/atyahara/sns-backend/internal/middleware"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func routerToken(t *testing.T, userID string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"user_id": userID, "exp": exp.Unix()}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(routerTestSecret))
	require.NoError(t, err)
	return signed
}

// newAdminRouter は本番のmain.goと同じ順序でJWT認証 + 管理者ロールチェックを
// 適用した管理者ルートを組み立てる（ミドルウェアの付け忘れを検出するため）
func newAdminRouter(t *testing.T, adminSvc *mockAdminSvc, users *mockUserRepo) *echo.Echo {
	t.Helper()

	e := echo.New()
	v1 := e.Group("/api/v1")
	adminHandler := NewAdminHandler(adminSvc)

	admin := v1.Group("/admin", appMiddleware.JWTAuth(routerTestSecret), appMiddleware.AdminOnly(users))
	admin.DELETE("/posts/:id", adminHandler.AdminDeletePost)
	admin.PUT("/users/:id/suspend", adminHandler.SuspendUser)
	admin.DELETE("/users/:id/suspend", adminHandler.UnsuspendUser)
	admin.GET("/users", adminHandler.ListUsers)

	return e
}

// adminRoutes は管理者専用エンドポイントの一覧
func adminRoutes() []struct {
	method string
	path   string
} {
	id := uuid.New().String()
	return []struct {
		method string
		path   string
	}{
		{http.MethodDelete, "/api/v1/admin/posts/" + id},
		{http.MethodPut, "/api/v1/admin/users/" + id + "/suspend"},
		{http.MethodDelete, "/api/v1/admin/users/" + id + "/suspend"},
		{http.MethodGet, "/api/v1/admin/users"},
	}
}

func TestAdminEndpoints_WithoutToken_Return401(t *testing.T) {
	adminSvc := new(mockAdminSvc)
	e := newAdminRouter(t, adminSvc, new(mockUserRepo))

	for _, route := range adminRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, "MISSING_TOKEN", decodeErrorResponse(t, rec).Code)
		})
	}

	// 認可で弾かれるのでサービス層は一切呼ばれない
	adminSvc.AssertExpectations(t)
}

func TestAdminEndpoints_WithNormalUserToken_Return403(t *testing.T) {
	adminSvc := new(mockAdminSvc)
	// ロールはトークンではなくDBのユーザー情報で判定する
	userID := uuid.New()
	users := new(mockUserRepo)
	users.On("FindByID", mock.Anything, userID).Return(&model.User{ID: userID, Role: "user"}, nil)
	e := newAdminRouter(t, adminSvc, users)
	token := routerToken(t, userID.String(), time.Now().Add(time.Hour))

	for _, route := range adminRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code)
			body := decodeErrorResponse(t, rec)
			assert.Equal(t, "FORBIDDEN", body.Code)
			assert.Equal(t, "管理者権限が必要です", body.Message)
		})
	}

	adminSvc.AssertExpectations(t)
}

func TestAdminEndpoint_WithAdminToken_ReachesHandler(t *testing.T) {
	postID := uuid.New()
	adminSvc := new(mockAdminSvc)
	adminSvc.On("ForceDeletePost", mock.Anything, postID).Return(nil)
	adminID := uuid.New()
	users := new(mockUserRepo)
	users.On("FindByID", mock.Anything, adminID).Return(&model.User{ID: adminID, Role: "admin"}, nil)
	e := newAdminRouter(t, adminSvc, users)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/posts/"+postID.String(), nil)
	req.Header.Set(echo.HeaderAuthorization,
		"Bearer "+routerToken(t, adminID.String(), time.Now().Add(time.Hour)))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	adminSvc.AssertExpectations(t)
}

// TestOptionalAuthEndpoint_WithoutToken_Returns200 は
// 任意認証のエンドポイントがトークンなしでも閲覧できることを検証する
func TestOptionalAuthEndpoint_WithoutToken_Returns200(t *testing.T) {
	postSvc := new(mockPostSvc)
	postSvc.On("GetExploreTimeline", mock.Anything, (*uuid.UUID)(nil), "", 20).
		Return(&dto.PostListResponse{Data: []dto.PostResponse{}}, nil)

	e := echo.New()
	v1 := e.Group("/api/v1")
	v1.GET("/posts", NewPostHandler(postSvc, new(mockCommentSvc)).GetExplore,
		appMiddleware.OptionalJWTAuth(routerTestSecret))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	postSvc.AssertExpectations(t)
}

// TestProtectedEndpoint_WithTokenInCookie_PassesAuthentication は
// ブラウザと同じく access_token Cookie で送ったトークンで認証が通ることを検証する
func TestProtectedEndpoint_WithTokenInCookie_PassesAuthentication(t *testing.T) {
	e := newProtectedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: routerToken(t, uuid.New().String(), time.Now().Add(time.Hour))})
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
