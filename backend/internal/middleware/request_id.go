package middleware

import (
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequestID はリクエストごとにユニークなID（UUID v4）を発行し、レスポンスヘッダー X-Request-ID で返す。
// リクエストのコンテキストに request_id 付きのロガーを入れるため、以降の logger.FromContext(ctx) のログには
// 自動でリクエストIDが含まれる。
//
// クライアントが送ってきた X-Request-ID は使わない（偽装・重複したIDでログの突き合わせが壊れるのを防ぐ）。
// 全ミドルウェアの先頭に置くこと。
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := uuid.NewString()
			req := c.Request()
			c.SetRequest(req.WithContext(logger.WithRequestID(req.Context(), id)))
			c.Response().Header().Set(echo.HeaderXRequestID, id)
			return next(c)
		}
	}
}
