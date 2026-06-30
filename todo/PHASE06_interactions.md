# PHASE 06 — バックエンド インタラクション（いいね・コメント・リポスト・ブックマーク）

> 目標: 投稿へのリアクション機能と通知トリガーをまとめて実装する。

---

## DTO（`internal/dto/interaction.go`）

- [ ] `LikeResponse`（post_id, likes_count, is_liked）
- [ ] `RepostResponse`（post_id, reposts_count, is_reposted）
- [ ] `BookmarkListResponse`（posts: []PostResponse, next_cursor, has_more）

## リポジトリ（`internal/repository/like_repository.go`）

- [ ] `Create(ctx, userID, postID) error`
- [ ] `Delete(ctx, userID, postID) error`
- [ ] `CountByPost(ctx, postID) (int64, error)`
- [ ] `IsLiked(ctx, userID, postID) (bool, error)`

## リポジトリ（`internal/repository/bookmark_repository.go`）

- [ ] `Create(ctx, userID, postID) error`
- [ ] `Delete(ctx, userID, postID) error`
- [ ] `GetByUser(ctx, userID, cursor, limit) ([]model.Post, string, error)`
- [ ] `IsBookmarked(ctx, userID, postID) (bool, error)`

## リポジトリ（`internal/repository/notification_repository.go`）

- [ ] `Create(ctx, notification) error`
- [ ] `GetByUser(ctx, userID, cursor, limit) ([]model.Notification, string, error)`
- [ ] `MarkAllRead(ctx, userID) error`
- [ ] `CountUnread(ctx, userID) (int64, error)`

## サービス（`internal/service/like_service.go`）

- [ ] `Like(ctx, userID, postID) (*dto.LikeResponse, error)`
  - 重複いいね → 409 `ALREADY_LIKED`
  - 自分の投稿以外なら通知を作成（type: `like`）
- [ ] `Unlike(ctx, userID, postID) (*dto.LikeResponse, error)`

## サービス（`internal/service/repost_service.go`）

リポストは `posts` テーブルに `repost_of` を持つレコードを作成する方式。

- [ ] `Repost(ctx, userID, postID) (*dto.RepostResponse, error)`
  - 既にリポスト済みなら 409 `ALREADY_REPOSTED`
  - 通知を作成（type: `repost`）
- [ ] `Unrepost(ctx, userID, postID) (*dto.RepostResponse, error)`
  - repost_of == postID かつ user_id == userID のレコードを is_deleted=true に更新

## サービス（`internal/service/bookmark_service.go`）

- [ ] `Bookmark(ctx, userID, postID) error`
  - 重複 → 409 `ALREADY_BOOKMARKED`
- [ ] `Unbookmark(ctx, userID, postID) error`
- [ ] `GetBookmarks(ctx, userID, cursor, limit) (*dto.BookmarkListResponse, error)`

## サービス（`internal/service/comment_service.go`）

コメントは `posts` テーブルに `reply_to` を持つ通常投稿として作成。

- [ ] `CreateComment(ctx, userID, postID, req, files) (*dto.PostResponse, error)`
  - post_service.CreatePost を内部的に呼び、reply_to を設定
  - 通知を作成（type: `comment`）

## 通知サービス（`internal/service/notification_service.go`）

- [ ] `Notify(ctx, notification) error`— 共通の通知作成ヘルパー
  - 通知先が自分自身の場合は作成しない
- [ ] `GetNotifications(ctx, userID, cursor, limit) (*dto.NotificationListResponse, error)`
- [ ] `MarkAllRead(ctx, userID) error`

## ハンドラー（`internal/handler/interaction_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [ ] `POST /api/v1/posts/:id/like`（要認証）
- [ ] `DELETE /api/v1/posts/:id/like`（要認証）
- [ ] `POST /api/v1/posts/:id/repost`（要認証）
- [ ] `DELETE /api/v1/posts/:id/repost`（要認証）
- [ ] `POST /api/v1/posts/:id/bookmark`（要認証）
- [ ] `DELETE /api/v1/posts/:id/bookmark`（要認証）
- [ ] `GET /api/v1/bookmarks`（要認証）
- [ ] `POST /api/v1/posts/:id/comments`（要認証）

## ハンドラー（`internal/handler/notification_handler.go`）

- [ ] `GET /api/v1/notifications`（要認証）
- [ ] `PUT /api/v1/notifications/read`（要認証）

## PostResponse の集計値対応

`post_service.go` の GetPost / GetTimeline 系で以下を含めて返すように更新:

- [ ] `likes_count`
- [ ] `comments_count`（reply_to == post.ID の件数）
- [ ] `reposts_count`（repost_of == post.ID の件数）
- [ ] `is_liked`（ログインユーザーがいいね済みか）
- [ ] `is_bookmarked`
- [ ] `is_reposted`

## 完了基準

- [ ] いいね → カウントが増える → もう一度押すと取消
- [ ] 自分の投稿以外をいいねすると通知が作成される
- [ ] リポストで `posts` テーブルに repost_of 付きレコードが作成される
- [ ] コメントで `posts` テーブルに reply_to 付きレコードが作成される
- [ ] ブックマーク一覧が cursor ページネーションで取得できる
