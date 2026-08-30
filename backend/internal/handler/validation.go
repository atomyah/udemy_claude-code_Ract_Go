package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// fieldLabels はDTOのフィールド名をユーザー向けの日本語ラベルに対応づける
var fieldLabels = map[string]string{
	"Email":           "メールアドレス",
	"NewEmail":        "新しいメールアドレス",
	"Password":        "パスワード",
	"CurrentPassword": "現在のパスワード",
	"NewPassword":     "新しいパスワード",
	"Handle":          "ユーザーID",
	"DisplayName":     "表示名",
	"Content":         "本文",
	"Bio":             "自己紹介",
	"Location":        "場所",
	"WebsiteURL":      "ウェブサイトURL",
	"Birthday":        "誕生日",
	"Theme":           "テーマ",
	"IDToken":         "IDトークン",
}

// fieldLabel はフィールド名に対応する日本語ラベルを返す（未定義ならフィールド名をそのまま返す）
func fieldLabel(field string) string {
	if label, ok := fieldLabels[field]; ok {
		return label
	}
	return field
}

// validationMessage はvalidatorのエラーをユーザーが理解できる日本語メッセージに変換する。
// 複数フィールドがエラーの場合は「、」で連結する。
func validationMessage(err error) string {
	if err == nil {
		return ""
	}

	var fieldErrs validator.ValidationErrors
	if !asValidationErrors(err, &fieldErrs) {
		return "入力内容が正しくありません"
	}

	messages := make([]string, 0, len(fieldErrs))
	for _, fe := range fieldErrs {
		messages = append(messages, fieldErrorMessage(fe))
	}
	if len(messages) == 0 {
		return "入力内容が正しくありません"
	}
	return strings.Join(messages, "、")
}

func asValidationErrors(err error, target *validator.ValidationErrors) bool {
	fieldErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return false
	}
	*target = fieldErrs
	return true
}

// fieldErrorMessage は1件のバリデーションエラーを日本語メッセージに変換する
func fieldErrorMessage(fe validator.FieldError) string {
	label := fieldLabel(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%sを入力してください", label)
	case "email":
		return fmt.Sprintf("%sの形式が正しくありません", label)
	case "min":
		return fmt.Sprintf("%sは%s文字以上で入力してください", label, fe.Param())
	case "max":
		return fmt.Sprintf("%sは%s文字以内で入力してください", label, fe.Param())
	case "len":
		return fmt.Sprintf("%sは%s文字で入力してください", label, fe.Param())
	case "oneof":
		return fmt.Sprintf("%sは%sのいずれかを指定してください", label, strings.ReplaceAll(fe.Param(), " ", "・"))
	case "url":
		return fmt.Sprintf("%sはURLの形式で入力してください", label)
	default:
		return fmt.Sprintf("%sの入力内容が正しくありません", label)
	}
}
