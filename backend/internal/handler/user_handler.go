package handler

import (
	"errors"
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	errUnauthorized = errors.New("unauthorized")
	errMissingFile  = errors.New("missing file")
)

const (
	defaultListLimit = 20
	maxListLimit     = 50
)

// UserHandler はユーザー・フォロー関連のHTTPハンドラー
type UserHandler struct {
	userRepo   repository.UserRepository
	storageSvc service.StorageService
	userSvc    service.UserService
	postSvc    service.PostService
	validate   *validator.Validate
}

// NewUserHandler はUserHandlerを生成する
func NewUserHandler(userRepo repository.UserRepository, storageSvc service.StorageService, userSvc service.UserService, postSvc service.PostService) *UserHandler {
	return &UserHandler{userRepo: userRepo, storageSvc: storageSvc, userSvc: userSvc, postSvc: postSvc, validate: validator.New()}
}

// optionalUserID はコンテキストからログイン済みユーザーIDを取得する（未ログインならnil）
func optionalUserID(c echo.Context) *uuid.UUID {
	if id, ok := c.Get("userID").(uuid.UUID); ok {
		return &id
	}
	return nil
}

// GetMe godoc
// @Summary      自分のプロフィール取得
// @Description  JWTから取得したユーザーID自身のプロフィールを返す
// @Tags         users
// @Produce      json
// @Success      200  {object} dto.UserResponse   "取得成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /users/me [get]
// @Security     BearerAuth
func (h *UserHandler) GetMe(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	user, err := h.userRepo.FindByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ユーザー情報の取得に失敗しました"})
	}

	resp, err := h.userSvc.GetProfile(c.Request().Context(), user.Handle, &userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ユーザー情報の取得に失敗しました"})
	}
	resp.Email = &user.Email

	return c.JSON(http.StatusOK, resp)
}

// UpdateProfile godoc
// @Summary      自分のプロフィール更新
// @Description  表示名・bio・場所・ウェブサイトURL・誕生日を更新する
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     dto.UpdateProfileRequest  true  "更新内容"
// @Success      200   {object} dto.UserResponse           "更新成功"
// @Failure      400   {object} dto.ErrorResponse          "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse          "未認証"
// @Failure      500   {object} dto.ErrorResponse          "サーバーエラー"
// @Router       /users/me [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	var req dto.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	resp, err := h.userSvc.UpdateProfile(c.Request().Context(), userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "更新に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateAvatar godoc
// @Summary      アバター画像更新
// @Description  アバター画像をFirebase Storageにアップロードし、URLをDBに保存する
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar  formData  file             true  "アバター画像（JPEG/PNG/WebP、最大5MB）"
// @Success      200     {object}  dto.AvatarResponse  "アップロード成功"
// @Failure      400     {object}  dto.ErrorResponse   "ファイル形式またはサイズが不正"
// @Failure      401     {object}  dto.ErrorResponse   "未認証"
// @Failure      500     {object}  dto.ErrorResponse   "サーバーエラー"
// @Router       /users/me/avatar [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateAvatar(c echo.Context) error {
	url, err := h.uploadUserImage(c, "avatar", "avatars")
	if err != nil {
		return h.respondUploadError(c, err)
	}

	userID := c.Get("userID").(uuid.UUID)
	user, err := h.userRepo.FindByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ユーザー情報の取得に失敗しました"})
	}
	user.AvatarURL = &url
	if err := h.userRepo.Update(c.Request().Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "保存に失敗しました"})
	}

	return c.JSON(http.StatusOK, dto.AvatarResponse{AvatarURL: url})
}

// UpdateBanner godoc
// @Summary      バナー画像更新
// @Description  バナー画像をFirebase Storageにアップロードし、URLをDBに保存する
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Param        banner  formData  file             true  "バナー画像（JPEG/PNG/WebP、最大5MB）"
// @Success      200     {object}  dto.BannerResponse  "アップロード成功"
// @Failure      400     {object}  dto.ErrorResponse   "ファイル形式またはサイズが不正"
// @Failure      401     {object}  dto.ErrorResponse   "未認証"
// @Failure      500     {object}  dto.ErrorResponse   "サーバーエラー"
// @Router       /users/me/banner [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateBanner(c echo.Context) error {
	url, err := h.uploadUserImage(c, "banner", "banners")
	if err != nil {
		return h.respondUploadError(c, err)
	}

	userID := c.Get("userID").(uuid.UUID)
	user, err := h.userRepo.FindByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ユーザー情報の取得に失敗しました"})
	}
	user.BannerURL = &url
	if err := h.userRepo.Update(c.Request().Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "保存に失敗しました"})
	}

	return c.JSON(http.StatusOK, dto.BannerResponse{BannerURL: url})
}

// uploadUserImage はマルチパートフォームからファイルを取り出しFirebase Storageにアップロードする
func (h *UserHandler) uploadUserImage(c echo.Context, formField, folder string) (string, error) {
	if _, ok := c.Get("userID").(uuid.UUID); !ok {
		return "", errUnauthorized
	}

	fh, err := c.FormFile(formField)
	if err != nil {
		return "", errMissingFile
	}

	return h.storageSvc.UploadImage(c.Request().Context(), folder, fh)
}

// respondUploadError はアップロード関連のエラーをHTTPレスポンスに変換する
func (h *UserHandler) respondUploadError(c echo.Context, err error) error {
	c.Logger().Errorf("upload error detail: %v", err)
	switch {
	case errors.Is(err, errUnauthorized):
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	case errors.Is(err, errMissingFile):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "ファイルが指定されていません"})
	case errors.Is(err, service.ErrFileTooLarge):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "ファイルサイズは5MB以内にしてください"})
	case errors.Is(err, service.ErrUnsupportedFileType):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "JPEG/PNG/WebP形式のみ対応しています"})
	default:
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "アップロードに失敗しました"})
	}
}

// UpdateTheme godoc
// @Summary      テーマ設定更新
// @Description  ユーザーのテーマ設定（light/dark）を保存する
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     dto.UpdateThemeRequest  true  "テーマ設定"
// @Success      200   {object} dto.SuccessResponse      "更新成功"
// @Failure      400   {object} dto.ErrorResponse        "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse        "未認証"
// @Failure      500   {object} dto.ErrorResponse        "サーバーエラー"
// @Router       /users/me/theme [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateTheme(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	var req dto.UpdateThemeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	if err := h.userSvc.UpdateTheme(c.Request().Context(), userID, req.Theme); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "更新に失敗しました"})
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Message: "success"})
}

// ChangeEmail godoc
// @Summary      メールアドレス変更
// @Description  現在のパスワードを確認した上でメールアドレスを変更する
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     dto.ChangeEmailRequest  true  "変更内容"
// @Success      200   {object} dto.SuccessResponse      "変更成功"
// @Failure      400   {object} dto.ErrorResponse        "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse        "未認証またはパスワード不一致"
// @Failure      409   {object} dto.ErrorResponse        "メールアドレスが既に使用されています"
// @Failure      500   {object} dto.ErrorResponse        "サーバーエラー"
// @Router       /users/me/email [put]
// @Security     BearerAuth
func (h *UserHandler) ChangeEmail(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	var req dto.ChangeEmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	if err := h.userSvc.ChangeEmail(c.Request().Context(), userID, &req); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPassword):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_CREDENTIALS", Message: "パスワードが正しくありません"})
		case errors.Is(err, service.ErrNoPassword):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "NO_PASSWORD", Message: "パスワード未設定のアカウントです"})
		case errors.Is(err, service.ErrEmailTaken):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "EMAIL_TAKEN", Message: "このメールアドレスは既に使用されています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "変更に失敗しました"})
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Message: "success"})
}

// ChangePassword godoc
// @Summary      パスワード変更
// @Description  現在のパスワードを確認した上で新しいパスワードに変更する
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body     dto.ChangePasswordRequest  true  "変更内容"
// @Success      200   {object} dto.SuccessResponse         "変更成功"
// @Failure      400   {object} dto.ErrorResponse           "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse           "未認証またはパスワード不一致"
// @Failure      500   {object} dto.ErrorResponse           "サーバーエラー"
// @Router       /users/me/password [put]
// @Security     BearerAuth
func (h *UserHandler) ChangePassword(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	var req dto.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	if err := h.userSvc.ChangePassword(c.Request().Context(), userID, &req); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPassword):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_CREDENTIALS", Message: "パスワードが正しくありません"})
		case errors.Is(err, service.ErrNoPassword):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "NO_PASSWORD", Message: "パスワード未設定のアカウントです"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "変更に失敗しました"})
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Message: "success"})
}

// GetProfile godoc
// @Summary      ユーザープロフィール取得
// @Description  指定ハンドル（@なし）のユーザープロフィールを返す。ログイン済みの場合はis_followingも返す
// @Tags         users
// @Produce      json
// @Param        handle  path     string           true  "ユーザーハンドル（@なし）" example(john_doe)
// @Success      200     {object} dto.UserResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse "ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse "サーバーエラー"
// @Router       /users/{handle} [get]
func (h *UserHandler) GetProfile(c echo.Context) error {
	handle := c.Param("handle")

	resp, err := h.userSvc.GetProfile(c.Request().Context(), handle, optionalUserID(c))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// GetUserPosts godoc
// @Summary      ユーザーの投稿一覧取得
// @Description  指定ユーザーの投稿（返信を除く）をカーソルページネーションで取得する（削除済み・停止ユーザーは除外）
// @Tags         users
// @Produce      json
// @Param        handle  path   string  true   "ユーザーハンドル（@なし）" example(john_doe)
// @Param        cursor  query  string  false  "ページネーションカーソル（前回レスポンスのnext_cursor）"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /users/{handle}/posts [get]
func (h *UserHandler) GetUserPosts(c echo.Context) error {
	handle := c.Param("handle")
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.postSvc.GetUserPosts(c.Request().Context(), handle, optionalUserID(c), cursor, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// GetUserReplies godoc
// @Summary      ユーザーの返信一覧取得
// @Description  指定ユーザーが他の投稿に行った返信をカーソルページネーションで取得する（削除済み・停止ユーザーは除外）
// @Tags         users
// @Produce      json
// @Param        handle  path   string  true   "ユーザーハンドル（@なし）" example(john_doe)
// @Param        cursor  query  string  false  "ページネーションカーソル（前回レスポンスのnext_cursor）"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /users/{handle}/replies [get]
func (h *UserHandler) GetUserReplies(c echo.Context) error {
	handle := c.Param("handle")
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.postSvc.GetUserReplies(c.Request().Context(), handle, optionalUserID(c), cursor, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// GetFollowers godoc
// @Summary      フォロワー一覧取得
// @Description  指定ユーザーのフォロワー一覧をカーソルページネーションで取得する
// @Tags         users
// @Produce      json
// @Param        handle  path   string  true   "ユーザーハンドル（@なし）" example(john_doe)
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.UserListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /users/{handle}/followers [get]
func (h *UserHandler) GetFollowers(c echo.Context) error {
	handle := c.Param("handle")
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.userSvc.GetFollowers(c.Request().Context(), handle, optionalUserID(c), cursor, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// GetFollowing godoc
// @Summary      フォロー中一覧取得
// @Description  指定ユーザーのフォロー中ユーザー一覧をカーソルページネーションで取得する
// @Tags         users
// @Produce      json
// @Param        handle  path   string  true   "ユーザーハンドル（@なし）" example(john_doe)
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.UserListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /users/{handle}/following [get]
func (h *UserHandler) GetFollowing(c echo.Context) error {
	handle := c.Param("handle")
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.userSvc.GetFollowing(c.Request().Context(), handle, optionalUserID(c), cursor, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// Follow godoc
// @Summary      ユーザーをフォロー
// @Description  指定ユーザーをフォローする。自分自身のフォローは不可
// @Tags         users
// @Produce      json
// @Param        handle  path  string  true  "フォロー対象のハンドル（@なし）" example(john_doe)
// @Success      200     {object} dto.SuccessResponse  "フォロー成功"
// @Failure      400     {object} dto.ErrorResponse    "自分自身をフォローしようとした"
// @Failure      401     {object} dto.ErrorResponse    "未認証"
// @Failure      404     {object} dto.ErrorResponse    "対象ユーザーが見つからない"
// @Failure      409     {object} dto.ErrorResponse    "既にフォロー済み"
// @Failure      500     {object} dto.ErrorResponse    "サーバーエラー"
// @Router       /users/{handle}/follow [post]
// @Security     BearerAuth
func (h *UserHandler) Follow(c echo.Context) error {
	followerID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	handle := c.Param("handle")

	err := h.userSvc.Follow(c.Request().Context(), followerID, handle)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		case errors.Is(err, service.ErrSelfFollow):
			return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "SELF_FOLLOW", Message: "自分自身をフォローすることはできません"})
		case errors.Is(err, service.ErrAlreadyFollowed):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "ALREADY_FOLLOWED", Message: "既にフォローしています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "フォローに失敗しました"})
		}
	}

	return c.JSON(http.StatusOK, dto.SuccessResponse{Message: "success"})
}

// Unfollow godoc
// @Summary      フォロー解除
// @Description  指定ユーザーのフォローを解除する
// @Tags         users
// @Produce      json
// @Param        handle  path  string  true  "フォロー解除対象のハンドル（@なし）" example(john_doe)
// @Success      204     "フォロー解除成功"
// @Failure      401     {object} dto.ErrorResponse  "未認証"
// @Failure      404     {object} dto.ErrorResponse  "対象ユーザーが見つからない"
// @Failure      500     {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /users/{handle}/follow [delete]
// @Security     BearerAuth
func (h *UserHandler) Unfollow(c echo.Context) error {
	followerID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	handle := c.Param("handle")

	err := h.userSvc.Unfollow(c.Request().Context(), followerID, handle)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound), errors.Is(err, service.ErrNotFollowing):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "フォロー解除に失敗しました"})
		}
	}

	return c.NoContent(http.StatusNoContent)
}
