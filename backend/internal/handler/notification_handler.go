package handler

import (
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// NotificationHandler は通知関連のHTTPハンドラー
type NotificationHandler struct {
	notificationSvc service.NotificationService
}

// NewNotificationHandler はNotificationHandlerを生成する
func NewNotificationHandler(notificationSvc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationSvc: notificationSvc}
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}
	cursor := c.QueryParam("cursor")
	limit := utils.ParseLimit(c.QueryParam("limit"), defaultListLimit, maxListLimit)

	resp, err := h.notificationSvc.GetNotifications(c.Request().Context(), userID, cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "取得に失敗しました"})
	}

	return c.JSON(http.StatusOK, resp)
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
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	if err := h.notificationSvc.MarkAllRead(c.Request().Context(), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "既読処理に失敗しました"})
	}

	return c.NoContent(http.StatusNoContent)
}
