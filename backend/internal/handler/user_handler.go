package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/labstack/echo/v4"
)

// UserHandler はユーザー・フォロー関連のHTTPハンドラー
type UserHandler struct{}

// NewUserHandler はUserHandlerを生成する
func NewUserHandler() *UserHandler {
	return &UserHandler{}
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// GetUserPosts godoc
// @Summary      ユーザーの投稿一覧取得
// @Description  指定ユーザーの投稿をカーソルページネーションで取得する（削除済み・停止ユーザーは除外）
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}
