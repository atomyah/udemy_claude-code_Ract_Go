# PHASE 05 — バックエンド 投稿・メディア・タイムライン

> 目標: 投稿の CRUD・メディアアップロード・ハッシュタグ処理・タイムライン（cursor ベース）を実装する。

---

## DTO（`internal/dto/post.go`）

- [x] `PostResponse`（id, user, content, media[], likes_count, comments_count, reposts_count, is_liked, is_bookmarked, is_reposted, is_edited, is_deleted, repost_of, reply_to, created_at）
- [x] `CreatePostRequest`（content: form, media[]: multipart files）
- [x] `UpdatePostRequest`（content）
- [x] `PostListResponse`（posts: []PostResponse, next_cursor, has_more）
- [x] `MediaResponse`（id, url, type, sort_order）
- [x] `UserInPostResponse`（`UserInPost` という名前で実装）— 投稿に埋め込む軽量ユーザー情報

## リポジトリ（`internal/repository/post_repository.go`）

- [x] `PostRepository` インターフェース定義
- [x] `Create(ctx, post) (*model.Post, error)`
- [x] `FindByID(ctx, id) (*model.Post, error)`— is_deleted=false のみ
- [x] `Update(ctx, post) error`
- [x] `SoftDelete(ctx, id, userID) error`— is_deleted=true に更新
- [x] `GetExplore(ctx, cursor, limit) ([]model.Post, string, error)`— 全体タイムライン（ホーム・探索の両方から利用）
- [x] `GetByUser(ctx, userID, cursor, limit) ([]model.Post, string, error)`— ユーザー投稿一覧
- [x] `GetComments(ctx, postID, cursor, limit) ([]model.Post, string, error)`

## リポジトリ（`internal/repository/media_repository.go`）

- [x] `CreateBulk(ctx, media []model.Media) error`
- [x] `FindByPostID(ctx, postID) ([]model.Media, error)`

## リポジトリ（`internal/repository/hashtag_repository.go`）

- [x] `FindOrCreate(ctx, name) (*model.Hashtag, error)`
- [x] `AttachToPost(ctx, postID, hashtagIDs) error`
- [x] `GetPostsByHashtag(ctx, name, cursor, limit) ([]model.Post, string, error)`（`GetPostIDsByHashtag` という名前で実装）

## メディアアップロードサービス（`internal/service/storage_service.go` に追加）

- [x] `UploadPostMedia(ctx, file, postID, index) (url, type string, error)`
  - 画像: JPEG / PNG / WebP、最大 5MB
  - 動画: MP4 / MOV、最大 100MB
  - ファイル種別の自動判定（Content-Type）
- [x] 1 投稿あたり画像 4 枚 or 動画 1 本のバリデーション（`validateMediaFiles`）

## ハッシュタグ抽出ユーティリティ（`internal/utils/hashtag.go`）

- [x] `ExtractHashtags(content string) []string`— `#tag` をテキストから抽出（正規表現）。動作確認でも `#playwright` が正しく抽出・リンク化されることを確認済み

## サービス（`internal/service/post_service.go`）

- [x] `CreatePost(ctx, userID, req, files) (*dto.PostResponse, error)`
  - ハッシュタグを抽出して `hashtags`・`post_hashtags` に保存
  - メディアを Firebase Storage にアップロードして `media` に保存
- [x] `GetPost(ctx, id, viewerID) (*dto.PostResponse, error)`
- [x] `UpdatePost(ctx, id, userID, req) (*dto.PostResponse, error)`— 本人のみ、is_edited=true に更新
- [x] `DeletePost(ctx, id, userID) error`— 本人のみ、is_deleted=true に更新
- [x] `GetHomeTimeline(ctx, userID, cursor, limit) (*dto.PostListResponse, error)`
  - 全ユーザーの投稿を新着順で返す（`GetExplore` を利用。要認証な点のみ探索と異なる）
- [x] `GetExploreTimeline(ctx, cursor, limit) (*dto.PostListResponse, error)`
- [x] `GetComments(ctx, postID, cursor, limit) (*dto.PostListResponse, error)`

## ハンドラー（`internal/handler/post_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [x] `GET /api/v1/posts`（探索タイムライン）
- [x] `GET /api/v1/posts/home`（要認証、ホームタイムライン）
- [x] `POST /api/v1/posts`（要認証、multipart/form-data）
- [x] `GET /api/v1/posts/:id`
- [x] `PUT /api/v1/posts/:id`（要認証、本人のみ）
- [x] `DELETE /api/v1/posts/:id`（要認証、本人のみ）
- [x] `GET /api/v1/posts/:id/comments`

## 権限チェック

- [x] 投稿編集・削除時に `post.UserID == requestUserID` を確認。違反時は 403（`ErrForbidden`）を返す。

## 完了基準

- [x] 投稿作成 → メディアが Firebase Storage にアップロードされる
- [x] ハッシュタグが自動抽出されて DB に保存される（動作確認済み）
- [x] ホームタイムライン（全体、要認証）と探索タイムライン（全体、認証任意）が cursor で取得できる（動作確認済み）
- [x] `limit=20` で 20 件、`cursor` 指定で次のページが取得できる
- [x] 本人以外が編集・削除しようとすると 403 が返る

## 備考（フロントエンド動作確認で判明した既知の不具合）

投稿詳細ページ（`PostDetailPage`）で表示中の投稿自身を削除すると、サーバー側は正しく削除される（再訪問すると404扱い）が、削除直後の画面はキャッシュが更新されず古い内容が残る。フロントエンド側 `PostCard.tsx` の削除処理に `navigate('/')` 等のリダイレクトが必要。詳細は `PHASE10_frontend_timeline.md` の備考を参照。
