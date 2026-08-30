package handler

import (
	"errors"
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// InteractionHandler はいいね・リポスト・ブックマーク関連のHTTPハンドラー
type InteractionHandler struct {
	likeSvc     service.LikeService
	repostSvc   service.RepostService
	bookmarkSvc service.BookmarkService
}

// NewInteractionHandler はInteractionHandlerを生成する
func NewInteractionHandler(likeSvc service.LikeService, repostSvc service.RepostService, bookmarkSvc service.BookmarkService) *InteractionHandler {
	return &InteractionHandler{likeSvc: likeSvc, repostSvc: repostSvc, bookmarkSvc: bookmarkSvc}
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	resp, err := h.likeSvc.Like(c.Request().Context(), userID, postID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		case errors.Is(err, service.ErrAlreadyLiked):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "ALREADY_LIKED", Message: "既にいいねしています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "いいねに失敗しました"})
		}
	}

	return c.JSON(http.StatusOK, resp)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	resp, err := h.likeSvc.Unlike(c.Request().Context(), userID, postID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "いいね取消に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// Repost godoc
// @Summary      リポスト
// @Description  指定投稿をリポストする。postsテーブルにrepost_of付きで保存され、投稿者に通知が届く
// @Tags         interactions
// @Accept       multipart/form-data
// @Produce      json
// @Param        id       path      string  true   "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Param        content  formData  string  false  "引用コメント（最大280文字）"
// @Success      200      {object}  dto.RepostResponse  "リポスト成功"
// @Failure      401      {object}  dto.ErrorResponse   "未認証"
// @Failure      404      {object}  dto.ErrorResponse   "投稿が見つからない"
// @Failure      409      {object}  dto.ErrorResponse   "既にリポスト済み"
// @Failure      500      {object}  dto.ErrorResponse   "サーバーエラー"
// @Router       /posts/{id}/repost [post]
// @Security     BearerAuth
func (h *InteractionHandler) Repost(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	content := c.FormValue("content")
	files, err := extractMediaFiles(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "メディアファイルの読み込みに失敗しました"})
	}

	resp, err := h.repostSvc.Repost(c.Request().Context(), userID, postID, content, files)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		case errors.Is(err, service.ErrAlreadyReposted):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "ALREADY_REPOSTED", Message: "既にリポストしています"})
		default:
			return respondPostMediaError(c, err)
		}
	}

	return c.JSON(http.StatusOK, resp)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	resp, err := h.repostSvc.Unrepost(c.Request().Context(), userID, postID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "リポスト取消に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	err = h.bookmarkSvc.Bookmark(c.Request().Context(), userID, postID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		case errors.Is(err, service.ErrAlreadyBookmarked):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "ALREADY_BOOKMARKED", Message: "既にブックマークしています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ブックマークに失敗しました"})
		}
	}

	return c.NoContent(http.StatusNoContent)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	if err := h.bookmarkSvc.Unbookmark(c.Request().Context(), userID, postID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "ブックマーク解除に失敗しました"})
	}

	return c.NoContent(http.StatusNoContent)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.bookmarkSvc.GetBookmarks(c.Request().Context(), userID, cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}
