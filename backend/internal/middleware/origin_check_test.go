package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestOriginCheck(t *testing.T) {
	allowed := []string{"http://localhost:5173", "https://app.example.com"}

	tests := []struct {
		name   string
		method string
		origin string
		host   string
		want   int
	}{
		{"許可オリジンからのPOSTは通す", http.MethodPost, "https://app.example.com", "api.example.com", http.StatusOK},
		{"他サイトからのPOSTは403", http.MethodPost, "https://evil.example.net", "api.example.com", http.StatusForbidden},
		{"他サイトからのPUTは403", http.MethodPut, "https://evil.example.net", "api.example.com", http.StatusForbidden},
		{"他サイトからのDELETEは403", http.MethodDelete, "https://evil.example.net", "api.example.com", http.StatusForbidden},
		{"Origin: null は403", http.MethodPost, "null", "api.example.com", http.StatusForbidden},
		{"ポート違いは別オリジンとして403", http.MethodPost, "http://localhost:6666", "localhost:8080", http.StatusForbidden},
		{"API自身のオリジン（Swagger UI）は通す", http.MethodPost, "http://localhost:8080", "localhost:8080", http.StatusOK},
		{"Originなし（ブラウザ以外のクライアント）は通す", http.MethodPost, "", "api.example.com", http.StatusOK},
		{"GETは他サイトでも通す（状態を変更しない）", http.MethodGet, "https://evil.example.net", "api.example.com", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tt.method, "/api/v1/posts", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set(echo.HeaderOrigin, tt.origin)
			}
			rec := httptest.NewRecorder()

			assert.NoError(t, OriginCheck(allowed)(okHandler)(e.NewContext(req, rec)))

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusForbidden {
				body := decodeError(t, rec)
				assert.Equal(t, "INVALID_ORIGIN", body["code"])
				assert.Equal(t, "許可されていない送信元からのリクエストです", body["message"])
			}
		})
	}
}
