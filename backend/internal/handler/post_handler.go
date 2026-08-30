package handler

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// PostHandler は投稿関連のHTTPハンドラー
type PostHandler struct {
	postSvc    service.PostService
	commentSvc service.CommentService
	validate   *validator.Validate
}

// NewPostHandler はPostHandlerを生成する
func NewPostHandler(postSvc service.PostService, commentSvc service.CommentService) *PostHandler {
	return &PostHandler{postSvc: postSvc, commentSvc: commentSvc, validate: validator.New()}
}

// GetExplore godoc
// @Summary      探索タイムライン取得
// @Description  全ユーザーの公開投稿を新着順（カーソルページネーション）で取得する。停止ユーザー・削除済み投稿は除外
// @Tags         posts
// @Produce      json
// @Param        cursor  query  string  false  "ページネーションカーソル（前回レスポンスのnext_cursor）"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /posts [get]
func (h *PostHandler) GetExplore(c echo.Context) error {
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.postSvc.GetExploreTimeline(c.Request().Context(), optionalUserID(c), cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}
	return c.JSON(http.StatusOK, resp)
}

// GetHome godoc
// @Summary      ホームタイムライン取得
// @Description  全ユーザーの投稿を新着順（カーソルページネーション）で取得する（探索タイムラインと同一の内容、要認証）
// @Tags         posts
// @Produce      json
// @Param        cursor  query  string  false  "ページネーションカーソル（前回レスポンスのnext_cursor）"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      401     {object} dto.ErrorResponse     "未認証"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /posts/home [get]
// @Security     BearerAuth
func (h *PostHandler) GetHome(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.postSvc.GetHomeTimeline(c.Request().Context(), userID, cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}
	return c.JSON(http.StatusOK, resp)
}

// CreatePost godoc
// @Summary      投稿作成
// @Description  テキスト必須（最大280文字）、画像最大4枚、動画最大1本を投稿する。ハッシュタグ・メンションは自動解析される
// @Tags         posts
// @Accept       multipart/form-data
// @Produce      json
// @Param        content  formData  string  true   "投稿本文（最大280文字）"
// @Param        media    formData  file    false  "メディアファイル（画像JPEG/PNG/WebP最大4枚各5MB、動画MP4/MOV最大1本100MB）"
// @Success      201      {object}  dto.PostResponse   "投稿成功"
// @Failure      400      {object}  dto.ErrorResponse  "バリデーションエラー（文字数超過・ファイル形式不正など）"
// @Failure      401      {object}  dto.ErrorResponse  "未認証"
// @Failure      500      {object}  dto.ErrorResponse  "サーバーエラー"
// @Router       /posts [post]
// @Security     BearerAuth
func (h *PostHandler) CreatePost(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	var req dto.CreatePostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	files, err := extractMediaFiles(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "メディアファイルの読み込みに失敗しました"})
	}

	resp, err := h.postSvc.CreatePost(c.Request().Context(), userID, req.Content, files)
	if err != nil {
		return respondPostMediaError(c, err)
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetPost godoc
// @Summary      投稿詳細取得
// @Description  指定IDの投稿を取得する。ログイン済みの場合はis_liked等の個人情報も含む
// @Tags         posts
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      200  {object} dto.PostResponse   "取得成功"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id} [get]
func (h *PostHandler) GetPost(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	resp, err := h.postSvc.GetPost(c.Request().Context(), id, optionalUserID(c))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdatePost godoc
// @Summary      投稿編集
// @Description  自分の投稿の本文を編集する。編集後はis_editedフラグがtrueになる
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        id    path  string               true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Param        body  body  dto.UpdatePostRequest  true  "編集内容"
// @Success      200   {object} dto.PostResponse   "編集成功"
// @Failure      400   {object} dto.ErrorResponse  "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse  "未認証"
// @Failure      403   {object} dto.ErrorResponse  "自分の投稿以外は編集不可"
// @Failure      404   {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500   {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id} [put]
// @Security     BearerAuth
func (h *PostHandler) UpdatePost(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	var req dto.UpdatePostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	resp, err := h.postSvc.UpdatePost(c.Request().Context(), id, userID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		case errors.Is(err, service.ErrForbidden):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{Code: "FORBIDDEN", Message: "自分の投稿のみ編集できます"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "更新に失敗しました"})
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// DeletePost godoc
// @Summary      投稿削除
// @Description  自分の投稿を論理削除する（is_deleted=true）
// @Tags         posts
// @Produce      json
// @Param        id  path  string  true  "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      204  "削除成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      403  {object} dto.ErrorResponse  "自分の投稿以外は削除不可"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id} [delete]
// @Security     BearerAuth
func (h *PostHandler) DeletePost(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	if err := h.postSvc.DeletePost(c.Request().Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		case errors.Is(err, service.ErrForbidden):
			return c.JSON(http.StatusForbidden, dto.ErrorResponse{Code: "FORBIDDEN", Message: "自分の投稿のみ削除できます"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "削除に失敗しました"})
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetComments godoc
// @Summary      コメント（返信）一覧取得
// @Description  指定投稿へのコメントをカーソルページネーションで取得する
// @Tags         posts
// @Produce      json
// @Param        id      path   string  true   "投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.PostListResponse  "取得成功"
// @Failure      404     {object} dto.ErrorResponse     "投稿が見つからない"
// @Failure      500     {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /posts/{id}/comments [get]
func (h *PostHandler) GetComments(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.postSvc.GetComments(c.Request().Context(), id, optionalUserID(c), cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}

// CreateComment godoc
// @Summary      コメント（返信）投稿
// @Description  指定投稿にコメントを投稿する。コメントはpostsテーブルにreply_to付きで保存され、投稿者に通知が届く
// @Tags         posts
// @Accept       multipart/form-data
// @Produce      json
// @Param        id       path      string  true   "コメント先の投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Param        content  formData  string  true   "コメント本文（最大280文字）"
// @Param        media    formData  file    false  "メディアファイル（画像最大4枚 or 動画1本）"
// @Success      201      {object}  dto.PostResponse   "コメント投稿成功"
// @Failure      400      {object}  dto.ErrorResponse  "バリデーションエラー"
// @Failure      401      {object}  dto.ErrorResponse  "未認証"
// @Failure      404      {object}  dto.ErrorResponse  "コメント先の投稿が見つからない"
// @Failure      500      {object}  dto.ErrorResponse  "サーバーエラー"
// @Router       /posts/{id}/comments [post]
// @Security     BearerAuth
func (h *PostHandler) CreateComment(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	var req dto.CreatePostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	files, err := extractMediaFiles(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "メディアファイルの読み込みに失敗しました"})
	}

	resp, err := h.commentSvc.CreateComment(c.Request().Context(), userID, postID, req.Content, files)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return respondPostMediaError(c, err)
	}

	return c.JSON(http.StatusCreated, resp)
}

// extractMediaFiles はmultipartフォームの"media"フィールドからファイル一覧を取得する
func extractMediaFiles(c echo.Context) ([]*multipart.FileHeader, error) {
	form, err := c.MultipartForm()
	if err != nil {
		// フォームがmultipartでない（JSONリクエストなど）場合はメディアなしとして扱う
		return nil, nil
	}
	return form.File["media"], nil
}

// respondPostMediaError は投稿作成時のメディア関連エラーをHTTPレスポンスに変換する
func respondPostMediaError(c echo.Context, err error) error {
	c.Logger().Errorf("post media error detail: %v", err)
	switch {
	case errors.Is(err, service.ErrTooManyMedia):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "メディアファイルの枚数上限を超えています"})
	case errors.Is(err, service.ErrMixedMediaTypes):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "動画は1本のみ、画像と同時投稿はできません"})
	case errors.Is(err, service.ErrUnsupportedFileType):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "対応していないファイル形式です"})
	case errors.Is(err, service.ErrFileTooLarge):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "ファイルサイズが上限を超えています"})
	default:
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "投稿に失敗しました"})
	}
}
