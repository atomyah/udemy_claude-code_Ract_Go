package handler

import (
	"net/http"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newInteractionHandler(likeSvc *mockLikeSvc) *InteractionHandler {
	return NewInteractionHandler(likeSvc, new(mockRepostSvc), new(mockBookmarkSvc))
}

func TestInteractionHandler_Like_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	likeSvc := new(mockLikeSvc)
	likeSvc.On("Like", mock.Anything, userID, postID).
		Return(&dto.LikeResponse{PostID: postID.String(), LikesCount: 1, IsLiked: true}, nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/like", "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(likeSvc).Like(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.LikeResponse
	decodeJSON(t, rec, &body)
	assert.True(t, body.IsLiked)
	assert.Equal(t, int64(1), body.LikesCount)
}

func TestInteractionHandler_Like_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/like", "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(new(mockLikeSvc)).Like(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "認証が必要です", body.Message)
}

func TestInteractionHandler_Like_AlreadyLiked_Returns409(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	likeSvc := new(mockLikeSvc)
	likeSvc.On("Like", mock.Anything, userID, postID).Return(nil, service.ErrAlreadyLiked)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/like", "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(likeSvc).Like(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "ALREADY_LIKED", body.Code)
	assert.Equal(t, "既にいいねしています", body.Message)
}

func TestInteractionHandler_Like_PostNotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	likeSvc := new(mockLikeSvc)
	likeSvc.On("Like", mock.Anything, userID, postID).Return(nil, repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/like", "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(likeSvc).Like(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "投稿が見つかりません", decodeErrorResponse(t, rec).Message)
}

func TestInteractionHandler_Like_InvalidPostID_Returns404(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/xxx/like", "")
	withAuth(c, uuid.New())
	withParam(c, []string{"id"}, []string{"xxx"})

	require.NoError(t, newInteractionHandler(new(mockLikeSvc)).Like(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "POST_NOT_FOUND", decodeErrorResponse(t, rec).Code)
}

func TestInteractionHandler_Unlike_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	likeSvc := new(mockLikeSvc)
	likeSvc.On("Unlike", mock.Anything, userID, postID).
		Return(&dto.LikeResponse{PostID: postID.String(), LikesCount: 0, IsLiked: false}, nil)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/posts/"+postID.String()+"/like", "")
	withAuth(c, userID)
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(likeSvc).Unlike(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.LikeResponse
	decodeJSON(t, rec, &body)
	assert.False(t, body.IsLiked)
}

func TestInteractionHandler_Unlike_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/posts/"+postID.String()+"/like", "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(new(mockLikeSvc)).Unlike(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestInteractionHandler_Bookmark_Unauthorized(t *testing.T) {
	postID := uuid.New()
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/posts/"+postID.String()+"/bookmark", "")
	withParam(c, []string{"id"}, []string{postID.String()})

	require.NoError(t, newInteractionHandler(new(mockLikeSvc)).Bookmark(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "MISSING_TOKEN", decodeErrorResponse(t, rec).Code)
}

func TestInteractionHandler_GetBookmarks_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/bookmarks", "")

	require.NoError(t, newInteractionHandler(new(mockLikeSvc)).GetBookmarks(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}
