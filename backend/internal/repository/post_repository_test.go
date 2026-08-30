package repository

import (
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostRepository_CreateAndFindByID(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")

	post := &model.Post{UserID: user.ID, Content: "こんにちは"}
	require.NoError(t, repo.Create(ctx, post))
	assert.NotEqual(t, uuid.Nil, post.ID)

	found, err := repo.FindByID(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, "こんにちは", found.Content)
	// Preloadで投稿者情報も一緒に取得される（N+1防止）
	assert.Equal(t, "taro", found.User.Handle)
}

func TestPostRepository_SoftDelete_ExcludesFromFind(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "削除される投稿")

	require.NoError(t, repo.SoftDelete(ctx, post.ID))

	_, err := repo.FindByID(ctx, post.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	// 物理削除ではなくis_deleted=trueになっているだけ
	var raw model.Post
	require.NoError(t, db.Unscoped().Where("id = ?", post.ID).First(&raw).Error)
	assert.True(t, raw.IsDeleted)
}

func TestPostRepository_Update(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "編集前")

	post.Content = "編集後"
	post.IsEdited = true
	require.NoError(t, repo.Update(ctx, post))

	updated, err := repo.FindByID(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, "編集後", updated.Content)
	assert.True(t, updated.IsEdited)
}

func TestPostRepository_GetExplore_ExcludesDeletedAndSuspended(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()

	active := createUser(t, db, "active")
	suspended := createUser(t, db, "suspended")
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", suspended.ID).Update("is_suspended", true).Error)

	createPost(t, db, active.ID, "表示される投稿")
	deleted := createPost(t, db, active.ID, "削除された投稿")
	require.NoError(t, repo.SoftDelete(ctx, deleted.ID))
	createPost(t, db, suspended.ID, "停止ユーザーの投稿")

	posts, _, err := repo.GetExplore(ctx, nil, 20)

	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "表示される投稿", posts[0].Content)
}

func TestPostRepository_GetExplore_Pagination(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	createPost(t, db, user.ID, "投稿1")
	createPost(t, db, user.ID, "投稿2")
	createPost(t, db, user.ID, "投稿3")

	firstPage, nextCursor, err := repo.GetExplore(ctx, nil, 2)
	require.NoError(t, err)
	assert.Len(t, firstPage, 2)
	require.NotNil(t, nextCursor)

	secondPage, next2, err := repo.GetExplore(ctx, nextCursor, 2)
	require.NoError(t, err)
	assert.Len(t, secondPage, 1)
	assert.Nil(t, next2)
}

func TestPostRepository_GetByUser_ExcludesReplies(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "通常の投稿")
	createReply(t, db, user.ID, post.ID, "返信の投稿")

	posts, _, err := repo.GetByUser(ctx, user.ID, nil, 20)

	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "通常の投稿", posts[0].Content)
}

func TestPostRepository_GetRepliesByUser_OnlyReplies(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "通常の投稿")
	createReply(t, db, user.ID, post.ID, "返信の投稿")

	replies, _, err := repo.GetRepliesByUser(ctx, user.ID, nil, 20)

	require.NoError(t, err)
	require.Len(t, replies, 1)
	assert.Equal(t, "返信の投稿", replies[0].Content)
}

func TestPostRepository_GetCommentsAndCountComments(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	author := createUser(t, db, "taro")
	commenter := createUser(t, db, "hanako")
	post := createPost(t, db, author.ID, "元の投稿")
	createReply(t, db, commenter.ID, post.ID, "コメント1")
	createReply(t, db, commenter.ID, post.ID, "コメント2")

	comments, _, err := repo.GetComments(ctx, post.ID, nil, 20)
	require.NoError(t, err)
	assert.Len(t, comments, 2)

	count, err := repo.CountComments(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestPostRepository_RepostCountAndFindActiveRepost(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	author := createUser(t, db, "taro")
	reposter := createUser(t, db, "hanako")
	original := createPost(t, db, author.ID, "元の投稿")

	repost := &model.Post{UserID: reposter.ID, Content: "", RepostOf: &original.ID}
	require.NoError(t, repo.Create(ctx, repost))

	count, err := repo.CountReposts(ctx, original.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	found, err := repo.FindActiveRepost(ctx, reposter.ID, original.ID)
	require.NoError(t, err)
	assert.Equal(t, repost.ID, found.ID)

	// リポストを取り消すと見つからなくなる
	require.NoError(t, repo.SoftDelete(ctx, repost.ID))
	_, err = repo.FindActiveRepost(ctx, reposter.ID, original.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPostRepository_FindByIDs(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post1 := createPost(t, db, user.ID, "投稿1")
	post2 := createPost(t, db, user.ID, "投稿2")

	posts, err := repo.FindByIDs(ctx, []uuid.UUID{post1.ID, post2.ID})
	require.NoError(t, err)
	assert.Len(t, posts, 2)

	empty, err := repo.FindByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
