package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingUserFinder は呼び出し回数を数えるユーザー取得スタブ
type countingUserFinder struct {
	user  *model.User
	err   error
	calls int
}

func (f *countingUserFinder) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	f.calls++
	return f.user, f.err
}

func validTokenContext(t *testing.T, userID uuid.UUID) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	token := makeToken(t, testSecret, userID.String(), time.Now().Add(time.Hour))
	c, rec := newContext("Bearer " + token)
	return c, rec
}

func TestJWTAuth_SuspendedUser_Returns401(t *testing.T) {
	userID := uuid.New()
	c, rec := validTokenContext(t, userID)
	users := &countingUserFinder{user: &model.User{ID: userID, IsSuspended: true}}

	called := false
	require.NoError(t, JWTAuth(testSecret, users)(func(c echo.Context) error {
		called = true
		return okHandler(c)
	})(c))

	assert.False(t, called, "停止中のユーザーはハンドラーに到達しない")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "ACCOUNT_SUSPENDED", body["code"])
	assert.Equal(t, "このアカウントは停止されています", body["message"])
	assert.Nil(t, c.Get("userID"))
}

func TestJWTAuth_DeletedUser_Returns401(t *testing.T) {
	c, rec := validTokenContext(t, uuid.New())

	require.NoError(t, JWTAuth(testSecret, &countingUserFinder{err: repository.ErrNotFound})(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_TOKEN", decodeError(t, rec)["code"])
}

func TestJWTAuth_DBError_Returns500(t *testing.T) {
	c, rec := validTokenContext(t, uuid.New())

	require.NoError(t, JWTAuth(testSecret, &countingUserFinder{err: errors.New("db down")})(okHandler)(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec)["code"])
}

func TestJWTAuth_InvalidToken_DoesNotQueryDB(t *testing.T) {
	c, rec := newContext("Bearer not-a-jwt")
	users := &countingUserFinder{}

	require.NoError(t, JWTAuth(testSecret, users)(okHandler)(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, users.calls, "署名検証に失敗したトークンで DB を引かない")
}

func TestOptionalJWTAuth_SuspendedUser_IsTreatedAsAnonymous(t *testing.T) {
	userID := uuid.New()
	c, rec := validTokenContext(t, userID)
	users := &countingUserFinder{user: &model.User{ID: userID, IsSuspended: true}}

	var gotUserID any = "unset"
	require.NoError(t, OptionalJWTAuth(testSecret, users)(func(c echo.Context) error {
		gotUserID = c.Get("userID")
		return okHandler(c)
	})(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, gotUserID, "停止中のユーザーは未ログイン扱い")
}

func TestOptionalJWTAuth_DeletedUser_IsTreatedAsAnonymous(t *testing.T) {
	c, rec := validTokenContext(t, uuid.New())

	var gotUserID any = "unset"
	require.NoError(t, OptionalJWTAuth(testSecret, &countingUserFinder{err: repository.ErrNotFound})(func(c echo.Context) error {
		gotUserID = c.Get("userID")
		return okHandler(c)
	})(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, gotUserID)
}

// main.go と同じ順序（任意認証 → 必須認証 → 管理者チェック）でも DB 参照は 1 回だけ
func TestAuthChain_LooksUpUserOnlyOnce(t *testing.T) {
	userID := uuid.New()
	c, rec := validTokenContext(t, userID)
	users := &countingUserFinder{user: &model.User{ID: userID, Role: "admin", Status: model.UserStatusApproved}}

	h := OptionalJWTAuth(testSecret, users)(JWTAuth(testSecret, users)(AdminOnly(users)(okHandler)))
	require.NoError(t, h(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, users.calls)
}

// 停止された管理者は管理者 API にも到達できない（JWTAuth の時点で 401）
func TestAuthChain_SuspendedAdmin_Returns401(t *testing.T) {
	userID := uuid.New()
	c, rec := validTokenContext(t, userID)
	users := &countingUserFinder{user: &model.User{ID: userID, Role: "admin", IsSuspended: true}}

	h := OptionalJWTAuth(testSecret, users)(JWTAuth(testSecret, users)(AdminOnly(users)(okHandler)))
	require.NoError(t, h(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "ACCOUNT_SUSPENDED", decodeError(t, rec)["code"])
}
