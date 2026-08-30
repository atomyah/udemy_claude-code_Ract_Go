package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// pngBytes はmimetypeがPNGと判定する最小限のバイト列
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

// mp4Bytes はmimetypeがMP4と判定する最小限のバイト列
var mp4Bytes = []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")

// newFileHeaders は実際のmultipartフォームを組み立ててFileHeaderを取得する。
// FileHeaderは内容を読めないと MIME 判定ができないため、手組みではなくフォーム経由で作る
func newFileHeaders(t *testing.T, contents ...[]byte) []*multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for i, content := range contents {
		part, err := writer.CreateFormFile("media", "file"+string(rune('a'+i)))
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	require.NoError(t, err)
	return form.File["media"]
}

// newPostService はモックを注入したPostServiceを組み立てる
func newPostService(postRepo *mockPostRepo, likeRepo *mockLikeRepo, bookmarkRepo *mockBookmarkRepo) PostService {
	return NewPostService(postRepo, new(mockMediaRepo), new(mockHashtagRepo), new(mockUserRepo), likeRepo, bookmarkRepo, new(mockStorageSvc))
}

// expectCounts はToPostResponseが呼ぶ集計系のモックを設定する
func expectCounts(postRepo *mockPostRepo, likeRepo *mockLikeRepo, bookmarkRepo *mockBookmarkRepo, postID uuid.UUID) {
	likeRepo.On("CountByPost", mock.Anything, postID).Return(0, nil)
	postRepo.On("CountComments", mock.Anything, postID).Return(0, nil)
	postRepo.On("CountReposts", mock.Anything, postID).Return(0, nil)
	likeRepo.On("IsLiked", mock.Anything, mock.Anything, postID).Return(false, nil)
	bookmarkRepo.On("IsBookmarked", mock.Anything, mock.Anything, postID).Return(false, nil)
	postRepo.On("FindActiveRepost", mock.Anything, mock.Anything, postID).Return(nil, repository.ErrNotFound)
}

func TestPostService_CreatePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	stored := &model.Post{ID: postID, UserID: userID, Content: "こんにちは"}

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	postRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).
		Run(func(args mock.Arguments) { args.Get(1).(*model.Post).ID = postID }).Return(nil)
	postRepo.On("FindByID", mock.Anything, postID).Return(stored, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	svc := newPostService(postRepo, likeRepo, bookmarkRepo)
	resp, err := svc.CreatePost(context.Background(), userID, "こんにちは", nil)

	require.NoError(t, err)
	assert.Equal(t, postID.String(), resp.ID)
	assert.Equal(t, "こんにちは", resp.Content)
	postRepo.AssertExpectations(t)
}

func TestPostService_CreatePost_TooManyImages(t *testing.T) {
	postRepo := new(mockPostRepo)
	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))

	files := newFileHeaders(t, pngBytes, pngBytes, pngBytes, pngBytes, pngBytes)
	_, err := svc.CreatePost(context.Background(), uuid.New(), "画像5枚", files)

	assert.ErrorIs(t, err, ErrTooManyMedia)
	postRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestPostService_CreatePost_MixedImageAndVideo(t *testing.T) {
	svc := newPostService(new(mockPostRepo), new(mockLikeRepo), new(mockBookmarkRepo))

	files := newFileHeaders(t, pngBytes, mp4Bytes)
	_, err := svc.CreatePost(context.Background(), uuid.New(), "画像と動画", files)

	assert.ErrorIs(t, err, ErrMixedMediaTypes)
}

func TestPostService_CreatePost_UnsupportedFileType(t *testing.T) {
	svc := newPostService(new(mockPostRepo), new(mockLikeRepo), new(mockBookmarkRepo))

	files := newFileHeaders(t, []byte("just a plain text file"))
	_, err := svc.CreatePost(context.Background(), uuid.New(), "テキストファイル", files)

	assert.ErrorIs(t, err, ErrUnsupportedFileType)
}

func TestPostService_CreateWithRefs_CommentDoesNotAllowVideo(t *testing.T) {
	svc := newPostService(new(mockPostRepo), new(mockLikeRepo), new(mockBookmarkRepo))

	postID := uuid.New()
	files := newFileHeaders(t, mp4Bytes)
	_, err := svc.CreateWithRefs(context.Background(), uuid.New(), "動画付き返信", files, &postID, nil, 2, false)

	assert.ErrorIs(t, err, ErrUnsupportedFileType)
}

func TestPostService_UpdatePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	stored := &model.Post{ID: postID, UserID: userID, Content: "編集前"}

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(stored, nil)
	postRepo.On("Update", mock.Anything, stored).Return(nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	svc := newPostService(postRepo, likeRepo, bookmarkRepo)
	resp, err := svc.UpdatePost(context.Background(), postID, userID, "編集後")

	require.NoError(t, err)
	assert.Equal(t, "編集後", resp.Content)
	assert.True(t, resp.IsEdited)
}

func TestPostService_UpdatePost_ForbiddenForOtherUsersPost(t *testing.T) {
	postID := uuid.New()
	owner := uuid.New()
	attacker := uuid.New()

	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: owner}, nil)

	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))
	_, err := svc.UpdatePost(context.Background(), postID, attacker, "乗っ取り")

	assert.ErrorIs(t, err, ErrForbidden)
	postRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPostService_UpdatePost_NotFound(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(nil, repository.ErrNotFound)

	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))
	_, err := svc.UpdatePost(context.Background(), postID, uuid.New(), "編集")

	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPostService_DeletePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()

	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: userID}, nil)
	postRepo.On("SoftDelete", mock.Anything, postID).Return(nil)

	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))
	err := svc.DeletePost(context.Background(), postID, userID)

	require.NoError(t, err)
	// 物理削除ではなく論理削除（SoftDelete）が呼ばれる
	postRepo.AssertCalled(t, "SoftDelete", mock.Anything, postID)
}

func TestPostService_DeletePost_ForbiddenForOtherUsersPost(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(&model.Post{ID: postID, UserID: uuid.New()}, nil)

	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))
	err := svc.DeletePost(context.Background(), postID, uuid.New())

	assert.ErrorIs(t, err, ErrForbidden)
	postRepo.AssertNotCalled(t, "SoftDelete", mock.Anything, mock.Anything)
}

func TestPostService_GetPost_NotFound(t *testing.T) {
	postID := uuid.New()
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, postID).Return(nil, repository.ErrNotFound)

	svc := newPostService(postRepo, new(mockLikeRepo), new(mockBookmarkRepo))
	_, err := svc.GetPost(context.Background(), postID, nil)

	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestPostService_GetExploreTimeline_ReturnsCursor(t *testing.T) {
	postID := uuid.New()
	post := model.Post{ID: postID, UserID: uuid.New(), Content: "タイムライン投稿"}

	postRepo := new(mockPostRepo)
	likeRepo := new(mockLikeRepo)
	bookmarkRepo := new(mockBookmarkRepo)
	next := post.CreatedAt
	postRepo.On("GetExplore", mock.Anything, (*time.Time)(nil), 20).Return([]model.Post{post}, &next, nil)
	expectCounts(postRepo, likeRepo, bookmarkRepo, postID)

	svc := newPostService(postRepo, likeRepo, bookmarkRepo)
	resp, err := svc.GetExploreTimeline(context.Background(), nil, "", 20)

	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.True(t, resp.HasMore)
	require.NotNil(t, resp.NextCursor)
}

func TestPostService_GetExploreTimeline_InvalidCursor(t *testing.T) {
	svc := newPostService(new(mockPostRepo), new(mockLikeRepo), new(mockBookmarkRepo))

	_, err := svc.GetExploreTimeline(context.Background(), nil, "not-a-timestamp", 20)

	assert.Error(t, err)
}
