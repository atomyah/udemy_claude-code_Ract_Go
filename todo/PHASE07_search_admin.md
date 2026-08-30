# PHASE 07 — バックエンド 検索・管理者機能

> 目標: 検索 API と管理者専用エンドポイントを実装し、バックエンド全体を完成させる。

---

## 検索

### DTO（`internal/dto/search.go`）

- [ ] `UserSearchResponse` — 専用ファイル/型はなく、既存の `dto.UserListResponse` を再利用（実質的に等価）
- [ ] `PostSearchResponse` — 専用ファイル/型はなく、既存の `dto.PostListResponse` を再利用（実質的に等価）

### リポジトリ（`internal/repository/search_repository.go`）

- [x] `SearchUsers(ctx, query, cursor, limit) ([]model.User, string, error)`
  - handle・display_name の `ILIKE '%q%'` 検索、`is_suspended = false` 条件込み
- [x] `SearchPosts(ctx, query, cursor, limit) ([]model.Post, string, error)`
  - content の `ILIKE '%q%'` 検索
  - is_deleted=false・is_suspended=false のユーザーのみ

### サービス（`internal/service/search_service.go`）

- [x] `SearchUsers(ctx, query, cursor, limit) (*dto.UserListResponse, error)`
- [x] `SearchPosts(ctx, query, cursor, limit) (*dto.PostListResponse, error)`
- [x] `GetPostsByHashtag(ctx, tag, cursor, limit) (*dto.PostListResponse, error)`

### ハンドラー（`internal/handler/search_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [x] `GET /api/v1/search/users?q=&cursor=&limit=`
- [x] `GET /api/v1/search/posts?q=&cursor=&limit=`（動作確認済み: ExplorePage の検索バーから利用）
- [x] `GET /api/v1/search/hashtags/:tag?cursor=&limit=`

---

## 管理者機能

### 管理者ミドルウェア（`internal/middleware/admin.go`）

- [x] JWT 検証済みユーザーの `role` が `admin` か確認
- [x] 違反時は 403 `FORBIDDEN` を返す

### リポジトリ（既存 repository に追加）

- [x] `post_repository.AdminDelete(ctx, id) error` — **物理削除ではなく `SoftDelete` を呼ぶ実装**（`rules/database.md` の「物理削除禁止」方針に合わせた意図的な差分。仕様書の文言が古い）
- [x] `user_repository.Suspend(ctx, id) error`
- [x] `user_repository.Unsuspend(ctx, id) error`
- [ ] `user_repository.FindAll(ctx, cursor, limit) ([]model.User, string, error)` — 未実装。管理者用ユーザー一覧 API は存在しない

### サービス（`internal/service/admin_service.go`）

- [x] `ForceDeletePost(ctx, adminID, postID) error`
- [x] `SuspendUser(ctx, adminID, targetID) error`
- [x] `UnsuspendUser(ctx, adminID, targetID) error`

### ハンドラー（`internal/handler/admin_handler.go`）

- [x] `DELETE /api/v1/admin/posts/:id`（要 admin ロール）
- [x] `PUT /api/v1/admin/users/:id/suspend`（要 admin ロール）
- [x] `DELETE /api/v1/admin/users/:id/suspend`（要 admin ロール）

---

## 停止ユーザーの除外処理

バックエンド全体で停止ユーザーの投稿を除外するよう既存クエリを更新:

- [x] タイムライン取得: `JOIN users ... WHERE users.is_suspended = false`（`post_repository.go` の `basePostQuery`）
- [x] ユーザー検索: `WHERE is_suspended = false`
- [x] 停止ユーザー本人がログインしようとした場合: 401 `ACCOUNT_SUSPENDED`（`auth_service.go` の `ErrAccountSuspended` で実装確認済み）

---

## OpenAPI 最終整備

- [x] `swag init` を実行して `docs/swagger.json` を最新化
- [x] 全エンドポイントに swaggo コメントが付いていることを確認
- [x] `docs/swagger.json` をフロントエンドの型生成用にエクスポート（`docs/openapi.yaml` も生成済み）

---

## バックエンド全体の完了基準

- [x] 全 API エンドポイントが Swagger UI に表示される
- [x] `GET /api/v1/search/users?q=test` でユーザーが返る
- [x] `GET /api/v1/search/hashtags/golang` でハッシュタグ付き投稿が返る
- [x] admin ロールのユーザーのみ管理者エンドポイントにアクセスできる
- [x] 一般ユーザーが管理者エンドポイントにアクセスすると 403 が返る
- [x] 停止ユーザーの投稿がタイムラインに表示されない

## 備考

- 管理者用のユーザー一覧取得 API（`FindAll`）は未実装。フロントエンドの `AdminPage` 自体もプレースホルダーのため、対応するUIもまだない（`PHASE11_frontend_profile.md` 参照）。
