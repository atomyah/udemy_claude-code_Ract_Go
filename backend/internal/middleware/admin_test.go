package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUserFinder は AdminOnly 用のユーザー取得スタブ
type fakeUserFinder struct {
	user *model.User
	err  error
}

func (f *fakeUserFinder) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return f.user, f.err
}

func TestAdminOnly_AdminRole_PassesThrough(t *testing.T) {
	userID := uuid.New()
	c, rec := newContext("")
	c.Set("userID", userID)

	require.NoError(t, AdminOnly(&fakeUserFinder{user: &model.User{ID: userID, Role: "admin"}})(okHandler)(c))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminOnly_UserRole_Returns403(t *testing.T) {
	userID := uuid.New()
	c, rec := newContext("")
	c.Set("userID", userID)

	require.NoError(t, AdminOnly(&fakeUserFinder{user: &model.User{ID: userID, Role: "user"}})(okHandler)(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "FORBIDDEN", body["code"])
	assert.Equal(t, "管理者権限が必要です", body["message"])
}

func TestAdminOnly_SuspendedAdmin_Returns403(t *testing.T) {
	userID := uuid.New()
	c, rec := newContext("")
	c.Set("userID", userID)

	require.NoError(t, AdminOnly(&fakeUserFinder{user: &model.User{ID: userID, Role: "admin", IsSuspended: true}})(okHandler)(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminOnly_NoUserID_Returns403(t *testing.T) {
	c, rec := newContext("")

	require.NoError(t, AdminOnly(&fakeUserFinder{})(okHandler)(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "FORBIDDEN", decodeError(t, rec)["code"])
}

func TestAdminOnly_UserNotFound_Returns403(t *testing.T) {
	c, rec := newContext("")
	c.Set("userID", uuid.New())

	require.NoError(t, AdminOnly(&fakeUserFinder{err: repository.ErrNotFound})(okHandler)(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminOnly_DBError_Returns500(t *testing.T) {
	c, rec := newContext("")
	c.Set("userID", uuid.New())

	require.NoError(t, AdminOnly(&fakeUserFinder{err: errors.New("db down")})(okHandler)(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec)["code"])
}
