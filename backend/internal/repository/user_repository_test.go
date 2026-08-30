package repository

import (
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()

	user := &model.User{Email: "taro@example.com", Handle: "taro", DisplayName: "タロウ", Theme: "light", Role: "user"}
	require.NoError(t, repo.Create(ctx, user))
	// BeforeCreateフックでUUIDが採番される
	assert.NotEqual(t, uuid.Nil, user.ID)

	byID, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "taro", byID.Handle)

	byEmail, err := repo.FindByEmail(ctx, "taro@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, byEmail.ID)

	byHandle, err := repo.FindByHandle(ctx, "taro")
	require.NoError(t, err)
	assert.Equal(t, user.ID, byHandle.ID)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByID(testCtx(), uuid.New())

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUserRepository_FindByEmailAndHandle_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()

	_, emailErr := repo.FindByEmail(ctx, "nobody@example.com")
	_, handleErr := repo.FindByHandle(ctx, "nobody")

	assert.ErrorIs(t, emailErr, ErrNotFound)
	assert.ErrorIs(t, handleErr, ErrNotFound)
}

func TestUserRepository_Exists(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()
	createUser(t, db, "taro")

	emailExists, err := repo.ExistsByEmail(ctx, "taro@example.com")
	require.NoError(t, err)
	assert.True(t, emailExists)

	handleExists, err := repo.ExistsByHandle(ctx, "taro")
	require.NoError(t, err)
	assert.True(t, handleExists)

	notExists, err := repo.ExistsByHandle(ctx, "hanako")
	require.NoError(t, err)
	assert.False(t, notExists)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")

	user.DisplayName = "タロウ改"
	user.Theme = "dark"
	require.NoError(t, repo.Update(ctx, user))

	updated, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "タロウ改", updated.DisplayName)
	assert.Equal(t, "dark", updated.Theme)
}

func TestUserRepository_SuspendAndUnsuspend(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")

	require.NoError(t, repo.Suspend(ctx, user.ID))
	suspended, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, suspended.IsSuspended)

	require.NoError(t, repo.Unsuspend(ctx, user.ID))
	unsuspended, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, unsuspended.IsSuspended)
}

func TestUserRepository_FindAll_Pagination(t *testing.T) {
	db := setupDB(t)
	repo := NewUserRepository(db)
	ctx := testCtx()
	createUser(t, db, "user1")
	createUser(t, db, "user2")
	createUser(t, db, "user3")

	firstPage, nextCursor, err := repo.FindAll(ctx, nil, 2)
	require.NoError(t, err)
	assert.Len(t, firstPage, 2)
	require.NotNil(t, nextCursor)

	secondPage, next2, err := repo.FindAll(ctx, nextCursor, 2)
	require.NoError(t, err)
	assert.Len(t, secondPage, 1)
	assert.Nil(t, next2)
}
