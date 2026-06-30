package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/labstack/echo/v4"
)

// SearchHandler は検索関連のHTTPハンドラー
type SearchHandler struct{}

// NewSearchHandler はSearchHandlerを生成する
func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

// SearchUsers godoc
// @Summary      ユーザー検索
// @Description  表示名・ハンドルで部分一致検索する（停止ユーザーは除外）
// @Tags         search
// @Produce      json
// @Param        q       query  string  true   "検索クエリ（2文字以上）"
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.UserListResponse  "検索結果"
// @Failure      400     {object} dto.ErrorResponse     "クエリが短すぎる"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /search/users [get]
func (h *SearchHandler) SearchUsers(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// SearchPosts godoc
// @Summary      投稿内容検索
// @Description  投稿のテキスト本文でキーワード検索する（削除済み・停止ユーザーの投稿は除外）
// @Tags         search
// @Produce      json
// @Param        q       query  string  true   "検索クエリ"
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "検索結果"
// @Failure      400     {object} dto.ErrorResponse     "クエリが必要"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /search/posts [get]
func (h *SearchHandler) SearchPosts(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// GetHashtagPosts godoc
// @Summary      ハッシュタグ投稿一覧取得
// @Description  指定ハッシュタグ（#なし）が付いた投稿をカーソルページネーションで取得する
// @Tags         search
// @Produce      json
// @Param        tag     path   string  true   "ハッシュタグ（#なし）" example(golang)
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "ハッシュタグが存在しない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /search/hashtags/{tag} [get]
func (h *SearchHandler) GetHashtagPosts(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}
