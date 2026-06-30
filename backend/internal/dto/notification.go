package dto

// NotificationResponse は通知レスポンス
type NotificationResponse struct {
	ID        string       `json:"id"         example:"550e8400-e29b-41d4-a716-446655440005"`
	Type      string       `json:"type"       example:"like"`
	Actor     UserInPost   `json:"actor"`
	Post      *PostSummary `json:"post,omitempty"`
	IsRead    bool         `json:"is_read"    example:"false"`
	CreatedAt string       `json:"created_at" example:"2024-01-01T12:00:00Z"`
}

// NotificationListResponse は通知一覧レスポンス
type NotificationListResponse struct {
	Data        []NotificationResponse `json:"data"`
	NextCursor  *string                `json:"next_cursor"  example:"2024-01-01T11:59:59Z"`
	HasMore     bool                   `json:"has_more"     example:"false"`
	UnreadCount int64                  `json:"unread_count" example:"3"`
}
