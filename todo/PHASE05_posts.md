# PHASE 05 — バックエンド 投稿・メディア・タイムライン

> 目標: 投稿の CRUD・メディアアップロード・ハッシュタグ処理・タイムライン（cursor ベース）を実装する。

---

## DTO（`internal/dto/post.go`）

- [ ] `PostResponse`（id, user, content, media[], likes_count, comments_count, reposts_count, is_liked, is_bookmarked, is_reposted, is_edited, is_deleted, repost_of, reply_to, created_at）
- [ ] `CreatePostRequest`（content: form, media[]: multipart files）
- [ ] `UpdatePostRequest`（content）
- [ ] `PostListResponse`（posts: []PostResponse, next_cursor, has_more）
- [ ] `MediaResponse`（id, url, type, sort_order）
- [ ] `UserInPostResponse`（id, handle, display_name, avatar_url）— 投稿に埋め込む軽量ユーザー情報

## リポジトリ（`internal/repository/post_repository.go`）

- [ ] `PostRepository` インターフェース定義
- [ ] `Create(ctx, post) (*model.Post, error)`
- [ ] `FindByID(ctx, id) (*model.Post, error)`— is_deleted=false のみ
- [ ] `Update(ctx, post) error`
- [ ] `SoftDelete(ctx, id, userID) error`— is_deleted=true に更新
- [ ] `GetTimeline(ctx, userIDs []uuid.UUID, cursor, limit) ([]model.Post, string, error)`— ホームタイムライン（cursor: created_at ベース）
- [ ] `GetExplore(ctx, cursor, limit) ([]model.Post, string, error)`— 全体タイムライン
- [ ] `GetByUser(ctx, userID, cursor, limit) ([]model.Post, string, error)`— ユーザー投稿一覧
- [ ] `GetComments(ctx, postID, cursor, limit) ([]model.Post, string, error)`

## リポジトリ（`internal/repository/media_repository.go`）

- [ ] `CreateBulk(ctx, media []model.Media) error`
- [ ] `FindByPostID(ctx, postID) ([]model.Media, error)`

## リポジトリ（`internal/repository/hashtag_repository.go`）

- [ ] `FindOrCreate(ctx, name) (*model.Hashtag, error)`
- [ ] `AttachToPost(ctx, postID, hashtagIDs) error`
- [ ] `GetPostsByHashtag(ctx, name, cursor, limit) ([]model.Post, string, error)`

## メディアアップロードサービス（`internal/service/storage_service.go` に追加）

- [ ] `UploadPostMedia(ctx, file, postID, index) (url, type string, error)`
  - 画像: JPEG / PNG / WebP、最大 5MB
  - 動画: MP4 / MOV、最大 100MB
  - ファイル種別の自動判定（Content-Type）
- [ ] 1 投稿あたり画像 4 枚 or 動画 1 本のバリデーション

## ハッシュタグ抽出ユーティリティ（`internal/utils/hashtag.go`）

- [ ] `ExtractHashtags(content string) []string`— `#tag` をテキストから抽出（正規表現）

## サービス（`internal/service/post_service.go`）

- [ ] `CreatePost(ctx, userID, req, files) (*dto.PostResponse, error)`
  - ハッシュタグを抽出して `hashtags`・`post_hashtags` に保存
  - メディアを Firebase Storage にアップロードして `media` に保存
- [ ] `GetPost(ctx, id, viewerID) (*dto.PostResponse, error)`
- [ ] `UpdatePost(ctx, id, userID, req) (*dto.PostResponse, error)`— 本人のみ、is_edited=true に更新
- [ ] `DeletePost(ctx, id, userID) error`— 本人のみ、is_deleted=true に更新
- [ ] `GetHomeTimeline(ctx, userID, cursor, limit) (*dto.PostListResponse, error)`
  - follow_repository でフォロー中ユーザー ID を取得してから投稿取得
- [ ] `GetExploreTimeline(ctx, cursor, limit) (*dto.PostListResponse, error)`
- [ ] `GetComments(ctx, postID, cursor, limit) (*dto.PostListResponse, error)`

## ハンドラー（`internal/handler/post_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [ ] `GET /api/v1/posts`（探索タイムライン）
- [ ] `GET /api/v1/posts/home`（要認証、ホームタイムライン）
- [ ] `POST /api/v1/posts`（要認証、multipart/form-data）
- [ ] `GET /api/v1/posts/:id`
- [ ] `PUT /api/v1/posts/:id`（要認証、本人のみ）
- [ ] `DELETE /api/v1/posts/:id`（要認証、本人のみ）
- [ ] `GET /api/v1/posts/:id/comments`

## 権限チェック

- [ ] 投稿編集・削除時に `post.UserID == requestUserID` を確認。違反時は 403 を返す。

## 完了基準

- [ ] 投稿作成 → メディアが Firebase Storage にアップロードされる
- [ ] ハッシュタグが自動抽出されて DB に保存される
- [ ] ホームタイムライン（フォロー中のみ）と探索タイムライン（全体）が cursor で取得できる
- [ ] `limit=20` で 20 件、`cursor` 指定で次のページが取得できる
- [ ] 本人以外が編集・削除しようとすると 403 が返る
