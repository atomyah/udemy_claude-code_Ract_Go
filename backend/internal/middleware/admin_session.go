package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/atyahara/sns-backend/internal/adminsession"
	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// 管理画面のセッション情報を保存する Context のキー
const (
	adminPrincipalKey    = "adminPrincipal"
	adminSessionTokenKey = "adminSessionToken"
)

// AdminLoginPath は管理画面のログインページ
const AdminLoginPath = "/admin/login"

// CSRFFormField は管理画面のフォームに埋め込む CSRF トークンのフィールド名
const CSRFFormField = "_csrf"

// AdminFinder は管理画面のセッション確認で admins テーブルの管理者を取得するインターフェース
type AdminFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.Admin, error)
}

// AdminPrincipalFromContext は AdminSession が確認したログイン中の管理者を返す
func AdminPrincipalFromContext(c echo.Context) (adminsession.Principal, bool) {
	p, ok := c.Get(adminPrincipalKey).(adminsession.Principal)
	return p, ok
}

// AdminCSRFToken はログイン中の管理者のセッションに結びついた CSRF トークンを返す（テンプレートに埋め込む）
func AdminCSRFToken(c echo.Context, secret string) string {
	token, _ := c.Get(adminSessionTokenKey).(string)
	if token == "" {
		return ""
	}
	return adminsession.CSRFToken(secret, token)
}

// AdminSecurityHeaders は管理画面のレスポンスにセキュリティヘッダーを付ける。
// 画面は外部リソースを使わないため CSP で自オリジン以外を禁止し、iframe への埋め込み・キャッシュも禁止する
func AdminSecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			// no-referrer にするとブラウザがフォーム送信時に Origin: null を送り、OriginCheck で拒否されるため same-origin にする
			h.Set("Referrer-Policy", "same-origin")
			h.Set("Cache-Control", "no-store")
			return next(c)
		}
	}
}

// AdminSession は管理画面の全ルート（ログインページ以外）に適用する認証ミドルウェア。
//   - セッション Cookie の署名・有効期限を検証する（一般ユーザーのアクセストークンでは通らない）
//   - admins テーブルの管理者は毎リクエスト DB で存在・有効（is_active）を確認する（無効化を即時反映するため）
//   - POST など状態を変更するリクエストは CSRF トークンを検証する
//
// 未ログイン・セッション切れはログインページへリダイレクトする
func AdminSession(secret string, admins AdminFinder, cookieSecure bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(adminsession.CookieName)
			if err != nil || cookie.Value == "" {
				return c.Redirect(http.StatusSeeOther, AdminLoginPath)
			}

			p, err := adminsession.Parse(secret, cookie.Value)
			if err != nil {
				ClearAdminSessionCookie(c, cookieSecure)
				return c.Redirect(http.StatusSeeOther, AdminLoginPath+"?expired=1")
			}

			if p.Kind == adminsession.KindAdmin {
				admin, err := admins.FindByID(c.Request().Context(), p.AdminID)
				if errors.Is(err, repository.ErrNotFound) || (err == nil && !admin.IsActive) {
					ClearAdminSessionCookie(c, cookieSecure)
					return c.Redirect(http.StatusSeeOther, AdminLoginPath)
				}
				if err != nil {
					return httperr.Internal(c, httperr.DefaultInternalMessage, err)
				}
			}

			switch c.Request().Method {
			case http.MethodGet, http.MethodHead:
			default:
				if !adminsession.ValidCSRFToken(secret, cookie.Value, c.FormValue(CSRFFormField)) {
					return c.HTML(http.StatusForbidden,
						`<!doctype html><meta charset="utf-8"><title>操作できません</title>`+
							`<p>画面の有効期限が切れたか、不正な送信元からの操作です。ページを再読み込みしてから操作してください。</p>`+
							`<p><a href="/admin">管理画面トップへ戻る</a></p>`)
				}
			}

			c.Set(adminPrincipalKey, p)
			c.Set(adminSessionTokenKey, cookie.Value)
			return next(c)
		}
	}
}

// SetAdminSessionCookie は管理画面のセッション Cookie を発行する。
// 管理画面は API と同じオリジンで動くため SameSite=Strict にし、他サイトからのリクエストには Cookie を送らせない
func SetAdminSessionCookie(c echo.Context, token string, secure bool) {
	c.SetCookie(adminSessionCookie(token, int(adminsession.TTL.Seconds()), secure))
}

// ClearAdminSessionCookie は管理画面のセッション Cookie を削除する
func ClearAdminSessionCookie(c echo.Context, secure bool) {
	c.SetCookie(adminSessionCookie("", -1, secure))
}

func adminSessionCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     adminsession.CookieName,
		Value:    value,
		Path:     adminsession.CookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}
