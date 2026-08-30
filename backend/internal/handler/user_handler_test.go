package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newUserHandler(userRepo *mockUserRepo, userSvc *mockUserSvc) *UserHandler {
	return NewUserHandler(userRepo, new(mockStorageSvc), userSvc, new(mockPostSvc))
}

func TestUserHandler_GetMe_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/users/me", "")

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).GetMe(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "MISSING_TOKEN", body.Code)
	assert.Equal(t, "認証が必要です", body.Message)
}

func TestUserHandler_GetMe_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userSvc := new(mockUserSvc)
	userRepo.On("FindByID", mock.Anything, userID).
		Return(&model.User{ID: userID, Email: "taro@example.com", Handle: "taro", DisplayName: "タロウ"}, nil)
	userSvc.On("GetProfile", mock.Anything, "taro", &userID).
		Return(&dto.UserResponse{ID: userID.String(), Handle: "taro", DisplayName: "タロウ"}, nil)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/users/me", "")
	withAuth(c, userID)

	require.NoError(t, newUserHandler(userRepo, userSvc).GetMe(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.UserResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "taro", body.Handle)
	// 自分のプロフィールにはメールアドレスが含まれる
	require.NotNil(t, body.Email)
	assert.Equal(t, "taro@example.com", *body.Email)
}

func TestUserHandler_UpdateProfile_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me", `{"display_name":"タロウ"}`)

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).UpdateProfile(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*dto.UpdateProfileRequest")).
		Return(&dto.UserResponse{ID: userID.String(), Handle: "taro", DisplayName: "タロウ改"}, nil)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me",
		`{"display_name":"タロウ改","bio":"よろしく","location":"東京","website_url":"https://example.com"}`)
	withAuth(c, userID)

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).UpdateProfile(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body dto.UserResponse
	decodeJSON(t, rec, &body)
	assert.Equal(t, "タロウ改", body.DisplayName)
}

func TestUserHandler_UpdateProfile_BioTooLong_ReturnsJapaneseMessage(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)

	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me",
		`{"display_name":"タロウ","bio":"`+strings.Repeat("あ", 161)+`"}`)
	withAuth(c, userID)

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).UpdateProfile(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Code)
	assert.Equal(t, "自己紹介は160文字以内で入力してください", body.Message)
	userSvc.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_UpdateProfile_DisplayNameTooLong_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me",
		`{"display_name":"`+strings.Repeat("あ", 51)+`"}`)
	withAuth(c, uuid.New())

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).UpdateProfile(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "表示名は50文字以内で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_UpdateTheme_InvalidValue_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me/theme", `{"theme":"blue"}`)
	withAuth(c, uuid.New())

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).UpdateTheme(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "テーマはlight・darkのいずれかを指定してください", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_Follow_Success(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Follow", mock.Anything, userID, "hanako").Return(nil)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/users/hanako/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"hanako"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Follow(c))

	assert.Equal(t, http.StatusOK, rec.Code)
	userSvc.AssertExpectations(t)
}

func TestUserHandler_Follow_Unauthorized(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/users/hanako/follow", "")
	withParam(c, []string{"handle"}, []string{"hanako"})

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).Follow(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "認証が必要です", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_Follow_SelfFollow_Returns400(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Follow", mock.Anything, userID, "taro").Return(service.ErrSelfFollow)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/users/taro/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"taro"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Follow(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "SELF_FOLLOW", body.Code)
	assert.Equal(t, "自分自身をフォローすることはできません", body.Message)
}

func TestUserHandler_Follow_AlreadyFollowed_Returns409(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Follow", mock.Anything, userID, "hanako").Return(service.ErrAlreadyFollowed)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/users/hanako/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"hanako"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Follow(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "既にフォローしています", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_Follow_UserNotFound_Returns404(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Follow", mock.Anything, userID, "unknown").Return(repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodPost, "/api/v1/users/unknown/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"unknown"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Follow(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	body := decodeErrorResponse(t, rec)
	assert.Equal(t, "USER_NOT_FOUND", body.Code)
	assert.Equal(t, "ユーザーが見つかりません", body.Message)
}

func TestUserHandler_Unfollow_Success(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Unfollow", mock.Anything, userID, "hanako").Return(nil)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/users/hanako/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"hanako"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Unfollow(c))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUserHandler_Unfollow_NotFollowing_Returns404(t *testing.T) {
	userID := uuid.New()
	userSvc := new(mockUserSvc)
	userSvc.On("Unfollow", mock.Anything, userID, "hanako").Return(service.ErrNotFollowing)

	c, rec := newJSONContext(t, http.MethodDelete, "/api/v1/users/hanako/follow", "")
	withAuth(c, userID)
	withParam(c, []string{"handle"}, []string{"hanako"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).Unfollow(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "ユーザーが見つかりません", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_GetProfile_NotFound(t *testing.T) {
	userSvc := new(mockUserSvc)
	userSvc.On("GetProfile", mock.Anything, "unknown", (*uuid.UUID)(nil)).Return(nil, repository.ErrNotFound)

	c, rec := newJSONContext(t, http.MethodGet, "/api/v1/users/unknown", "")
	withParam(c, []string{"handle"}, []string{"unknown"})

	require.NoError(t, newUserHandler(new(mockUserRepo), userSvc).GetProfile(c))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "USER_NOT_FOUND", decodeErrorResponse(t, rec).Code)
}

func TestUserHandler_ChangePassword_ShortNewPassword_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me/password",
		`{"current_password":"password123","new_password":"short"}`)
	withAuth(c, uuid.New())

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).ChangePassword(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "新しいパスワードは8文字以上で入力してください", decodeErrorResponse(t, rec).Message)
}

func TestUserHandler_ChangeEmail_InvalidEmail_ReturnsJapaneseMessage(t *testing.T) {
	c, rec := newJSONContext(t, http.MethodPut, "/api/v1/users/me/email",
		`{"new_email":"not-an-email","current_password":"password123"}`)
	withAuth(c, uuid.New())

	require.NoError(t, newUserHandler(new(mockUserRepo), new(mockUserSvc)).ChangeEmail(c))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "新しいメールアドレスの形式が正しくありません", decodeErrorResponse(t, rec).Message)
}
