package middleware

import (
	"net/http"
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 管理者の承認待ち・拒否されたユーザーは、有効なトークンを持っていても保護 API を使えない
func TestJWTAuth_NotApprovedUser_Returns401(t *testing.T) {
	for _, status := range []string{model.UserStatusPending, model.UserStatusRejected} {
		userID := uuid.New()
		c, rec := validTokenContext(t, userID)
		users := &countingUserFinder{user: &model.User{ID: userID, Status: status}}

		called := false
		require.NoError(t, JWTAuth(testSecret, users)(func(c echo.Context) error {
			called = true
			return okHandler(c)
		})(c))

		assert.False(t, called, status)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, status)
		body := decodeError(t, rec)
		assert.Equal(t, "ACCOUNT_NOT_APPROVED", body["code"], status)
		assert.Equal(t, "このアカウントは管理者の承認が完了していないため利用できません", body["message"], status)
	}
}

func TestOptionalJWTAuth_NotApprovedUser_IsTreatedAsAnonymous(t *testing.T) {
	userID := uuid.New()
	c, rec := validTokenContext(t, userID)
	users := &countingUserFinder{user: &model.User{ID: userID, Status: model.UserStatusPending}}

	var gotUserID any = "unset"
	require.NoError(t, OptionalJWTAuth(testSecret, users)(func(c echo.Context) error {
		gotUserID = c.Get("userID")
		return okHandler(c)
	})(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, gotUserID, "承認待ちのユーザーは未ログイン扱い")
}
