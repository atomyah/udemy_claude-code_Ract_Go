package dto

// UserInPost は投稿に埋め込む軽量ユーザー情報
type UserInPost struct {
	ID          string  `json:"id"           example:"550e8400-e29b-41d4-a716-446655440000"`
	Handle      string  `json:"handle"       example:"john_doe"`
	DisplayName string  `json:"display_name" example:"John Doe"`
	AvatarURL   *string `json:"avatar_url"   example:"https://storage.googleapis.com/bucket/avatar.jpg"`
}

// MediaResponse はメディアファイル情報
type MediaResponse struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440001"`
	URL       string `json:"url"        example:"https://storage.googleapis.com/bucket/image.jpg"`
	Type      string `json:"type"       example:"image"`
	SortOrder int    `json:"sort_order" example:"0"`
}

// PostSummary はリポスト元投稿の要約（再帰定義回避）
type PostSummary struct {
	ID        string          `json:"id"         example:"550e8400-e29b-41d4-a716-446655440002"`
	User      UserInPost      `json:"user"`
	Content   string          `json:"content"    example:"元の投稿内容"`
	Media     []MediaResponse `json:"media"`
	CreatedAt string          `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// PostResponse は投稿レスポンス
type PostResponse struct {
	ID            string          `json:"id"             example:"550e8400-e29b-41d4-a716-446655440003"`
	User          UserInPost      `json:"user"`
	Content       string          `json:"content"        example:"こんにちは！今日もよい天気ですね #天気 @john_doe"`
	Media         []MediaResponse `json:"media"`
	LikesCount    int64           `json:"likes_count"    example:"42"`
	CommentsCount int64           `json:"comments_count" example:"5"`
	RepostsCount  int64           `json:"reposts_count"  example:"3"`
	IsLiked       bool            `json:"is_liked"       example:"false"`
	IsBookmarked  bool            `json:"is_bookmarked"  example:"false"`
	IsReposted    bool            `json:"is_reposted"    example:"false"`
	IsEdited      bool            `json:"is_edited"      example:"false"`
	RepostOf      *PostSummary    `json:"repost_of,omitempty"`
	ReplyTo       *string         `json:"reply_to,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	ReplyToUser   *UserInPost     `json:"reply_to_user,omitempty"`
	CreatedAt     string          `json:"created_at"     example:"2024-01-01T12:00:00Z"`
	UpdatedAt     string          `json:"updated_at"     example:"2024-01-01T12:00:00Z"`
}

// PostListResponse は投稿一覧レスポンス（カーソルページネーション）
type PostListResponse struct {
	Data       []PostResponse `json:"data"`
	NextCursor *string        `json:"next_cursor" example:"2024-01-01T11:59:59Z"`
	HasMore    bool           `json:"has_more"    example:"true"`
}

// CreatePostRequest は投稿作成リクエスト（multipart/form-data）
type CreatePostRequest struct {
	Content string `form:"content" example:"こんにちは！ #test" validate:"required,max=280"`
}

// UpdatePostRequest は投稿編集リクエスト
type UpdatePostRequest struct {
	Content string `json:"content" example:"編集後の内容" validate:"required,max=280"`
}
