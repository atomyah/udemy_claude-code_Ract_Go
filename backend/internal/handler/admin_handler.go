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

// AdminHandler は管理者専用のHTTPハンドラー（roleがadminのユーザーのみアクセス可）
type AdminHandler struct {
	adminSvc service.AdminService
}

// NewAdminHandler はAdminHandlerを生成する
func NewAdminHandler(adminSvc service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

// AdminDeletePost godoc
// @Summary      投稿の強制削除（管理者専用）
// @Description  管理者が任意のユーザーの投稿を強制削除する（論理削除）
// @Tags         admin
// @Produce      json
// @Param        id  path  string  true  "削除対象の投稿ID（UUID）" example(550e8400-e29b-41d4-a716-446655440003)
// @Success      204  "削除成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      403  {object} dto.ErrorResponse  "管理者権限が必要"
// @Failure      404  {object} dto.ErrorResponse  "投稿が見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /admin/posts/{id} [delete]
// @Security     BearerAuth
func (h *AdminHandler) AdminDeletePost(c echo.Context) error {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
	}

	if err := h.adminSvc.ForceDeletePost(c.Request().Context(), postID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "POST_NOT_FOUND", Message: "投稿が見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "削除に失敗しました"})
	}

	return c.NoContent(http.StatusNoContent)
}

// SuspendUser godoc
// @Summary      ユーザーアカウント停止（管理者専用）
// @Description  管理者が指定ユーザーのアカウントを停止する（is_suspended=true）。停止後はログイン不可
// @Tags         admin
// @Produce      json
// @Param        id  path  string  true  "停止対象のユーザーID（UUID）" example(550e8400-e29b-41d4-a716-446655440000)
// @Success      204  "停止成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      403  {object} dto.ErrorResponse  "管理者権限が必要"
// @Failure      404  {object} dto.ErrorResponse  "ユーザーが見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /admin/users/{id}/suspend [put]
// @Security     BearerAuth
func (h *AdminHandler) SuspendUser(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
	}

	if err := h.adminSvc.SuspendUser(c.Request().Context(), userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "停止処理に失敗しました"})
	}

	return c.NoContent(http.StatusNoContent)
}

// UnsuspendUser godoc
// @Summary      ユーザーアカウント停止解除（管理者専用）
// @Description  管理者が停止中のユーザーアカウントを復活させる（is_suspended=false）
// @Tags         admin
// @Produce      json
// @Param        id  path  string  true  "停止解除対象のユーザーID（UUID）" example(550e8400-e29b-41d4-a716-446655440000)
// @Success      204  "停止解除成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      403  {object} dto.ErrorResponse  "管理者権限が必要"
// @Failure      404  {object} dto.ErrorResponse  "ユーザーが見つからない"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /admin/users/{id}/suspend [delete]
// @Security     BearerAuth
func (h *AdminHandler) UnsuspendUser(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
	}

	if err := h.adminSvc.UnsuspendUser(c.Request().Context(), userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "ユーザーが見つかりません"})
		}
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "停止解除に失敗しました"})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListUsers godoc
// @Summary      ユーザー一覧取得（管理者専用）
// @Description  全ユーザーをカーソルページネーションで取得する（メールアドレス・停止状態を含む）
// @Tags         admin
// @Produce      json
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.AdminUserListResponse  "取得成功"
// @Failure      401     {object} dto.ErrorResponse          "未認証"
// @Failure      403     {object} dto.ErrorResponse          "管理者権限が必要"
// @Failure      500     {object} dto.ErrorResponse          "サーバーエラー"
// @Router       /admin/users [get]
// @Security     BearerAuth
func (h *AdminHandler) ListUsers(c echo.Context) error {
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.adminSvc.ListUsers(c.Request().Context(), cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
}
