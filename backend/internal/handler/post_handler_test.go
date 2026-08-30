package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPostHandler_CreatePost_Unauthorized(t *testing.T) {
	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts", map[string]string{"content": "こんにちは"})

	h := NewPostHandler(new(mockPostSvc), new(mockCommentSvc))
	require.NoError(t, h.CreatePost(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "認証が必要です", body.Message)
}

func TestPostHandler_CreatePost_Success(t *testing.T) {
	userID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("CreatePost", mock.Anything, userID, "こんにちは", mock.Anything).
		Return(&dto.PostResponse{ID: uuid.NewString(), Content: "こんにちは"}, nil)

	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts", map[string]string{"content": "こんにちは"})
	withAuth(c, userID)

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).CreatePost(c))

	assert.Equal(t, http.StatusCreated, rec.Code)
	var body dto.PostResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "こんにちは", body.Content)
}

func TestPostHandler_CreatePost_EmptyContent_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts", map[string]string{"content": ""})
	withAuth(c, uuid.New())

	postSvc := new(mockPostSvc)
	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).CreatePost(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "本文を入力してください", body.Message)
	postSvc.AssertNotCalled(t, "CreatePost", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPostHandler_CreatePost_ContentTooLong_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts",
		map[string]string{"content": strings.Repeat("あ", 281)})
	withAuth(c, uuid.New())

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).CreatePost(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "本文は280文字以内で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_CreatePost_TooManyMedia_ReturnsJapaneseMessage(t *testing.T) {
	userID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("CreatePost", mock.Anything, userID, mock.Anything, mock.Anything).
		Return(nil, service.ErrTooManyMedia)

	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts", map[string]string{"content": "画像たくさん"})
	withAuth(c, userID)

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).CreatePost(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "メディアファイルの枚数上限を超えています", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_UpdatePost_ForbiddenForOtherUsersPost(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("UpdatePost", mock.Anything, postID, userID, "編集後").Return(nil, service.ErrForbidden)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/posts/"+postID.String(), `{"content":"編集後"}`)
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).UpdatePost(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "FORBIDDEN", body.Code)
	assert.Equal(t, "自分の投稿のみ編集できます", body.Message)
}

func TestPostHandler_UpdatePost_NotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("UpdatePost", mock.Anything, postID, userID, "編集後").Return(nil, repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/posts/"+postID.String(), `{"content":"編集後"}`)
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).UpdatePost(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "POST_NOT_FOUND", body.Code)
	assert.Equal(t, "投稿が見つかりません", body.Message)
}

func TestPostHandler_UpdatePost_EmptyContent_ReturnsJapaneseMessage(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/posts/"+postID.String(), `{"content":""}`)
	withAuth(c, uuid.New())
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).UpdatePost(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "本文を入力してください", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_UpdatePost_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/posts/"+postID.String(), `{"content":"編集後"}`)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).UpdatePost(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_DeletePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("DeletePost", mock.Anything, postID, userID).Return(nil)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/posts/"+postID.String(), "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).DeletePost(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestPostHandler_DeletePost_ForbiddenForOtherUsersPost(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("DeletePost", mock.Anything, postID, userID).Return(service.ErrForbidden)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/posts/"+postID.String(), "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).DeletePost(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "自分の投稿のみ削除できます", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_DeletePost_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/posts/"+postID.String(), "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).DeletePost(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "MISSING_TOKEN", decodeErrorResponse(t, rec).Code)
}

func TestPostHandler_GetPost_InvalidUUID_Returns404(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/posts/not-a-uuid", "")
	withParam(c, []string{"id"}, []string{"not-a-uuid"})

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).GetPost(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "投稿が見つかりません", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_GetPost_NotFound(t *testing.T) {
	postID := uuid.New()
	postSvc := new(mockPostSvc)
	postSvc.On("GetPost", mock.Anything, postID, (*uuid.UUID)(nil)).Return(nil, repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/posts/"+postID.String(), "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).GetPost(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "POST_NOT_FOUND", decodeErrorResponse(t, rec).Code)
}

func TestPostHandler_GetHome_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/posts/home", "")

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).GetHome(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_GetExplore_Success(t *testing.T) {
	postSvc := new(mockPostSvc)
	postSvc.On("GetExploreTimeline", mock.Anything, (*uuid.UUID)(nil), "", 20).
		Return(&dto.PostListResponse{Data: []dto.PostResponse{{ID: uuid.NewString(), Content: "投稿"}}}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/posts", "")

	require.NoError(t, NewPostHandler(postSvc, new(mockCommentSvc)).GetExplore(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.PostListResponse
	decodeJSON(t, rec, &body)
	require.Len(t, body.Data, 1)
}

func TestPostHandler_CreateComment_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/comments",
		map[string]string{"content": "返信します"})
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), new(mockCommentSvc)).CreateComment(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestPostHandler_CreateComment_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	commentSvc := new(mockCommentSvc)
	commentSvc.On("CreateComment", mock.Anything, userID, postID, "返信します", mock.Anything).
		Return(&dto.PostResponse{ID: uuid.NewString(), Content: "返信します"}, nil)

	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/comments",
		map[string]string{"content": "返信します"})
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), commentSvc).CreateComment(c))

	assert.Equal(t, http.StatusCreated, rec.Code)
	var body dto.PostResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "返信します", body.Content)
}

func TestPostHandler_CreateComment_ParentNotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	commentSvc := new(mockCommentSvc)
	commentSvc.On("CreateComment", mock.Anything, userID, postID, "返信します", mock.Anything).
		Return(nil, repository.ErrNotFound)

	c, rec := newFormContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/comments",
		map[string]string{"content": "返信します"})
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, NewPostHandler(new(mockPostSvc), commentSvc).CreateComment(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "投稿が見つかりません", decodeErrorResponse(t, rec).Message)
}
