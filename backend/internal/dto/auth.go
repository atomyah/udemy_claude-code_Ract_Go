package dto

// RegisterRequest は新規ユーザー登録リクエスト
type RegisterRequest struct {
	Email       string `json:"email"        example:"user@example.com"  validate:"required,email"`
	Password    string `json:"password"     example:"password123"        validate:"required,min=8,max=72"`
	Handle      string `json:"handle"       example:"john_doe"            validate:"required,min=3,max=50"`
	DisplayName string `json:"display_name" example:"John Doe"            validate:"required,max=50"`
}

// LoginRequest はログインリクエスト
type LoginRequest struct {
	Email    string `json:"email"    example:"user@example.com" validate:"required,email"`
	Password string `json:"password" example:"password123"      validate:"required"`
}

// GoogleLoginRequest はGoogle OAuth ログインリクエスト
type GoogleLoginRequest struct {
	IDToken string `json:"id_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." validate:"required"`
}

// AuthResponse は認証成功レスポンス
type AuthResponse struct {
	AccessToken string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User        UserResponse `json:"user"`
}

// RefreshResponse はトークンリフレッシュレスポンス
type RefreshResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
