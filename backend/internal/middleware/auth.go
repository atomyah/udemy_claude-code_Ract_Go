package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// authUserKey は認証済みユーザー（DB から取得した *model.User）を保存する Context のキー。
// OptionalJWTAuth → JWTAuth → AdminOnly と続くときに DB 参照を 1 回で済ませるために使う
const authUserKey = "authUser"

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

// authUserFromContext は同じリクエストで確認済みのユーザーを返す（userID と一致するときだけ）
func authUserFromContext(c echo.Context, userID uuid.UUID) (*model.User, bool) {
	user, ok := c.Get(authUserKey).(*model.User)
	if !ok || user == nil || user.ID != userID {
		return nil, false
	}
	return user, true
}

// loadAuthUser はトークンのユーザーを DB から取得する（同じリクエストで取得済みならそれを使う）
func loadAuthUser(c echo.Context, users UserFinder, userID uuid.UUID) (*model.User, error) {
	if user, ok := authUserFromContext(c, userID); ok {
		return user, nil
	}
	user, err := users.FindByID(c.Request().Context(), userID)
	if err != nil {
		return nil, err
	}
	c.Set(authUserKey, user)
	return user, nil
}

func unauthorized(c echo.Context, code, message string) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{"code": code, "message": message})
}

// JWTAuth はアクセストークンを検証し、userID を Context にセットする。トークンがない・無効なら 401 を返す。
// トークンが有効でも、ユーザーが存在しない・停止されている・承認済みでない（pending / rejected）場合は 401 にする
// （停止はリフレッシュトークンの失効だけでは発行済みのアクセストークン（最長 1 時間）を止められないため、毎回 DB で確認する）
func JWTAuth(secret string, users UserFinder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := accessTokenFromRequest(c)
			if tokenStr == "" {
				return unauthorized(c, "MISSING_TOKEN", "認証トークンが必要です")
			}

			userID, err := authtoken.ParseAccessToken(secret, tokenStr)
			if err != nil {
				if errors.Is(err, authtoken.ErrExpired) {
					return unauthorized(c, "EXPIRED_TOKEN", "トークンの有効期限が切れています")
				}
				return unauthorized(c, "INVALID_TOKEN", "無効なトークンです")
			}

			user, err := loadAuthUser(c, users, userID)
			if errors.Is(err, repository.ErrNotFound) {
				return unauthorized(c, "INVALID_TOKEN", "無効なトークンです")
			}
			if err != nil {
				return httperr.Internal(c, httperr.DefaultInternalMessage, err)
			}
			if user.IsSuspended {
				return unauthorized(c, "ACCOUNT_SUSPENDED", "このアカウントは停止されています")
			}
			// 管理者の承認待ち・拒否されたユーザーは通常機能を使えない
			if !user.IsApproved() {
				return unauthorized(c, "ACCOUNT_NOT_APPROVED", "このアカウントは管理者の承認が完了していないため利用できません")
			}

			c.Set("userID", userID)
			return next(c)
		}
	}
}

// OptionalJWTAuth はトークンがあれば検証して userID を Context にセットするが、
// トークンがない/無効でも401にせずそのまま次のハンドラーに進む（公開エンドポイントの個人化用）。
// 存在しない・停止中・承認済みでないユーザーのトークンは未ログインとして扱う
func OptionalJWTAuth(secret string, users UserFinder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := accessTokenFromRequest(c)
			if tokenStr == "" {
				return next(c)
			}
			userID, err := authtoken.ParseAccessToken(secret, tokenStr)
			if err != nil {
				return next(c)
			}

			user, err := loadAuthUser(c, users, userID)
			if errors.Is(err, repository.ErrNotFound) {
				return next(c)
			}
			if err != nil {
				return httperr.Internal(c, httperr.DefaultInternalMessage, err)
			}
			if !user.IsSuspended && user.IsApproved() {
				c.Set("userID", userID)
			}
			return next(c)
		}
	}
}
