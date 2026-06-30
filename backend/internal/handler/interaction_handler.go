package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/labstack/echo/v4"
)

// InteractionHandler はいいね・リポスト・ブックマーク関連のHTTPハンドラー
type InteractionHandler struct{}

// NewInteractionHandler はInteractionHandlerを生成する
func NewInteractionHandler() *InteractionHandler {
	return &InteractionHandler{}
}

// Like godoc
// @Summary      いいね
// @Description  指定投稿にいいねする。自分の投稿以外の場合は投稿者に通知が届く
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      200  {object} dto.LikeResponse   "いいね成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      409  {object} dto.ErrorResponse  "既にいいね済み"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id}/like [post]
// @Security     BearerAuth
func (h *InteractionHandler) Like(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// Unlike godoc
// @Summary      いいね取消
// @Description  指定投稿のいいねを取り消す
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      200  {object} dto.LikeResponse   "いいね取消成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id}/like [delete]
// @Security     BearerAuth
func (h *InteractionHandler) Unlike(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// Repost godoc
// @Summary      リポスト
// @Description  指定投稿をリポストする。postsテーブルにrepost_of付きで保存され、投稿者に通知が届く
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      200  {object} dto.RepostResponse  "リポスト成功"
// @Failure      401  {object} dto.ErrorResponse   "未認証"
// @Failure      404  {object} dto.ErrorResponse   "投稿が見つからない"
// @Failure      409  {object} dto.ErrorResponse   "既にリポスト済み"
// @Failure      500  {object} dto.ErrorResponse   "サーバーエラー"
// @Router       /posts/{id}/repost [post]
// @Security     BearerAuth
func (h *InteractionHandler) Repost(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// Unrepost godoc
// @Summary      リポスト取消
// @Description  自分のリポストを論理削除する
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "元投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      200  {object} dto.RepostResponse  "リポスト取消成功"
// @Failure      401  {object} dto.ErrorResponse   "未認証"
// @Failure      404  {object} dto.ErrorResponse   "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse   "サーバーエラー"
// @Router       /posts/{id}/repost [delete]
// @Security     BearerAuth
func (h *InteractionHandler) Unrepost(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// Bookmark godoc
// @Summary      ブックマーク追加
// @Description  指定投稿をブックマークに保存する（自分だけに見える）
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      204  "ブックマーク成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      409  {object} dto.ErrorResponse  "既にブックマーク済み"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id}/bookmark [post]
// @Security     BearerAuth
func (h *InteractionHandler) Bookmark(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// Unbookmark godoc
// @Summary      ブックマーク解除
// @Description  指定投稿のブックマークを解除する
// @Tags         interactions
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      204  "ブックマーク解除成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id}/bookmark [delete]
// @Security     BearerAuth
func (h *InteractionHandler) Unbookmark(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// GetBookmarks godoc
// @Summary      ブックマーク一覧取得
// @Description  自分のブックマーク済み投稿をカーソルページネーションで取得する
// @Tags         interactions
// @Produce      json
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      401     {object} dto.ErrorResponse     "未認証"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /bookmarks [get]
// @Security     BearerAuth
func (h *InteractionHandler) GetBookmarks(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}
