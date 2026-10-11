package middleware

import (
	"fmt"
	"net/http"

	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/labstack/echo/v4"
)

// Recover は panic を回復し、panic した地点のスタックトレース付きのエラーに変換して返す。
// ログ出力と 500 レスポンスは共通エラーハンドラー（httperr.Handler）が行う。
// AccessLog の後に置くこと（panic したリクエストも 500 としてアクセスログに残すため）。
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				// クライアント切断などで意図的に中断されたときは net/http に任せる
				if r == http.ErrAbortHandler {
					panic(r)
				}
				cause, ok := r.(error)
				if !ok {
					cause = fmt.Errorf("%v", r)
				}
				// 遅延関数の中で取るスタックには panic した関数のフレームが残っている
				err = logger.WithStack(fmt.Errorf("panic recovered: %w", cause), 0)
			}()
			return next(c)
		}
	}
}
