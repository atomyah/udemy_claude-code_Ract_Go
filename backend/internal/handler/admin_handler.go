package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/labstack/echo/v4"
)

// AdminHandler は管理者専用のHTTPハンドラー（roleがadminのユーザーのみアクセス可）
type AdminHandler struct{}

// NewAdminHandler はAdminHandlerを生成する
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
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
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}
