package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/atyahara/sns-backend/internal/querycount"
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
		require.NoError(t, json.Unmarshal([]byte(line), &m), "ログは JSON でなければならない: %s", line)
		lines = append(lines, m)
	}
	return lines
}

// findLog は message が一致する最初のログを返す
func findLog(t *testing.T, lines []map[string]any, message string) map[string]any {
	t.Helper()
	for _, l := range lines {
		if l["message"] == message {
			return l
		}
	}
	t.Fatalf("message=%q のログがありません: %v", message, lines)
	return nil
}

// newLoggingServer は main.go と同じ順序（RequestID → AccessLog → Recover）でミドルウェアを組んだ Echo を返す
func newLoggingServer() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = httperr.Handler
	e.Use(RequestID(), AccessLog(testActions), Recover())
	return e
}

func serve(e *echo.Echo, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestRequestID_SetsUniqueResponseHeader(t *testing.T) {
	captureLogs(t)
	e := newLoggingServer()
	e.GET("/ok", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	first := serve(e, http.MethodGet, "/ok").Header().Get(echo.HeaderXRequestID)
	second := serve(e, http.MethodGet, "/ok").Header().Get(echo.HeaderXRequestID)

	_, err := uuid.Parse(first)
	assert.NoError(t, err, "リクエストIDは UUID")
	assert.NotEqual(t, first, second, "リクエストごとに別のID")
}

func TestRequestID_IgnoresClientSuppliedID(t *testing.T) {
	captureLogs(t)
	e := newLoggingServer()
	e.GET("/ok", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set(echo.HeaderXRequestID, "forged-id")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.NotEqual(t, "forged-id", rec.Header().Get(echo.HeaderXRequestID))
}

func TestRequestID_IsIncludedInLogsDuringRequest(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()
	e.GET("/ok", func(c echo.Context) error {
		// ハンドラー（やサービス）がコンテキストのロガーで出したログ
		logger.FromContext(c.Request().Context()).Info().Msg("in handler")
		return c.NoContent(http.StatusOK)
	})

	rec := serve(e, http.MethodGet, "/ok")

	id := rec.Header().Get(echo.HeaderXRequestID)
	lines := logLines(t, buf)
	assert.Equal(t, id, findLog(t, lines, "in handler")["request_id"])
	assert.Equal(t, id, findLog(t, lines, "access")["request_id"])
}

func TestAccessLog_OutputsRequiredFields(t *testing.T) {
	buf := captureLogs(t)
	userID := uuid.New()
	e := newLoggingServer()
	e.GET("/api/v1/posts/:id", func(c echo.Context) error {
		c.Set("userID", userID) // JWT ミドルウェア通過後の状態
		return c.JSON(http.StatusOK, map[string]string{"ok": "1"})
	})

	rec := serve(e, http.MethodGet, "/api/v1/posts/123?limit=20")

	log := findLog(t, logLines(t, buf), "access")
	assert.Equal(t, "info", log["level"])
	assert.NotEmpty(t, log["time"])
	assert.Equal(t, http.MethodGet, log["method"])
	assert.Equal(t, "/api/v1/posts/123", log["path"], "パスにクエリ文字列は含めない")
	assert.Equal(t, "/api/v1/posts/:id", log["route"])
	assert.Equal(t, float64(200), log["status"])
	assert.Contains(t, log, "latency_ms")
	assert.Equal(t, rec.Header().Get(echo.HeaderXRequestID), log["request_id"])
	assert.Equal(t, userID.String(), log["user_id"])
	assert.NotContains(t, log, "db_queries", "クエリ数カウントが無効なときは出さない")
}

func TestAccessLog_UnauthenticatedRequestHasNoUserID(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()
	e.GET("/ok", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	serve(e, http.MethodGet, "/ok")

	assert.NotContains(t, findLog(t, logLines(t, buf), "access"), "user_id")
}

func TestAccessLog_LevelDependsOnStatus(t *testing.T) {
	cases := []struct {
		name    string
		handler echo.HandlerFunc
		status  int
		level   string
	}{
		{"2xx は INFO", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, 200, "info"},
		{"4xx は WARN", func(c echo.Context) error { return c.JSON(http.StatusBadRequest, nil) }, 400, "warn"},
		{"返されたエラーは 500・ERROR", func(c echo.Context) error { return errors.New("boom") }, 500, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLogs(t)
			e := newLoggingServer()
			e.GET("/x", tc.handler)

			rec := serve(e, http.MethodGet, "/x")

			assert.Equal(t, tc.status, rec.Code)
			log := findLog(t, logLines(t, buf), "access")
			assert.Equal(t, tc.level, log["level"])
			assert.Equal(t, float64(tc.status), log["status"])
		})
	}
}

func TestAccessLog_UnknownRouteIs404InJapanese(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()

	rec := serve(e, http.MethodGet, "/no-such-route")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "NOT_FOUND", decodeError(t, rec)["code"])
	lines := logLines(t, buf)
	require.Len(t, lines, 1, "4xx はアクセスログだけ")
	assert.Equal(t, "warn", lines[0]["level"])
}

func TestAccessLog_IncludesQueryCountWhenEnabled(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()
	e.Use(QueryCount())
	e.GET("/ok", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	serve(e, http.MethodGet, "/ok")

	assert.Equal(t, float64(0), findLog(t, logLines(t, buf), "access")["db_queries"])
	_, ok := querycount.Lookup(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	assert.False(t, ok)
}

func panickingHandler(echo.Context) error {
	panic("nil map write")
}

func TestRecover_PanicIsLoggedWithStackAndReturns500WithRequestID(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()
	e.GET("/panic", panickingHandler)

	rec := serve(e, http.MethodGet, "/panic")

	// レスポンス: 内部情報（panic の内容・スタック）は返さず、リクエストIDだけ
	id := rec.Header().Get(echo.HeaderXRequestID)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "INTERNAL_ERROR", body["code"])
	assert.Equal(t, id, body["request_id"])
	assert.Contains(t, body["message"], "このIDをお伝えください: "+id)
	assert.NotContains(t, rec.Body.String(), "nil map write")

	// ログ: panic した関数を含むスタックトレース付きの ERROR と、500 のアクセスログ
	lines := logLines(t, buf)
	failed := findLog(t, lines, "request failed")
	assert.Equal(t, "error", failed["level"])
	assert.Equal(t, id, failed["request_id"])
	assert.Equal(t, "panic recovered: nil map write", failed["error"])
	assert.Contains(t, strings.Join(toStrings(failed["stack"]), "\n"), "panickingHandler")

	access := findLog(t, lines, "access")
	assert.Equal(t, float64(500), access["status"])
	assert.Equal(t, "error", access["level"])
}

func toStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, _ := item.(string)
		out = append(out, s)
	}
	return out
}
