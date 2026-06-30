# PHASE 02 — データベーススキーマ・マイグレーション

> 目標: 全テーブルのマイグレーションファイルを作成し、GORM モデルを定義する。

---

## golang-migrate セットアップ

- [ ] `go get github.com/golang-migrate/migrate/v4`
- [ ] `migrations/` ディレクトリを作成
- [ ] マイグレーション CLI を使えるようにする（`migrate` コマンド or Go コードから実行）
- [ ] `cmd/server/main.go` でアプリ起動時にマイグレーションを自動実行する仕組みを追加

## マイグレーションファイル作成（up / down のペア）

- [ ] `001_create_users.up.sql` / `001_create_users.down.sql`
  - `id`, `email`, `password_hash`, `handle`, `display_name`, `avatar_url`, `banner_url`
  - `bio`, `location`, `website_url`, `birthday`
  - `theme`, `role`, `is_suspended`, `created_at`, `updated_at`
- [ ] `002_create_posts.up.sql` / `002_create_posts.down.sql`
  - `id`, `user_id`, `content`, `is_edited`, `is_deleted`
  - `repost_of`（自己参照 FK）, `reply_to`（自己参照 FK）
  - `created_at`, `updated_at`
- [ ] `003_create_media.up.sql` / `003_create_media.down.sql`
  - `id`, `post_id`, `url`, `type`, `sort_order`, `created_at`
- [ ] `004_create_likes.up.sql` / `004_create_likes.down.sql`
  - `id`, `user_id`, `post_id`, `created_at`
  - UNIQUE(user_id, post_id)
- [ ] `005_create_bookmarks.up.sql` / `005_create_bookmarks.down.sql`
  - `id`, `user_id`, `post_id`, `created_at`
  - UNIQUE(user_id, post_id)
- [ ] `006_create_follows.up.sql` / `006_create_follows.down.sql`
  - `id`, `follower_id`, `following_id`, `created_at`
  - UNIQUE(follower_id, following_id)
  - CHECK(follower_id != following_id)
- [ ] `007_create_hashtags.up.sql` / `007_create_hashtags.down.sql`
  - `id`, `name`, `created_at`
- [ ] `008_create_post_hashtags.up.sql` / `008_create_post_hashtags.down.sql`
  - `post_id`, `hashtag_id`（複合 PK）
- [ ] `009_create_notifications.up.sql` / `009_create_notifications.down.sql`
  - `id`, `user_id`, `actor_id`, `type`, `post_id`, `is_read`, `created_at`

## インデックス追加マイグレーション

- [ ] `010_add_indexes.up.sql` / `010_add_indexes.down.sql`
  - `idx_posts_user_id`
  - `idx_posts_created_at DESC`
  - `idx_posts_reply_to`（reply_to IS NOT NULL 部分インデックス）
  - `idx_likes_post_id`
  - `idx_follows_follower_id`
  - `idx_follows_following_id`
  - `idx_notifications_user_id`
  - `idx_notifications_is_read`（部分インデックス: is_read = false）
  - `idx_users_handle`
  - `idx_hashtags_name`

## GORM モデル定義（`internal/model/`）

- [ ] `user.go` — User モデル
- [ ] `post.go` — Post モデル（関連: User, Media, Likes, Bookmarks）
- [ ] `media.go` — Media モデル
- [ ] `like.go` — Like モデル
- [ ] `bookmark.go` — Bookmark モデル
- [ ] `follow.go` — Follow モデル
- [ ] `hashtag.go` — Hashtag モデル
- [ ] `post_hashtag.go` — PostHashtag モデル
- [ ] `notification.go` — Notification モデル

## DB 接続

- [ ] `internal/config/database.go` に GORM + PostgreSQL 接続処理を実装
- [ ] 接続リトライ処理（起動順序のズレ対策）
- [ ] `main.go` で DB 接続 → マイグレーション → Echo 起動の順で初期化

## 完了基準

- [ ] `docker-compose up` 後に全テーブルが PostgreSQL に作成されている
- [ ] `migrate down` でロールバックして `migrate up` で再適用できる
- [ ] GORM の `db.AutoMigrate` は削除済み（マイグレーションファイルのみで管理）
