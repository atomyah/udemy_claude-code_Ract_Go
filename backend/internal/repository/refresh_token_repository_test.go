package repository

import (
	"context"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newRefreshToken は保存前のリフレッシュトークンレコードを作る
func newRefreshToken(userID, familyID uuid.UUID, hash string) *model.RefreshToken {
	return &model.RefreshToken{
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
}

func findToken(t *testing.T, db *gorm.DB, hash string) *model.RefreshToken {
	t.Helper()
	token, err := NewRefreshTokenRepository(db).FindByHash(context.Background(), hash)
	require.NoError(t, err)
	return token
}

func TestRefreshTokenRepository_CreateAndFindByHash(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()

	token := newRefreshToken(user.ID, uuid.New(), "hash-1")
	require.NoError(t, repo.Create(ctx, token))

	got, err := repo.FindByHash(ctx, "hash-1")
	require.NoError(t, err)
	assert.Equal(t, token.ID, got.ID)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, token.FamilyID, got.FamilyID)
	assert.Nil(t, got.UsedAt)
	assert.Nil(t, got.RevokedAt)

	_, err = repo.FindByHash(ctx, "unknown")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRefreshTokenRepository_HashIsUnique(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRefreshToken(user.ID, uuid.New(), "same-hash")))
	assert.Error(t, repo.Create(ctx, newRefreshToken(user.ID, uuid.New(), "same-hash")))
}

func TestRefreshTokenRepository_Rotate(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()
	family := uuid.New()

	old := newRefreshToken(user.ID, family, "old")
	require.NoError(t, repo.Create(ctx, old))

	// 1回目: 古いトークンが使用済みになり、新しいトークンが保存される
	require.NoError(t, repo.Rotate(ctx, old.ID, newRefreshToken(user.ID, family, "new"), time.Now()))
	assert.NotNil(t, findToken(t, db, "old").UsedAt)
	assert.Nil(t, findToken(t, db, "new").UsedAt)

	// 2回目: 使用済みのトークンは交換できず、新しいトークンも保存されない
	err := repo.Rotate(ctx, old.ID, newRefreshToken(user.ID, family, "another"), time.Now())
	assert.ErrorIs(t, err, ErrRefreshTokenNotActive)
	_, err = repo.FindByHash(ctx, "another")
	assert.ErrorIs(t, err, ErrNotFound, "失敗した交換はロールバックされる")
}

func TestRefreshTokenRepository_Rotate_RevokedTokenCannotBeRotated(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()
	family := uuid.New()

	old := newRefreshToken(user.ID, family, "old")
	require.NoError(t, repo.Create(ctx, old))
	require.NoError(t, repo.RevokeFamily(ctx, family, time.Now()))

	err := repo.Rotate(ctx, old.ID, newRefreshToken(user.ID, family, "new"), time.Now())

	assert.ErrorIs(t, err, ErrRefreshTokenNotActive)
}

func TestRefreshTokenRepository_RevokeFamily_OnlyAffectsThatFamily(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()
	familyA, familyB := uuid.New(), uuid.New()

	require.NoError(t, repo.Create(ctx, newRefreshToken(user.ID, familyA, "a1")))
	require.NoError(t, repo.Create(ctx, newRefreshToken(user.ID, familyA, "a2")))
	require.NoError(t, repo.Create(ctx, newRefreshToken(user.ID, familyB, "b1")))

	require.NoError(t, repo.RevokeFamily(ctx, familyA, time.Now()))

	assert.NotNil(t, findToken(t, db, "a1").RevokedAt)
	assert.NotNil(t, findToken(t, db, "a2").RevokedAt)
	assert.Nil(t, findToken(t, db, "b1").RevokedAt, "別の端末（系列）のログインは失効しない")
}

func TestRefreshTokenRepository_RevokeAllByUser(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	taro := createUser(t, db, "taro")
	hanako := createUser(t, db, "hanako")
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRefreshToken(taro.ID, uuid.New(), "taro-pc")))
	require.NoError(t, repo.Create(ctx, newRefreshToken(taro.ID, uuid.New(), "taro-phone")))
	require.NoError(t, repo.Create(ctx, newRefreshToken(hanako.ID, uuid.New(), "hanako-pc")))

	require.NoError(t, repo.RevokeAllByUser(ctx, taro.ID, time.Now()))

	assert.NotNil(t, findToken(t, db, "taro-pc").RevokedAt)
	assert.NotNil(t, findToken(t, db, "taro-phone").RevokedAt)
	assert.Nil(t, findToken(t, db, "hanako-pc").RevokedAt, "他のユーザーには影響しない")
}

func TestRefreshTokenRepository_DeletedWithUser(t *testing.T) {
	db := setupDB(t)
	repo := NewRefreshTokenRepository(db)
	user := createUser(t, db, "taro")
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newRefreshToken(user.ID, uuid.New(), "hash")))

	require.NoError(t, db.Exec("DELETE FROM users WHERE id = ?", user.ID).Error)

	_, err := repo.FindByHash(ctx, "hash")
	assert.ErrorIs(t, err, ErrNotFound, "ユーザー削除時に ON DELETE CASCADE で消える")
}
