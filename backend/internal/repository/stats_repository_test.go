package repository

import (
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/querycount"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostRepository_FindStatsByIDs(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	ctx := testCtx()
	author := createUser(t, db, "taro")
	viewer := createUser(t, db, "hanako")
	other := createUser(t, db, "jiro")
	p1 := createPost(t, db, author.ID, "集計される投稿")
	p2 := createPost(t, db, author.ID, "反応のない投稿")

	likeRepo := NewLikeRepository(db)
	require.NoError(t, likeRepo.Create(ctx, viewer.ID, p1.ID))
	require.NoError(t, likeRepo.Create(ctx, other.ID, p1.ID))
	require.NoError(t, NewBookmarkRepository(db).Create(ctx, viewer.ID, p1.ID))
	createReply(t, db, other.ID, p1.ID, "コメント")
	deletedReply := createReply(t, db, other.ID, p1.ID, "削除済みコメント")
	require.NoError(t, repo.SoftDelete(ctx, deletedReply.ID))
	require.NoError(t, repo.Create(ctx, &model.Post{UserID: viewer.ID, Content: "", RepostOf: &p1.ID}))
	deletedRepost := &model.Post{UserID: other.ID, Content: "", RepostOf: &p1.ID}
	require.NoError(t, repo.Create(ctx, deletedRepost))
	require.NoError(t, repo.SoftDelete(ctx, deletedRepost.ID))

	stats, err := repo.FindStatsByIDs(ctx, []uuid.UUID{p1.ID, p2.ID}, &viewer.ID)
	require.NoError(t, err)

	assert.Equal(t, PostStats{
		PostID: p1.ID, LikesCount: 2, CommentsCount: 1, RepostsCount: 1,
		IsLiked: true, IsBookmarked: true, IsReposted: true,
	}, stats[p1.ID])
	assert.Equal(t, PostStats{PostID: p2.ID}, stats[p2.ID])

	// 閲覧者なし（未ログイン）なら集計値のみで、閲覧者の状態はすべて false
	anonymous, err := repo.FindStatsByIDs(ctx, []uuid.UUID{p1.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, PostStats{PostID: p1.ID, LikesCount: 2, CommentsCount: 1, RepostsCount: 1}, anonymous[p1.ID])

	// 削除済みリポストしかしていないユーザーは is_reposted にならない
	otherView, err := repo.FindStatsByIDs(ctx, []uuid.UUID{p1.ID}, &other.ID)
	require.NoError(t, err)
	assert.True(t, otherView[p1.ID].IsLiked)
	assert.False(t, otherView[p1.ID].IsBookmarked)
	assert.False(t, otherView[p1.ID].IsReposted)

	empty, err := repo.FindStatsByIDs(ctx, nil, &viewer.ID)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// TestPostRepository_TimelineQueryCountDoesNotGrowWithPosts はタイムライン取得
// （GetExplore + FindStatsByIDs）のクエリ本数が投稿件数に比例しない（N+1でない）ことを確認する
func TestPostRepository_TimelineQueryCountDoesNotGrowWithPosts(t *testing.T) {
	db := setupDB(t)
	repo := NewPostRepository(db)
	author := createUser(t, db, "taro")
	viewer := createUser(t, db, "hanako")

	countTimelineQueries := func() int64 {
		ctx := querycount.WithCounter(testCtx())
		posts, _, err := repo.GetExplore(ctx, nil, 20)
		require.NoError(t, err)
		ids := make([]uuid.UUID, len(posts))
		for i := range posts {
			ids[i] = posts[i].ID
			// 投稿者は InnerJoins で同じクエリから読み込まれている
			assert.Equal(t, "taro", posts[i].User.Handle)
		}
		_, err = repo.FindStatsByIDs(ctx, ids, &viewer.ID)
		require.NoError(t, err)
		return querycount.Count(ctx)
	}

	createPost(t, db, author.ID, "1件目")
	fewPosts := countTimelineQueries()

	for i := 0; i < 4; i++ {
		createPost(t, db, author.ID, "追加の投稿")
	}
	morePosts := countTimelineQueries()

	assert.Equal(t, fewPosts, morePosts, "投稿件数が増えてもクエリ本数は変わらないこと")
}

func TestFollowRepository_FindStatsByUserIDs(t *testing.T) {
	db := setupDB(t)
	repo := NewFollowRepository(db)
	ctx := testCtx()
	taro := createUser(t, db, "taro")
	hanako := createUser(t, db, "hanako")
	jiro := createUser(t, db, "jiro")

	require.NoError(t, repo.Create(ctx, &model.Follow{FollowerID: hanako.ID, FollowingID: taro.ID}))
	require.NoError(t, repo.Create(ctx, &model.Follow{FollowerID: jiro.ID, FollowingID: taro.ID}))
	require.NoError(t, repo.Create(ctx, &model.Follow{FollowerID: taro.ID, FollowingID: jiro.ID}))

	stats, err := repo.FindStatsByUserIDs(ctx, []uuid.UUID{taro.ID, hanako.ID}, &hanako.ID)
	require.NoError(t, err)
	assert.Equal(t, UserStats{UserID: taro.ID, FollowersCount: 2, FollowingCount: 1, IsFollowing: true}, stats[taro.ID])
	assert.Equal(t, UserStats{UserID: hanako.ID, FollowersCount: 0, FollowingCount: 1, IsFollowing: false}, stats[hanako.ID])

	anonymous, err := repo.FindStatsByUserIDs(ctx, []uuid.UUID{taro.ID}, nil)
	require.NoError(t, err)
	assert.False(t, anonymous[taro.ID].IsFollowing)
	assert.Equal(t, int64(2), anonymous[taro.ID].FollowersCount)

	empty, err := repo.FindStatsByUserIDs(ctx, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
