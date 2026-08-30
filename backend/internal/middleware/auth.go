package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type JWTClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"code":    "MISSING_TOKEN",
					"message": "認証トークンが必要です",
				})
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims := &JWTClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				if errors.Is(err, jwt.ErrTokenExpired) {
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

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"code":    "INVALID_TOKEN",
					"message": "無効なトークンです",
				})
			}

			c.Set("userID", userID)
			c.Set("userRole", claims.Role)
			return next(c)
		}
	}
}

// OptionalJWTAuth はトークンがあれば検証してuserID/userRoleをContextにセットするが、
// トークンがない/無効でも401にせずそのまま次のハンドラーに進む（公開エンドポイントの個人化用）
func OptionalJWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return next(c)
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims := &JWTClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return next(c)
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return next(c)
			}

			c.Set("userID", userID)
			c.Set("userRole", claims.Role)
			return next(c)
		}
	}
}
