package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/labstack/echo/v4"
)

// accessTokenFromRequest はアクセストークンを取り出す。
// ブラウザ（フロントエンド）は HttpOnly Cookie で送る。Swagger UI・計測スクリプトなどブラウザ以外の
// クライアント向けに Authorization: Bearer ヘッダーも受け付ける（ヘッダーは CSRF の対象にならない）
func accessTokenFromRequest(c echo.Context) string {
	if h := c.Request().Header.Get(echo.HeaderAuthorization); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if cookie, err := c.Cookie(authtoken.AccessTokenCookie); err == nil {
		return cookie.Value
	}
	return ""
}

// JWTAuth はアクセストークンを検証し、userID を Context にセットする。トークンがない・無効なら 401 を返す
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := accessTokenFromRequest(c)
			if tokenStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"code":    "MISSING_TOKEN",
					"message": "認証トークンが必要です",
				})
			}

			userID, err := authtoken.ParseAccessToken(secret, tokenStr)
			if err != nil {
				if errors.Is(err, authtoken.ErrExpired) {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"code":    "EXPIRED_TOKEN",
						"message": "トークンの有効期限が切れています",
					})
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"code":    "INVALID_TOKEN",
					"message": "無効なトークンです",
				})
			}

			c.Set("userID", userID)
			return next(c)
		}
	}
}

// OptionalJWTAuth はトークンがあれば検証して userID を Context にセットするが、
// トークンがない/無効でも401にせずそのまま次のハンドラーに進む（公開エンドポイントの個人化用）
func OptionalJWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := accessTokenFromRequest(c)
			if tokenStr == "" {
				return next(c)
			}
			if userID, err := authtoken.ParseAccessToken(secret, tokenStr); err == nil {
				c.Set("userID", userID)
			}
			return next(c)
		}
	}
}
