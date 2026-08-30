package handler

import (
	"errors"
	"testing"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationMessage_ReturnsJapaneseMessages(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request interface{}
		want    string
	}{
		{
			name:    "メールアドレス未入力",
			request: &dto.RegisterRequest{Password: "password123", Handle: "taro", DisplayName: "タロウ"},
			want:    "メールアドレスを入力してください",
		},
		{
			name:    "メールアドレスの形式不正",
			request: &dto.RegisterRequest{Email: "invalid", Password: "password123", Handle: "taro", DisplayName: "タロウ"},
			want:    "メールアドレスの形式が正しくありません",
		},
		{
			name:    "パスワードが短い",
			request: &dto.RegisterRequest{Email: "taro@example.com", Password: "short", Handle: "taro", DisplayName: "タロウ"},
			want:    "パスワードは8文字以上で入力してください",
		},
		{
			name:    "テーマの値が不正",
			request: &dto.UpdateThemeRequest{Theme: "blue"},
			want:    "テーマはlight・darkのいずれかを指定してください",
		},
		{
			name:    "投稿本文が未入力",
			request: &dto.UpdatePostRequest{Content: ""},
			want:    "本文を入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			require.Error(t, err)
			assert.Equal(t, tt.want, validationMessage(err))
		})
	}
}

func TestValidationMessage_MultipleFieldsAreJoined(t *testing.T) {
	err := validator.New().Struct(&dto.RegisterRequest{})
	require.Error(t, err)

	message := validationMessage(err)

	assert.Contains(t, message, "メールアドレスを入力してください")
	assert.Contains(t, message, "パスワードを入力してください")
	assert.Contains(t, message, "ユーザーIDを入力してください")
	assert.Contains(t, message, "表示名を入力してください")
	assert.Contains(t, message, "、")
}

func TestValidationMessage_NonValidationError(t *testing.T) {
	assert.Equal(t, "入力内容が正しくありません", validationMessage(errors.New("なにかのエラー")))
	assert.Equal(t, "", validationMessage(nil))
}

func TestFieldLabel_UnknownFieldFallsBackToFieldName(t *testing.T) {
	assert.Equal(t, "メールアドレス", fieldLabel("Email"))
	assert.Equal(t, "UnknownField", fieldLabel("UnknownField"))
}
