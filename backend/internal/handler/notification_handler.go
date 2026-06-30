package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/labstack/echo/v4"
)

// NotificationHandler は通知関連のHTTPハンドラー
type NotificationHandler struct{}

// NewNotificationHandler はNotificationHandlerを生成する
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// GetNotifications godoc
// @Summary      通知一覧取得
// @Description  自分宛の通知をカーソルページネーションで取得する。未読数も返す
// @Tags         notifications
// @Produce      json
// @Param        cursor  query  string  false  "ページネーションカーソル"
// @Param        limit   query  int     false  "取得件数（デフォルト20、最大50）" minimum(1) maximum(50)
// @Success      200     {object} dto.NotificationListResponse  "取得成功"
// @Failure      401     {object} dto.ErrorResponse              "未認証"
// @Failure      500     {object} dto.ErrorResponse              "サーバーエラー"
// @Router       /notifications [get]
// @Security     BearerAuth
func (h *NotificationHandler) GetNotifications(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}

// MarkAllRead godoc
// @Summary      全通知を既読にする
// @Description  自分の未読通知を全て既読（is_read=true）にする
// @Tags         notifications
// @Produce      json
// @Success      204  "既読処理成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /notifications/read [put]
// @Security     BearerAuth
func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, dto.ErrorResponse{Code: "NOT_IMPLEMENTED", Message: "実装予定"})
}
