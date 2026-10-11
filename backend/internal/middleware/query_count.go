package middleware

import (
	"log/slog"
	"strconv"

	"github.com/atyahara/sns-backend/internal/querycount"
	"github.com/labstack/echo/v4"
)

// HeaderDBQueryCount はリクエスト処理中に発行されたSQLクエリ数を返すレスポンスヘッダー
const HeaderDBQueryCount = "X-DB-Query-Count"

// QueryCount はリクエストごとにSQLクエリ数を数え、レスポンスヘッダーとログに出力する。
// N+1検出・性能計測用。DB構成が分かる情報を返すため、開発・テスト環境でのみ有効にすること。
// カウント自体はDB側で querycount.Register が登録されている必要がある。
func QueryCount() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := querycount.WithCounter(req.Context())
			c.SetRequest(req.WithContext(ctx))

			// ヘッダーはボディ書き込み前にしか付けられないため、書き込み直前のフックで設定する
			c.Response().Before(func() {
				c.Response().Header().Set(HeaderDBQueryCount, strconv.FormatInt(querycount.Count(ctx), 10))
			})

			err := next(c)

			slog.InfoContext(ctx, "db query count",
				"method", req.Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"queries", querycount.Count(ctx),
			)
			return err
		}
	}
}
