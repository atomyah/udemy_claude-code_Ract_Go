package dto

// ErrorResponse はAPIエラーの統一レスポンス形式
type ErrorResponse struct {
	Code    string `json:"code"    example:"INVALID_REQUEST"`
	Message string `json:"message" example:"リクエストが不正です"`
}

// SuccessResponse は204以外の成功レスポンスで使う汎用型
type SuccessResponse struct {
	Message string `json:"message" example:"success"`
}
