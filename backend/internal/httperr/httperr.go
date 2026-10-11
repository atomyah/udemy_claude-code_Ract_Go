// Package httperr はエラーレスポンスの返却とエラーログの記録をまとめる。
//
// サーバーエラー（5xx）では原因エラーとスタックトレースを ERROR ログに残し、
// レスポンスには内部情報を含めずリクエストIDだけを返す（利用者はそのIDを問い合わせに使う）。
package httperr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CodeInternal はサーバーエラーのエラーコード
const CodeInternal = "INTERNAL_ERROR"

// DefaultInternalMessage はサーバーエラーの既定メッセージ
const DefaultInternalMessage = "サーバーエラーが発生しました"

// Internal は原因エラーをスタックトレース付きで ERROR ログに記録し、500 を返す。
// message には利用者向けの日本語（例:「削除に失敗しました」）を渡す。err の内容はレスポンスに含めない。
func Internal(c echo.Context, message string, err error) error {
	stack := logger.StackOf(err)
	if stack == nil {
		// 呼び出し元（ハンドラー）から始まるスタックトレース
		stack = logger.Stack(1)
	}
	logError(c, http.StatusInternalServerError, err, stack)
	return writeInternal(c, http.StatusInternalServerError, message)
}

// InternalMessage は利用者に返すサーバーエラーのメッセージ。リクエストIDがあれば問い合わせ用に添える
func InternalMessage(message, requestID string) string {
	if requestID == "" {
		return message
	}
	return fmt.Sprintf("%s。お問い合わせの際はこのIDをお伝えください: %s", message, requestID)
}

func writeInternal(c echo.Context, status int, message string) error {
	requestID := logger.RequestID(c.Request().Context())
	return c.JSON(status, dto.ErrorResponse{
		Code:      CodeInternal,
		Message:   InternalMessage(message, requestID),
		RequestID: requestID,
	})
}

func logError(c echo.Context, status int, err error, stack []string) {
	req := c.Request()
	event := logger.FromContext(req.Context()).Error().
		Err(err).
		Strs("stack", stack).
		Str("method", req.Method).
		Str("path", req.URL.Path).
		Int("status", status)
	if userID, ok := c.Get("userID").(uuid.UUID); ok {
		event = event.Str("user_id", userID.String())
	}
	event.Msg("request failed")
}

// httpErrorMessages は Echo 標準エラー（ルート未定義・メソッド不一致など）を日本語に変換する表
var httpErrorMessages = map[int]dto.ErrorResponse{
	http.StatusBadRequest:            {Code: "BAD_REQUEST", Message: "リクエストが不正です"},
	http.StatusUnauthorized:          {Code: "UNAUTHORIZED", Message: "ログインが必要です"},
	http.StatusForbidden:             {Code: "FORBIDDEN", Message: "この操作を行う権限がありません"},
	http.StatusNotFound:              {Code: "NOT_FOUND", Message: "指定されたページが見つかりません"},
	http.StatusMethodNotAllowed:      {Code: "METHOD_NOT_ALLOWED", Message: "許可されていない操作です"},
	http.StatusRequestEntityTooLarge: {Code: "PAYLOAD_TOO_LARGE", Message: "リクエストのサイズが大きすぎます"},
	http.StatusUnsupportedMediaType:  {Code: "UNSUPPORTED_MEDIA_TYPE", Message: "対応していない形式のリクエストです"},
	http.StatusTooManyRequests:       {Code: "TOO_MANY_REQUESTS", Message: "リクエストが多すぎます。しばらくしてから再度お試しください"},
}

// Handler は e.HTTPErrorHandler に設定する共通エラーハンドラー。
// ハンドラーが返したエラー・panic・Echo 標準エラーを { code, message } 形式の日本語レスポンスにする。
//   - 4xx の *echo.HTTPError: 上の表で日本語に変換する（ログはアクセスログの WARN で足りるため出さない）
//   - それ以外（5xx・想定外のエラー・panic）: スタックトレース付きで ERROR ログを出し、リクエストIDだけを返す
func Handler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status := http.StatusInternalServerError
	var he *echo.HTTPError
	if errors.As(err, &he) {
		status = he.Code
	}

	var writeErr error
	if status < http.StatusInternalServerError {
		body, ok := httpErrorMessages[status]
		if !ok {
			body = dto.ErrorResponse{Code: "REQUEST_ERROR", Message: "リクエストを処理できませんでした"}
		}
		writeErr = respond(c, status, body)
	} else {
		stack := logger.StackOf(err)
		if stack == nil {
			stack = logger.Stack(1)
		}
		logError(c, status, err, stack)
		requestID := logger.RequestID(c.Request().Context())
		writeErr = respond(c, status, dto.ErrorResponse{
			Code:      CodeInternal,
			Message:   InternalMessage(DefaultInternalMessage, requestID),
			RequestID: requestID,
		})
	}
	if writeErr != nil {
		logger.FromContext(c.Request().Context()).Warn().Err(writeErr).Msg("failed to write error response")
	}
}

func respond(c echo.Context, status int, body dto.ErrorResponse) error {
	if c.Request().Method == http.MethodHead {
		return c.NoContent(status)
	}
	return c.JSON(status, body)
}
