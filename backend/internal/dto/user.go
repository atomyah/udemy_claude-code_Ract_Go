package dto

// UserResponse はユーザー情報レスポンス
type UserResponse struct {
	ID             string  `json:"id"              example:"550e8400-e29b-41d4-a716-446655440000"`
	Handle         string  `json:"handle"          example:"john_doe"`
	DisplayName    string  `json:"display_name"    example:"John Doe"`
	AvatarURL      *string `json:"avatar_url"      example:"https://storage.googleapis.com/bucket/avatar.jpg"`
	BannerURL      *string `json:"banner_url"      example:"https://storage.googleapis.com/bucket/banner.jpg"`
	Bio            *string `json:"bio"             example:"ソフトウェアエンジニアです"`
	Location       *string `json:"location"        example:"東京"`
	WebsiteURL     *string `json:"website_url"     example:"https://example.com"`
	Birthday       *string `json:"birthday"        example:"1990-01-15"`
	Theme          string  `json:"theme"           example:"light"`
	FollowersCount int64   `json:"followers_count" example:"120"`
	FollowingCount int64   `json:"following_count" example:"80"`
	IsFollowing    bool    `json:"is_following"    example:"false"`
	CreatedAt      string  `json:"created_at"      example:"2024-01-01T00:00:00Z"`
}

// UserListResponse はユーザー一覧レスポンス（カーソルページネーション）
type UserListResponse struct {
	Data       []UserResponse `json:"data"`
	NextCursor *string        `json:"next_cursor" example:"2024-01-01T00:00:00Z"`
	HasMore    bool           `json:"has_more"    example:"true"`
}

// UpdateProfileRequest はプロフィール更新リクエスト
type UpdateProfileRequest struct {
	DisplayName string  `json:"display_name" example:"John Doe"           validate:"omitempty,max=50"`
	Bio         *string `json:"bio"          example:"ソフトウェアエンジニア" validate:"omitempty,max=160"`
	Location    *string `json:"location"     example:"東京"               validate:"omitempty,max=30"`
	WebsiteURL  *string `json:"website_url"  example:"https://example.com" validate:"omitempty,max=100"`
	Birthday    *string `json:"birthday"     example:"1990-01-15"         validate:"omitempty"`
}

// UpdateThemeRequest はテーマ更新リクエスト
type UpdateThemeRequest struct {
	Theme string `json:"theme" example:"dark" validate:"required,oneof=light dark"`
}

// AvatarResponse はアバター更新レスポンス
type AvatarResponse struct {
	AvatarURL string `json:"avatar_url" example:"https://storage.googleapis.com/bucket/avatar.jpg"`
}

// BannerResponse はバナー更新レスポンス
type BannerResponse struct {
	BannerURL string `json:"banner_url" example:"https://storage.googleapis.com/bucket/banner.jpg"`
}
