package httperr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	logger.SetOutput(buf, zerolog.DebugLevel)
	t.Cleanup(func() { logger.SetOutput(os.Stdout, zerolog.InfoLevel) })
	return buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		lines = append(lines, m)
	}
	return lines
}

// newContext はリクエストID付きのコンテキスト（RequestID ミドルウェア通過後の状態）を作る
func newContext(method, requestID string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, "/api/v1/posts", nil)
	if requestID != "" {
		req = req.WithContext(logger.WithRequestID(req.Context(), requestID))
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) dto.ErrorResponse {
	t.Helper()
	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestInternal_LogsErrorWithStackAndReturnsOnlyRequestID(t *testing.T) {
	buf := captureLogs(t)
	c, rec := newContext(http.MethodGet, "req-abc")
	userID := uuid.New()
	c.Set("userID", userID)

	require.NoError(t, Internal(c, "取得に失敗しました", errors.New(`pq: relation "posts" does not exist`)))

	// レスポンス: 内部情報を含まず、リクエストIDと問い合わせの案内だけ
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decode(t, rec)
	assert.Equal(t, "INTERNAL_ERROR", body.Code)
	assert.Equal(t, "req-abc", body.RequestID)
	assert.Equal(t, "取得に失敗しました。お問い合わせの際はこのIDをお伝えください: req-abc", body.Message)
	assert.NotContains(t, rec.Body.String(), "relation")
	assert.NotContains(t, rec.Body.String(), "stack")

	// ログ: ERROR・原因エラー・スタックトレース・リクエストID・ユーザーID
	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	log := lines[0]
	assert.Equal(t, "error", log["level"])
	assert.Equal(t, `pq: relation "posts" does not exist`, log["error"])
	assert.Equal(t, "req-abc", log["request_id"])
	assert.Equal(t, userID.String(), log["user_id"])
	assert.Equal(t, float64(500), log["status"])
	stack, ok := log["stack"].([]any)
	require.True(t, ok, "stack は配列")
	require.NotEmpty(t, stack)
	assert.Contains(t, stack[0], "TestInternal_LogsErrorWithStackAndReturnsOnlyRequestID", "スタックは呼び出し元から始まる")
}

func TestInternal_WithoutRequestID_ReturnsPlainMessage(t *testing.T) {
	captureLogs(t)
	c, rec := newContext(http.MethodGet, "")

	require.NoError(t, Internal(c, DefaultInternalMessage, errors.New("db down")))

	body := decode(t, rec)
	assert.Equal(t, DefaultInternalMessage, body.Message)
	assert.Empty(t, body.RequestID)
	assert.NotContains(t, rec.Body.String(), "request_id", "空のリクエストIDは返さない")
}

func TestHandler_ConvertsEchoHTTPErrorsToJapanese(t *testing.T) {
	cases := []struct {
		status int
		code   string
	}{
		{http.StatusNotFound, "NOT_FOUND"},
		{http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE"},
		{http.StatusTooManyRequests, "TOO_MANY_REQUESTS"},
		{http.StatusTeapot, "REQUEST_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			buf := captureLogs(t)
			c, rec := newContext(http.MethodGet, "req-4xx")

			Handler(echo.NewHTTPError(tc.status, "internal detail"), c)

			assert.Equal(t, tc.status, rec.Code)
			body := decode(t, rec)
			assert.Equal(t, tc.code, body.Code)
			assert.NotEmpty(t, body.Message)
			assert.NotContains(t, body.Message, "internal detail", "Echo のメッセージはそのまま返さない")
			assert.Empty(t, body.RequestID, "4xx ではリクエストIDを返さない")
			assert.Empty(t, buf.String(), "4xx はエラーログを出さない（アクセスログの WARN で記録される）")
		})
	}
}

func TestHandler_UnknownErrorIsLoggedAndHidden(t *testing.T) {
	buf := captureLogs(t)
	c, rec := newContext(http.MethodGet, "req-500")

	Handler(fmt.Errorf("wrap: %w", errors.New("secret connection string")), c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decode(t, rec)
	assert.Equal(t, "INTERNAL_ERROR", body.Code)
	assert.Equal(t, "req-500", body.RequestID)
	assert.Contains(t, body.Message, "このIDをお伝えください")
	assert.NotContains(t, rec.Body.String(), "secret")

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "error", lines[0]["level"])
	assert.Equal(t, "wrap: secret connection string", lines[0]["error"])
	assert.NotEmpty(t, lines[0]["stack"])
}

func TestHandler_UsesStackAttachedToError(t *testing.T) {
	buf := captureLogs(t)
	c, _ := newContext(http.MethodGet, "req-stack")

	err := logger.WithStack(errors.New("panic recovered: boom"), 0)
	Handler(err, c)

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	stack := lines[0]["stack"].([]any)
	assert.Contains(t, stack[0], "TestHandler_UsesStackAttachedToError", "エラー発生地点のスタックを使う")
}

func TestHandler_HeadRequestHasNoBody(t *testing.T) {
	captureLogs(t)
	c, rec := newContext(http.MethodHead, "")

	Handler(echo.ErrNotFound, c)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestHandler_CommittedResponseIsNotOverwritten(t *testing.T) {
	buf := captureLogs(t)
	c, rec := newContext(http.MethodGet, "")
	require.NoError(t, c.NoContent(http.StatusNoContent))

	Handler(errors.New("late error"), c)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, buf.String())
}
