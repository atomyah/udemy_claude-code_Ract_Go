package authtoken

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "authtoken-test-secret"

// decodePayload は JWT のペイロード部分を map にデコードする
func decodePayload(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func TestIssueAccessToken_PayloadHasOnlyUserIDAndExp(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	token, err := IssueAccessToken(testSecret, userID, now)
	require.NoError(t, err)

	payload := decodePayload(t, token)
	assert.Len(t, payload, 2, "ペイロードは user_id と exp のみ: %v", payload)
	assert.Equal(t, userID.String(), payload["user_id"])
	assert.Equal(t, float64(now.Add(time.Hour).Unix()), payload["exp"], "有効期限は1時間")
}

func TestParseAccessToken_ValidToken_ReturnsUserID(t *testing.T) {
	userID := uuid.New()
	token, err := IssueAccessToken(testSecret, userID, time.Now())
	require.NoError(t, err)

	got, err := ParseAccessToken(testSecret, token)

	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

func TestParseAccessToken_ExpiredAfterOneHour(t *testing.T) {
	// 1時間と1秒前に発行したトークンは期限切れ
	token, err := IssueAccessToken(testSecret, uuid.New(), time.Now().Add(-time.Hour-time.Second))
	require.NoError(t, err)

	_, err = ParseAccessToken(testSecret, token)

	assert.ErrorIs(t, err, ErrExpired)
}

func TestParseAccessToken_StillValidJustBeforeOneHour(t *testing.T) {
	token, err := IssueAccessToken(testSecret, uuid.New(), time.Now().Add(-59*time.Minute))
	require.NoError(t, err)

	_, err = ParseAccessToken(testSecret, token)

	assert.NoError(t, err)
}

func TestParseAccessToken_InvalidTokens(t *testing.T) {
	sign := func(method jwt.SigningMethod, key interface{}, claims jwt.Claims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		require.NoError(t, err)
		return s
	}
	exp := jwt.NewNumericDate(time.Now().Add(time.Hour))
	valid := AccessClaims{UserID: uuid.NewString(), RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: exp}}

	tests := []struct {
		name  string
		token string
	}{
		{"別の鍵で署名", sign(jwt.SigningMethodHS256, []byte("other-secret"), valid)},
		{"HS512（許可していないアルゴリズム）", sign(jwt.SigningMethodHS512, []byte(testSecret), valid)},
		{"alg=none", sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid)},
		{"exp なし", sign(jwt.SigningMethodHS256, []byte(testSecret), AccessClaims{UserID: uuid.NewString()})},
		{"user_id が UUID でない", sign(jwt.SigningMethodHS256, []byte(testSecret), AccessClaims{UserID: "admin", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: exp}})},
		{"user_id なし（旧形式の sub のみ）", sign(jwt.SigningMethodHS256, []byte(testSecret), jwt.RegisteredClaims{Subject: uuid.NewString(), ExpiresAt: exp})},
		{"JWT でない文字列", "not-a-jwt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAccessToken(testSecret, tt.token)
			assert.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestNewRefreshToken_IsRandomAndURLSafe(t *testing.T) {
	a, err := NewRefreshToken()
	require.NoError(t, err)
	b, err := NewRefreshToken()
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
	raw, err := base64.RawURLEncoding.DecodeString(a)
	require.NoError(t, err, "Cookie にそのまま入れられる base64url であること")
	assert.Len(t, raw, 32, "256ビットの乱数")
}

func TestHashRefreshToken_IsDeterministicAndKeyed(t *testing.T) {
	h1 := HashRefreshToken(testSecret, "token")
	h2 := HashRefreshToken(testSecret, "token")

	assert.Equal(t, h1, h2, "同じトークンは同じハッシュになる（DB で照合できる）")
	assert.Len(t, h1, 64)
	assert.NotEqual(t, h1, HashRefreshToken(testSecret, "other"))
	assert.NotEqual(t, h1, HashRefreshToken("other-secret", "token"), "鍵が違えばハッシュも違う")
	assert.NotContains(t, h1, "token")
}
