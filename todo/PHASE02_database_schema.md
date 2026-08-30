# PHASE 02 — データベーススキーマ・マイグレーション

> 目標: 全テーブルのマイグレーションファイルを作成し、GORM モデルを定義する。

---

## golang-migrate セットアップ

- [ ] `go get github.com/golang-migrate/migrate/v4` — 未導入（go.mod に依存なし）
- [x] `migrations/` ディレクトリを作成
- [ ] マイグレーション CLI を使えるようにする（`migrate` コマンド or Go コードから実行）— 未実装
- [ ] `cmd/server/main.go` でアプリ起動時にマイグレーションを自動実行する仕組みを追加 — 未実装。実際は `internal/config/db.go` で開発時に `GORM AutoMigrate` を実行しているのみで、`migrations/*.sql` は現状アプリから実行されていない

## マイグレーションファイル作成（up / down のペア）

> 実際のファイル構成は仕様と統合が異なる（テーブルごとではなく関連テーブルをまとめている）が、全カラム・制約は網羅されている。

- [x] users — `001_create_users.up/down.sql`
- [x] posts + media — `002_create_posts_and_media.up/down.sql`（仕様では 002/003 に分割だが 1 ファイルに統合）
- [x] likes + bookmarks — `003_create_interactions.up/down.sql`（仕様では 004/005 に分割だが 1 ファイルに統合）
- [x] follows — `004_create_follows.up/down.sql`
  - UNIQUE(follower_id, following_id) / CHECK(follower_id != following_id) 含む
- [x] hashtags + post_hashtags — `005_create_hashtags.up/down.sql`（仕様では 007/008 に分割だが 1 ファイルに統合）
- [x] notifications — `006_create_notifications.up/down.sql`

## インデックス追加マイグレーション

> 独立した `010_add_indexes` ファイルではなく、各テーブルの作成マイグレーション内にインデックスを含める方式。項目自体はすべて存在する。

- [x] `idx_posts_user_id`
- [x] `idx_posts_created_at DESC`
- [x] `idx_posts_reply_to`（部分インデックス）
- [x] `idx_likes_post_id`
- [x] `idx_follows_follower_id`
- [x] `idx_follows_following_id`
- [x] `idx_notifications_user_id`
- [x] `idx_notifications_is_read`（`idx_notifications_unread` という名前で実装、部分インデックス）
- [x] `idx_users_handle`
- [x] `idx_hashtags_name`

## GORM モデル定義（`internal/model/`）

- [x] `user.go` — User モデル
- [x] `post.go` — Post モデル（関連: User, Media, Likes, Bookmarks）
- [x] `media.go` — Media モデル
- [x] `like.go` — Like モデル
- [x] `bookmark.go` — Bookmark モデル
- [x] `follow.go` — Follow モデル
- [x] `hashtag.go` — Hashtag モデル
- [x] `post_hashtag.go` — PostHashtag モデル（`hashtag.go` 内に同居）
- [x] `notification.go` — Notification モデル

## DB 接続

- [x] `internal/config/db.go` に GORM + PostgreSQL 接続処理を実装（仕様のファイル名は `database.go` だが同等）
- [ ] 接続リトライ処理（起動順序のズレ対策）— アプリ側には未実装。`docker-compose.yml` の `depends_on: condition: service_healthy` で代替
- [x] `main.go` で DB 接続 → Echo 起動の順で初期化（マイグレーションはアプリ内で明示実行されず `NewDB` 内の `AutoMigrate` に統合）

## 完了基準

- [x] `docker-compose up` 後に全テーブルが PostgreSQL に作成されている（`AutoMigrate` 経由）
- [ ] `migrate down` でロールバックして `migrate up` で再適用できる — golang-migrate 未導入のため未検証
- [ ] GORM の `db.AutoMigrate` は削除済み（マイグレーションファイルのみで管理）— **未達成。現状は開発環境で `AutoMigrate` を使用中**（`rules/database.md` 的には開発初期の許容範囲内だが、本番前に `migrations/*.sql` への一本化が必要）
