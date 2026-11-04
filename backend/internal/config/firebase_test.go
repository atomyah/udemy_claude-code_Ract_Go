package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServiceAccount_Success(t *testing.T) {
	sa, err := ParseServiceAccount(`{"type":"service_account","client_email":"a@example.com","private_key":"dummy-key"}`)

	require.NoError(t, err)
	assert.Equal(t, "a@example.com", sa.ClientEmail)
	assert.Equal(t, "dummy-key", sa.PrivateKey)
}

func TestParseServiceAccount_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "JSONではない", value: "secret-value-not-json"},
		{name: "未設定時の既定値", value: "{}"},
		{name: "秘密鍵がない", value: `{"client_email":"a@example.com"}`},
		{name: "メールアドレスがない", value: `{"private_key":"dummy-key"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseServiceAccount(tt.value)

			require.Error(t, err)
			// エラーはログに残るため、環境変数の値を含めない
			assert.NotContains(t, err.Error(), "secret-value")
			assert.NotContains(t, err.Error(), "dummy-key")
		})
	}
}
