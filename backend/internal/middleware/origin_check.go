package middleware

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

// OriginCheck は状態を変更するリクエスト（POST / PUT / PATCH / DELETE）の Origin ヘッダーを検証する CSRF 対策。
//
// 認証 Cookie は SameSite=None のため、他サイトのページからのリクエストにも Cookie が付いてしまう。
// ブラウザはクロスオリジンの状態変更リクエストに必ず Origin を付けるので、
// 許可オリジン（フロントエンド）または API 自身（Swagger UI）以外の Origin は 403 にする。
// Origin がないリクエストはブラウザ以外のクライアント（curl・サーバー間通信）とみなして通す。
func OriginCheck(allowedOrigins []string) echo.MiddlewareFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch c.Request().Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return next(c)
			}

			origin := c.Request().Header.Get(echo.HeaderOrigin)
			if origin == "" {
				return next(c)
			}
			if _, ok := allowed[origin]; ok {
				return next(c)
			}
			if u, err := url.Parse(origin); err == nil && u.Host != "" && u.Host == c.Request().Host {
				return next(c)
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"code":    "INVALID_ORIGIN",
				"message": "許可されていない送信元からのリクエストです",
			})
		}
	}
}
