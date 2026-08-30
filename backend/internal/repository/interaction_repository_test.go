package repository

import (
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLikeRepository_CreateDeleteCount(t *testing.T) {
	db := setupDB(t)
	repo := NewLikeRepository(db)
	ctx := testCtx()
	author := createUser(t, db, "taro")
	liker := createUser(t, db, "hanako")
	post := createPost(t, db, author.ID, "いいねされる投稿")

	require.NoError(t, repo.Create(ctx, liker.ID, post.ID))

	liked, err := repo.IsLiked(ctx, liker.ID, post.ID)
	require.NoError(t, err)
	assert.True(t, liked)

	count, err := repo.CountByPost(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	require.NoError(t, repo.Delete(ctx, liker.ID, post.ID))

	liked, err = repo.IsLiked(ctx, liker.ID, post.ID)
	require.NoError(t, err)
	assert.False(t, liked)

	count, err = repo.CountByPost(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestBookmarkRepository_CreateDeleteAndList(t *testing.T) {
	db := setupDB(t)
	repo := NewBookmarkRepository(db)
	ctx := testCtx()
	author := createUser(t, db, "taro")
	reader := createUser(t, db, "hanako")
	post := createPost(t, db, author.ID, "ブックマークされる投稿")

	require.NoError(t, repo.Create(ctx, reader.ID, post.ID))

	bookmarked, err := repo.IsBookmarked(ctx, reader.ID, post.ID)
	require.NoError(t, err)
	assert.True(t, bookmarked)

	posts, next, err := repo.GetByUser(ctx, reader.ID, nil, 20)
	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "ブックマークされる投稿", posts[0].Content)
	assert.Nil(t, next)

	require.NoError(t, repo.Delete(ctx, reader.ID, post.ID))

	posts, _, err = repo.GetByUser(ctx, reader.ID, nil, 20)
	require.NoError(t, err)
	assert.Empty(t, posts)
}

func TestFollowRepository_CreateDeleteCountAndList(t *testing.T) {
	db := setupDB(t)
	repo := NewFollowRepository(db)
	ctx := testCtx()
	follower := createUser(t, db, "taro")
	following := createUser(t, db, "hanako")

	require.NoError(t, repo.Create(ctx, &model.Follow{FollowerID: follower.ID, FollowingID: following.ID}))

	exists, err := repo.Exists(ctx, follower.ID, following.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	followersCount, err := repo.CountFollowers(ctx, following.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), followersCount)

	followingCount, err := repo.CountFollowing(ctx, follower.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), followingCount)

	followers, _, err := repo.GetFollowers(ctx, following.ID, nil, 20)
	require.NoError(t, err)
	require.Len(t, followers, 1)
	assert.Equal(t, "taro", followers[0].Handle)

	followingList, _, err := repo.GetFollowing(ctx, follower.ID, nil, 20)
	require.NoError(t, err)
	require.Len(t, followingList, 1)
	assert.Equal(t, "hanako", followingList[0].Handle)

	require.NoError(t, repo.Delete(ctx, follower.ID, following.ID))

	exists, err = repo.Exists(ctx, follower.ID, following.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNotificationRepository_CreateListAndMarkRead(t *testing.T) {
	db := setupDB(t)
	repo := NewNotificationRepository(db)
	ctx := testCtx()
	recipient := createUser(t, db, "taro")
	actor := createUser(t, db, "hanako")
	post := createPost(t, db, recipient.ID, "いいねされる投稿")

	require.NoError(t, repo.Create(ctx, &model.Notification{
		UserID: recipient.ID, ActorID: actor.ID, Type: "like", PostID: &post.ID,
	}))

	notifications, next, err := repo.GetByUser(ctx, recipient.ID, nil, 20)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.Equal(t, "like", notifications[0].Type)
	assert.Equal(t, "hanako", notifications[0].Actor.Handle)
	assert.Nil(t, next)

	unread, err := repo.CountUnread(ctx, recipient.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), unread)

	require.NoError(t, repo.MarkAllRead(ctx, recipient.ID))

	unread, err = repo.CountUnread(ctx, recipient.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), unread)
}

func TestHashtagRepository_FindOrCreateAndAttach(t *testing.T) {
	db := setupDB(t)
	repo := NewHashtagRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "今日は #天気 がいい")

	hashtag, err := repo.FindOrCreate(ctx, "天気")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, hashtag.ID)

	// 同名のハッシュタグは重複作成されず既存のものが返る
	same, err := repo.FindOrCreate(ctx, "天気")
	require.NoError(t, err)
	assert.Equal(t, hashtag.ID, same.ID)

	require.NoError(t, repo.AttachToPost(ctx, post.ID, []uuid.UUID{hashtag.ID}))

	ids, err := repo.GetPostIDsByHashtag(ctx, hashtag.ID, nil, 20)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Equal(t, post.ID, ids[0])

	found, err := repo.FindByName(ctx, "天気")
	require.NoError(t, err)
	assert.Equal(t, hashtag.ID, found.ID)

	_, err = repo.FindByName(ctx, "存在しないタグ")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestHashtagRepository_AttachToPost_EmptyIsNoop(t *testing.T) {
	db := setupDB(t)
	repo := NewHashtagRepository(db)
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "タグなし")

	assert.NoError(t, repo.AttachToPost(testCtx(), post.ID, nil))
}

func TestMediaRepository_CreateBulkAndFind(t *testing.T) {
	db := setupDB(t)
	repo := NewMediaRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	post := createPost(t, db, user.ID, "画像付き投稿")

	media := []model.Media{
		{PostID: post.ID, URL: "https://example.com/1.jpg", Type: "image", SortOrder: 0},
		{PostID: post.ID, URL: "https://example.com/2.jpg", Type: "image", SortOrder: 1},
	}
	require.NoError(t, repo.CreateBulk(ctx, media))

	found, err := repo.FindByPostID(ctx, post.ID)
	require.NoError(t, err)
	require.Len(t, found, 2)
	assert.Equal(t, "https://example.com/1.jpg", found[0].URL)

	byIDs, err := repo.FindByPostIDs(ctx, []uuid.UUID{post.ID})
	require.NoError(t, err)
	assert.Len(t, byIDs[post.ID], 2)

	// 空スライスは何もせずエラーにもならない
	require.NoError(t, repo.CreateBulk(ctx, nil))
	empty, err := repo.FindByPostIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestSearchRepository_SearchUsersAndPosts(t *testing.T) {
	db := setupDB(t)
	repo := NewSearchRepository(db)
	ctx := testCtx()
	user := createUser(t, db, "taro")
	suspended := createUser(t, db, "taro_suspended")
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", suspended.ID).Update("is_suspended", true).Error)
	createPost(t, db, user.ID, "Goのテストを書いています")
	createPost(t, db, user.ID, "今日はいい天気")

	users, _, err := repo.SearchUsers(ctx, "taro", nil, 20)
	require.NoError(t, err)
	// 停止ユーザーは検索結果から除外される
	require.Len(t, users, 1)
	assert.Equal(t, "taro", users[0].Handle)

	posts, _, err := repo.SearchPosts(ctx, "テスト", nil, 20)
	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Contains(t, posts[0].Content, "テスト")

	none, _, err := repo.SearchPosts(ctx, "該当なし", nil, 20)
	require.NoError(t, err)
	assert.Empty(t, none)
}
