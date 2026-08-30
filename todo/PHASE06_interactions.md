# PHASE 06 — バックエンド インタラクション（いいね・コメント・リポスト・ブックマーク）

> 目標: 投稿へのリアクション機能と通知トリガーをまとめて実装する。

---

## DTO（`internal/dto/interaction.go`）

- [x] `LikeResponse`（post_id, likes_count, is_liked）
- [x] `RepostResponse`（post_id, reposts_count, is_reposted）
- [ ] `BookmarkListResponse` — 専用 DTO はなく、既存の `dto.PostListResponse` を再利用（実質的に等価）

## リポジトリ（`internal/repository/like_repository.go`）

- [x] `Create(ctx, userID, postID) error`
- [x] `Delete(ctx, userID, postID) error`
- [x] `CountByPost(ctx, postID) (int64, error)`
- [x] `IsLiked(ctx, userID, postID) (bool, error)`

## リポジトリ（`internal/repository/bookmark_repository.go`）

- [x] `Create(ctx, userID, postID) error`
- [x] `Delete(ctx, userID, postID) error`
- [x] `GetByUser(ctx, userID, cursor, limit) ([]model.Post, string, error)`
- [x] `IsBookmarked(ctx, userID, postID) (bool, error)`

## リポジトリ（`internal/repository/notification_repository.go`）

- [x] `Create(ctx, notification) error`
- [x] `GetByUser(ctx, userID, cursor, limit) ([]model.Notification, string, error)`
- [x] `MarkAllRead(ctx, userID) error`
- [x] `CountUnread(ctx, userID) (int64, error)`

## サービス（`internal/service/like_service.go`）

- [x] `Like(ctx, userID, postID) (*dto.LikeResponse, error)`
  - 重複いいね → `ErrAlreadyLiked`（ハンドラー側で 409 に変換）
  - 自分の投稿以外なら通知を作成（type: `like`）
- [x] `Unlike(ctx, userID, postID) (*dto.LikeResponse, error)`

## サービス（`internal/service/repost_service.go`）

リポストは `posts` テーブルに `repost_of` を持つレコードを作成する方式。

- [x] `Repost(ctx, userID, postID, req, files) (*dto.RepostResponse, error)`
  - 既にリポスト済みなら 409 相当のエラー
  - 添付メディアは画像最大 2 枚のみ（動画不可）
  - 通知を作成（type: `repost`）
- [x] `Unrepost(ctx, userID, postID) (*dto.RepostResponse, error)`
  - repost_of == postID かつ user_id == userID のレコードを is_deleted=true に更新

## サービス（`internal/service/bookmark_service.go`）

- [x] `Bookmark(ctx, userID, postID) error`
  - 重複 → 409 相当のエラー
- [x] `Unbookmark(ctx, userID, postID) error`
- [x] `GetBookmarks(ctx, userID, cursor, limit) (*dto.BookmarkListResponse, error)`（`dto.PostListResponse` を返す）

## サービス（`internal/service/comment_service.go`）

コメントは `posts` テーブルに `reply_to` を持つ通常投稿として作成。

- [x] `CreateComment(ctx, userID, postID, req, files) (*dto.PostResponse, error)`
  - post_service.CreatePost を内部的に呼び、reply_to を設定
  - 添付メディアは画像最大 2 枚のみ（動画不可）
  - 通知を作成（type: `comment`）
  - 動作確認済み: コメント投稿 → 即座に一覧・カウントに反映

## 通知サービス（`internal/service/notification_service.go`）

- [x] `Notify(ctx, notification) error`— 共通の通知作成ヘルパー
  - 通知先が自分自身の場合は作成しない（`recipientID == actorID` チェック済み）
- [x] `GetNotifications(ctx, userID, cursor, limit) (*dto.NotificationListResponse, error)`
- [x] `MarkAllRead(ctx, userID) error`

## ハンドラー（`internal/handler/interaction_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [x] `POST /api/v1/posts/:id/like`（要認証）
- [x] `DELETE /api/v1/posts/:id/like`（要認証）
- [x] `POST /api/v1/posts/:id/repost`（要認証）
- [x] `DELETE /api/v1/posts/:id/repost`（要認証）
- [x] `POST /api/v1/posts/:id/bookmark`（要認証）
- [x] `DELETE /api/v1/posts/:id/bookmark`（要認証）
- [x] `GET /api/v1/bookmarks`（要認証）
- [x] `POST /api/v1/posts/:id/comments`（要認証）

## ハンドラー（`internal/handler/notification_handler.go`）

- [x] `GET /api/v1/notifications`（要認証）
- [x] `PUT /api/v1/notifications/read`（要認証）

## PostResponse の集計値対応

`post_service.go` の GetPost / GetTimeline 系で以下を含めて返すように更新:

- [x] `likes_count`
- [x] `comments_count`（reply_to == post.ID の件数）
- [x] `reposts_count`（repost_of == post.ID の件数）
- [x] `is_liked`（ログインユーザーがいいね済みか）
- [x] `is_bookmarked`
- [x] `is_reposted`

## 完了基準

- [x] いいね → カウントが増える → もう一度押すと取消（動作確認済み）
- [x] 自分の投稿以外をいいねすると通知が作成される（コード上確認、自分の投稿は作成されないことを確認）
- [x] リポストで `posts` テーブルに repost_of 付きレコードが作成される
- [x] コメントで `posts` テーブルに reply_to 付きレコードが作成される（動作確認済み）
- [x] ブックマーク一覧が cursor ページネーションで取得できる（API 自体は完成。対応するフロントエンド一覧画面は `PHASE11_frontend_profile.md` で未着手）

## 備考

- コメント削除後、詳細ページ上の親投稿のコメント数バッジがその場では減算されず、ページ再読み込みで正しい値になる（フロントエンドのキャッシュ更新の軽微な抜け、`PHASE10_frontend_timeline.md` 参照）。
