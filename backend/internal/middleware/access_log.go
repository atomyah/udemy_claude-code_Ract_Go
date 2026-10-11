package middleware

import (
	"time"

	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/atyahara/sns-backend/internal/querycount"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// SlowRequestThreshold を超えたリクエストは WARN で記録する（非機能要件: 一覧取得 API は 500ms 以内）
const SlowRequestThreshold = 500 * time.Millisecond

// AccessLog は全リクエストのアクセスログを 1 行の JSON で出力する。
// 出力項目: time / request_id / action（操作名）/ description（操作内容の日本語説明）/ method / path / route /
// status / latency_ms / user_id（認証済みのとき）/ remote_ip / user_agent / bytes_out / db_queries（クエリ数カウント有効時）。
//
// action は actions（Swagger の @Summary から作る表）で引く。例:
//
//	"action": "いいね",
//	"description": "ログインユーザーが「いいね」（id=8a16...）をリクエストし、成功しました（201）"
//
// レベル: 5xx=ERROR、4xx=WARN、SlowRequestThreshold 超=WARN（slow=true）、それ以外=INFO。
// RequestID の直後に置くこと（request_id 付きロガーを使うため）。
func AccessLog(actions ActionCatalog) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			if err := next(c); err != nil {
				// エラーハンドラーでレスポンスを確定させてから、最終的なステータスを記録する
				c.Error(err)
			}
			latency := time.Since(start)

			req := c.Request()
			res := c.Response()
			log := logger.FromContext(req.Context())

			var event *zerolog.Event
			switch {
			case res.Status >= 500:
				event = log.Error()
			case res.Status >= 400:
				event = log.Warn()
			case latency > SlowRequestThreshold:
				event = log.Warn().Bool("slow", true)
			default:
				event = log.Info()
			}

			userID, authenticated := c.Get("userID").(uuid.UUID)
			action := actions.Action(req.Method, c.Path(), res.Status)

			event = event.
				Str("action", action).
				Str("description", describeRequest(c, action, res.Status, authenticated)).
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Str("route", c.Path()).
				Int("status", res.Status).
				Float64("latency_ms", float64(latency.Microseconds())/1000).
				Str("remote_ip", c.RealIP()).
				Str("user_agent", req.UserAgent()).
				Int64("bytes_out", res.Size)
			if authenticated {
				event = event.Str("user_id", userID.String())
			}
			if queries, ok := querycount.Lookup(req.Context()); ok {
				event = event.Int64("db_queries", queries)
			}
			event.Msg("access")
			return nil
		}
	}
}
