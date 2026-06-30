package dto

// LikeResponse はいいねのトグル結果
type LikeResponse struct {
	PostID     string `json:"post_id"     example:"550e8400-e29b-41d4-a716-446655440003"`
	LikesCount int64  `json:"likes_count" example:"43"`
	IsLiked    bool   `json:"is_liked"    example:"true"`
}

// RepostResponse はリポストのトグル結果
type RepostResponse struct {
	PostID       string `json:"post_id"       example:"550e8400-e29b-41d4-a716-446655440003"`
	RepostsCount int64  `json:"reposts_count" example:"4"`
	IsReposted   bool   `json:"is_reposted"   example:"true"`
}
