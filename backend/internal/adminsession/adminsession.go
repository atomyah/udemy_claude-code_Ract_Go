// Package adminsession は管理画面のログインセッション（署名付き Cookie）と CSRF トークンを扱う。
// 管理画面の利用者は一般ユーザー（users）とは別で、環境変数で定義するルート管理者と admins テーブルの管理者の 2 種類がある。
package adminsession

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// CookieName は管理画面のセッション Cookie 名
	CookieName = "admin_session"
	// CookiePath はセッション Cookie の Path（管理画面のルートにだけ送る）
	CookiePath = "/admin"
	// TTL はセッションの有効期限（期限が切れたら再ログイン）
	TTL = 8 * time.Hour
)

// 管理者の種類
const (
	// KindRoot は環境変数で定義するルート管理者（admins テーブルに存在しない）
	KindRoot = "root"
	// KindAdmin は admins テーブルに登録された管理者
	KindAdmin = "admin"
)

var (
	// ErrInvalid は署名不正・形式不正・期限切れのセッションを表す
	ErrInvalid = errors.New("invalid admin session")
)

// Principal はログイン中の管理者。監査ログの「誰が」に使う
type Principal struct {
	Kind string
	// AdminID は admins テーブルの ID（ルート管理者は uuid.Nil）
	AdminID  uuid.UUID
	Username string
}

// IsRoot はルート管理者かどうかを返す
func (p Principal) IsRoot() bool {
	return p.Kind == KindRoot
}

// Label は監査記録（reviewed_by / handled_by など）に残す管理者の表記（例: "root:admin" / "admin:yamada"）
func (p Principal) Label() string {
	return p.Kind + ":" + p.Username
}

// ID はログに出す管理者の識別子（ルート管理者は "root"）
func (p Principal) ID() string {
	if p.IsRoot() {
		return KindRoot
	}
	return p.AdminID.String()
}

type claims struct {
	Kind     string `json:"kind"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Issue はセッション Cookie に入れる署名付きトークン（HS256）を発行する
func Issue(secret string, p Principal, now time.Time) (string, error) {
	c := claims{
		Kind:     p.Kind,
		Username: p.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.ID(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TTL)),
			// 一般ユーザーのアクセストークンと取り違えないよう用途を入れる
			Audience: jwt.ClaimStrings{"admin-panel"},
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// Parse はセッショントークンを検証して管理者を返す。
// admins テーブルの管理者が現在も有効かどうかは呼び出し側（ミドルウェア）で DB を見て確認する
func Parse(secret, token string) (Principal, error) {
	c := &claims{}
	_, err := jwt.ParseWithClaims(token, c,
		func(*jwt.Token) (interface{}, error) { return []byte(secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience("admin-panel"),
	)
	if err != nil {
		return Principal{}, ErrInvalid
	}

	switch c.Kind {
	case KindRoot:
		if c.Subject != KindRoot || c.Username == "" {
			return Principal{}, ErrInvalid
		}
		return Principal{Kind: KindRoot, Username: c.Username}, nil
	case KindAdmin:
		id, err := uuid.Parse(c.Subject)
		if err != nil {
			return Principal{}, ErrInvalid
		}
		return Principal{Kind: KindAdmin, AdminID: id, Username: c.Username}, nil
	default:
		return Principal{}, ErrInvalid
	}
}

// CSRFToken はセッションに結びついた CSRF トークンを返す。
// 管理画面のフォームに hidden で埋め込み、POST 時に一致を確認する（セッションが変わると無効になる）
func CSRFToken(secret, sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("csrf:" + sessionToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidCSRFToken は CSRF トークンが一致するかを定数時間で比較する
func ValidCSRFToken(secret, sessionToken, token string) bool {
	expected := CSRFToken(secret, sessionToken)
	return hmac.Equal([]byte(expected), []byte(token))
}
