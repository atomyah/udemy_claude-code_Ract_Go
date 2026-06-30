# PHASE 07 — バックエンド 検索・管理者機能

> 目標: 検索 API と管理者専用エンドポイントを実装し、バックエンド全体を完成させる。

---

## 検索

### DTO（`internal/dto/search.go`）

- [ ] `UserSearchResponse`（users: []UserResponse, next_cursor, has_more）
- [ ] `PostSearchResponse`（posts: []PostResponse, next_cursor, has_more）

### リポジトリ（`internal/repository/search_repository.go`）

- [ ] `SearchUsers(ctx, query, cursor, limit) ([]model.User, string, error)`
  - handle・display_name の前方一致（`ILIKE '%q%'`）
- [ ] `SearchPosts(ctx, query, cursor, limit) ([]model.Post, string, error)`
  - content の全文検索（PostgreSQL `ILIKE '%q%'` or `tsvector`）
  - is_deleted=false・is_suspended=false のユーザーのみ

### サービス（`internal/service/search_service.go`）

- [ ] `SearchUsers(ctx, query, cursor, limit) (*dto.UserSearchResponse, error)`
- [ ] `SearchPosts(ctx, query, cursor, limit) (*dto.PostSearchResponse, error)`
- [ ] `GetPostsByHashtag(ctx, tag, cursor, limit) (*dto.PostSearchResponse, error)`

### ハンドラー（`internal/handler/search_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [ ] `GET /api/v1/search/users?q=&cursor=&limit=`
- [ ] `GET /api/v1/search/posts?q=&cursor=&limit=`
- [ ] `GET /api/v1/search/hashtags/:tag?cursor=&limit=`

---

## 管理者機能

### 管理者ミドルウェア（`internal/middleware/admin.go`）

- [ ] JWT 検証済みユーザーの `role` が `admin` か確認
- [ ] 違反時は 403 `FORBIDDEN` を返す

### リポジトリ（既存 repository に追加）

- [ ] `post_repository.AdminDelete(ctx, id) error`— 物理削除（管理者による強制削除）
- [ ] `user_repository.Suspend(ctx, id) error`
- [ ] `user_repository.Unsuspend(ctx, id) error`
- [ ] `user_repository.FindAll(ctx, cursor, limit) ([]model.User, string, error)`— 管理者用ユーザー一覧（停止ユーザー含む）

### サービス（`internal/service/admin_service.go`）

- [ ] `ForceDeletePost(ctx, adminID, postID) error`
- [ ] `SuspendUser(ctx, adminID, targetID) error`
- [ ] `UnsuspendUser(ctx, adminID, targetID) error`

### ハンドラー（`internal/handler/admin_handler.go`）

- [ ] `DELETE /api/v1/admin/posts/:id`（要 admin ロール）
- [ ] `PUT /api/v1/admin/users/:id/suspend`（要 admin ロール）
- [ ] `DELETE /api/v1/admin/users/:id/suspend`（要 admin ロール）

---

## 停止ユーザーの除外処理

バックエンド全体で停止ユーザーの投稿を除外するよう既存クエリを更新:

- [ ] タイムライン取得: `JOIN users ... WHERE users.is_suspended = false`
- [ ] ユーザー検索: `WHERE is_suspended = false`
- [ ] 停止ユーザー本人がログインしようとした場合: 401 `ACCOUNT_SUSPENDED`

---

## OpenAPI 最終整備

- [ ] `swag init` を実行して `docs/swagger.json` を最新化
- [ ] 全エンドポイントに swaggo コメントが付いていることを確認
- [ ] `docs/swagger.json` をフロントエンドの型生成用にエクスポート

---

## バックエンド全体の完了基準

- [ ] 全 API エンドポイントが Swagger UI に表示される
- [ ] `GET /api/v1/search/users?q=test` でユーザーが返る
- [ ] `GET /api/v1/search/hashtags/golang` でハッシュタグ付き投稿が返る
- [ ] admin ロールのユーザーのみ管理者エンドポイントにアクセスできる
- [ ] 一般ユーザーが管理者エンドポイントにアクセスすると 403 が返る
- [ ] 停止ユーザーの投稿がタイムラインに表示されない
