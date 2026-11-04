package service

import (
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifiedGoogleEmail_Success(t *testing.T) {
	email, err := verifiedGoogleEmail(&auth.Token{
		Firebase: auth.FirebaseInfo{SignInProvider: "google.com"},
		Claims:   map[string]interface{}{"email": "taro@example.com", "email_verified": true},
	})

	require.NoError(t, err)
	assert.Equal(t, "taro@example.com", email)
}

func TestVerifiedGoogleEmail_Rejected(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		claims   map[string]interface{}
	}{
		{
			name:     "メール/パスワードで作られた未確認のFirebaseアカウント",
			provider: "password",
			claims:   map[string]interface{}{"email": "victim@example.com", "email_verified": false},
		},
		{
			name:     "メール確認済みでもGoogle以外のプロバイダー",
			provider: "password",
			claims:   map[string]interface{}{"email": "victim@example.com", "email_verified": true},
		},
		{
			name:     "プロバイダーが不明",
			provider: "",
			claims:   map[string]interface{}{"email": "victim@example.com", "email_verified": true},
		},
		{
			name:     "Googleでもメールアドレスが未確認",
			provider: "google.com",
			claims:   map[string]interface{}{"email": "victim@example.com", "email_verified": false},
		},
		{
			name:     "email_verifiedがない",
			provider: "google.com",
			claims:   map[string]interface{}{"email": "victim@example.com"},
		},
		{
			name:     "email_verifiedが真偽値ではない",
			provider: "google.com",
			claims:   map[string]interface{}{"email": "victim@example.com", "email_verified": "true"},
		},
		{
			name:     "メールアドレスがない",
			provider: "google.com",
			claims:   map[string]interface{}{"email_verified": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := verifiedGoogleEmail(&auth.Token{
				Firebase: auth.FirebaseInfo{SignInProvider: tt.provider},
				Claims:   tt.claims,
			})

			assert.ErrorIs(t, err, ErrInvalidIDToken)
		})
	}
}
