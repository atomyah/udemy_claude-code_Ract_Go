package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// newJSONContext はJSONボディを持つechoコンテキストを生成する
func newJSONContext(t *testing.T, method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// newFormContext はmultipart/form-dataのechoコンテキストを生成する
func newFormContext(t *testing.T, method, target string, fields map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	require.NoError(t, writer.Close())

	e := echo.New()
	req := httptest.NewRequest(method, target, body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// withAuth は認証済みユーザーをコンテキストにセットする（JWTミドルウェア通過後の状態を再現）
func withAuth(c echo.Context, userID uuid.UUID) echo.Context {
	c.Set("userID", userID)
	c.Set("userRole", "user")
	return c
}

// withParam はパスパラメータをセットする
func withParam(c echo.Context, names []string, values []string) echo.Context {
	c.SetParamNames(names...)
	c.SetParamValues(values...)
	return c
}

// decodeErrorResponse はエラーレスポンスをデコードする
func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) dto.ErrorResponse {
	t.Helper()
	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

// decodeJSON は成功レスポンスを任意の型にデコードする
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), target))
}

// findCookie はレスポンスから指定名のCookieを取得する
func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
