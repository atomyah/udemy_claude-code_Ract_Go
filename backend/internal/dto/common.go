package dto

// ErrorResponse はAPIエラーの統一レスポンス形式
type ErrorResponse struct {
	Code    string `json:"code"    example:"INVALID_REQUEST"`
	Message string `json:"message" example:"リクエストが不正です"`
	// RequestID はサーバーエラー（5xx）のときだけ返す。問い合わせ時にログと突き合わせるためのID
	RequestID string `json:"request_id,omitempty" example:"0b5e2f9c-3d6a-4f0e-9a51-7c2d8e4b1f60"`
}

// SuccessResponse は204以外の成功レスポンスで使う汎用型
type SuccessResponse struct {
	Message string `json:"message" example:"success"`
}
