# PHASE 04 — バックエンド ユーザー・フォロー機能

> 目標: プロフィールの取得・更新と、フォロー/フォロワー機能を実装する。Firebase Storage への画像アップロードも含む。

---

## DTO（`internal/dto/user.go`）

- [x] `UserResponse`（id, handle, display_name, avatar_url, banner_url, bio, location, website_url, birthday, theme, followers_count, following_count, is_following）
- [x] `UpdateProfileRequest`（display_name, bio, location, website_url, birthday）
- [x] `FollowListResponse`（`UserListResponse` という名前で実装、users, next_cursor, has_more）

## リポジトリ（`internal/repository/user_repository.go` に追加）

- [x] `FindByHandle(ctx, handle) (*model.User, error)`
- [x] `Update(ctx, user) error`
- [ ] `UpdateAvatar(ctx, userID, url) error` — 専用メソッドはなく、汎用 `Update` で代用
- [ ] `UpdateBanner(ctx, userID, url) error` — 専用メソッドはなく、汎用 `Update` で代用
- [x] `CountFollowers(ctx, userID) (int64, error)`（`follow_repository.go` 側に実装）
- [x] `CountFollowing(ctx, userID) (int64, error)`（`follow_repository.go` 側に実装）
- [x] `IsFollowing(ctx, followerID, followingID) (bool, error)`（`Exists` という名前で `follow_repository.go` に実装）

## リポジトリ（`internal/repository/follow_repository.go`）

- [x] `FollowRepository` インターフェース定義
- [x] `Create(ctx, follow) error`
- [x] `Delete(ctx, followerID, followingID) error`
- [x] `GetFollowers(ctx, userID, cursor, limit) ([]model.User, string, error)`
- [x] `GetFollowing(ctx, userID, cursor, limit) ([]model.User, string, error)`

## Firebase Storage アップロードサービス（`internal/service/storage_service.go`）

- [x] `UploadImage(ctx, file, path) (url string, error)`
  - Firebase Storage SDK でアップロード
  - アップロード後に公開 URL を取得して返す
- [ ] `DeleteFile(ctx, url) error`（古い画像の削除）— 未実装。アバター/バナー更新時に旧ファイルは削除されず残る
- [x] 対応形式: JPEG / PNG / WebP
- [x] サイズ制限: 1 枚 5MB 以内のバリデーション

## サービス（`internal/service/user_service.go`）

- [x] `GetProfile(ctx, handle, viewerID) (*dto.UserResponse, error)`
  - フォロワー数・フォロー数・is_following を含めて返す
- [x] `UpdateProfile(ctx, userID, req) (*dto.UserResponse, error)`
- [x] `UploadAvatar(ctx, userID, file) (url string, error)`（`UserHandler.UpdateAvatar` 経由で実装）
- [x] `UploadBanner(ctx, userID, file) (url string, error)`（`UserHandler.UpdateBanner` 経由で実装）
- [x] `Follow(ctx, followerID, handle) error`
  - 自分自身のフォロー禁止チェック
  - 既にフォロー済みなら 409
- [x] `Unfollow(ctx, followerID, handle) error`
- [x] `GetFollowers(ctx, handle, cursor, limit) (*dto.FollowListResponse, error)`
- [x] `GetFollowing(ctx, handle, cursor, limit) (*dto.FollowListResponse, error)`

## ハンドラー（`internal/handler/user_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [x] `GET /api/v1/users/me`（要認証）— 自分のプロフィール
- [x] `PUT /api/v1/users/me`（要認証）— プロフィールテキスト更新
- [x] `PUT /api/v1/users/me/avatar`（要認証、multipart）— アバター画像更新
- [x] `PUT /api/v1/users/me/banner`（要認証、multipart）— バナー画像更新
- [x] `GET /api/v1/users/:handle` — 指定ユーザーのプロフィール
- [x] `POST /api/v1/users/:handle/follow`（要認証）
- [x] `DELETE /api/v1/users/:handle/follow`（要認証）
- [x] `GET /api/v1/users/:handle/followers`
- [x] `GET /api/v1/users/:handle/following`

## テーマ更新

- [x] `PUT /api/v1/users/me/theme` — テーマ設定を更新（`light` | `dark`）

## 完了基準

- [x] `GET /api/v1/users/:handle` でプロフィールが返る
- [x] フォロー → フォロワー数が増える → アンフォローで減る
- [x] アバター・バナー画像を Firebase Storage にアップロードして URL が DB に保存される
- [x] Swagger UI で全ユーザーエンドポイントが表示される

## 備考

バックエンド API 自体は完成しているが、対応するフロントエンド画面（プロフィールページ・フォロー一覧・アバター/バナー編集 UI）は `PHASE11_frontend_profile.md` 側が未着手のため、まだ画面から確認できない。
