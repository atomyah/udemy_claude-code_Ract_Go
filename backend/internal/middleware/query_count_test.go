package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atyahara/sns-backend/internal/querycount"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryCount_SetsHeaderWithCounterInRequestContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := QueryCount()(func(c echo.Context) error {
		// ハンドラーが受け取るコンテキストにカウンターが埋め込まれていること
		ctx := c.Request().Context()
		assert.Equal(t, int64(0), querycount.Count(ctx))
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "0", rec.Header().Get(HeaderDBQueryCount))
}

func TestQueryCount_SetsHeaderOnNoContent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1/like", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := QueryCount()(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "0", rec.Header().Get(HeaderDBQueryCount))
}
