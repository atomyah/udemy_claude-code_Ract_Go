// Package authtoken はアクセストークン（JWT）とリフレッシュトークン（ランダム値）の発行・検証を行う。
// トークンはどちらも HttpOnly Cookie でブラウザに渡す。
package authtoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// AccessTokenTTL はアクセストークンの有効期限
	AccessTokenTTL = time.Hour
	// RefreshTokenTTL はリフレッシュトークンの有効期限
	RefreshTokenTTL = 7 * 24 * time.Hour

	// AccessTokenCookie はアクセストークンを入れる Cookie 名（全APIに送る）
	AccessTokenCookie = "access_token"
	// AccessTokenCookiePath はアクセストークン Cookie の Path
	AccessTokenCookiePath = "/api/v1"
	// RefreshTokenCookie はリフレッシュトークンを入れる Cookie 名
	RefreshTokenCookie = "refresh_token"
	// RefreshTokenCookiePath はリフレッシュトークン Cookie の Path。
	// refresh / logout / revoke-all にだけ送られるよう認証APIのパスに限定する
	RefreshTokenCookiePath = "/api/v1/auth"
)

var (
	// ErrExpired はトークンの有効期限切れを表す
	ErrExpired = errors.New("token expired")
	// ErrInvalid は署名不正・形式不正などのトークンを表す
	ErrInvalid = errors.New("invalid token")
)

// AccessClaims はアクセストークンのペイロード。user_id と exp のみを含む
// （RegisteredClaims の他のフィールドは空なので JSON に出力されない）
type AccessClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// IssueAccessToken は userID のアクセストークン（HS256、有効期限 1 時間）を発行する
func IssueAccessToken(secret string, userID uuid.UUID, now time.Time) (string, error) {
	claims := AccessClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken はアクセストークンを検証して user_id を返す。
// 期限切れは ErrExpired、それ以外の不正は ErrInvalid を返す
func ParseAccessToken(secret, tokenStr string) (uuid.UUID, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(*jwt.Token) (interface{}, error) { return []byte(secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpired
		}
		return uuid.Nil, ErrInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	return userID, nil
}

// NewRefreshToken は推測不能なリフレッシュトークン（256 ビットの乱数を base64url にしたもの）を生成する
func NewRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken は DB に保存・照合するためのハッシュ（HMAC-SHA256 の16進表現）を返す。
// DB にはトークンそのものを保存しないため、DB が漏洩してもトークンとしては使えない
func HashRefreshToken(secret, token string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
