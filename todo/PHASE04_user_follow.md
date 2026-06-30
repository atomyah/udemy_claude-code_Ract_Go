# PHASE 04 — バックエンド ユーザー・フォロー機能

> 目標: プロフィールの取得・更新と、フォロー/フォロワー機能を実装する。Firebase Storage への画像アップロードも含む。

---

## DTO（`internal/dto/user.go`）

- [ ] `UserResponse`（id, handle, display_name, avatar_url, banner_url, bio, location, website_url, birthday, theme, followers_count, following_count, is_following）
- [ ] `UpdateProfileRequest`（display_name, bio, location, website_url, birthday）
- [ ] `FollowListResponse`（users: []UserResponse, next_cursor, has_more）

## リポジトリ（`internal/repository/user_repository.go` に追加）

- [ ] `FindByHandle(ctx, handle) (*model.User, error)`
- [ ] `Update(ctx, user) error`
- [ ] `UpdateAvatar(ctx, userID, url) error`
- [ ] `UpdateBanner(ctx, userID, url) error`
- [ ] `CountFollowers(ctx, userID) (int64, error)`
- [ ] `CountFollowing(ctx, userID) (int64, error)`
- [ ] `IsFollowing(ctx, followerID, followingID) (bool, error)`

## リポジトリ（`internal/repository/follow_repository.go`）

- [ ] `FollowRepository` インターフェース定義
- [ ] `Create(ctx, follow) error`
- [ ] `Delete(ctx, followerID, followingID) error`
- [ ] `GetFollowers(ctx, userID, cursor, limit) ([]model.User, string, error)`
- [ ] `GetFollowing(ctx, userID, cursor, limit) ([]model.User, string, error)`
- [ ] `GetFollowingIDs(ctx, userID) ([]uuid.UUID, error)` — タイムライン用

## Firebase Storage アップロードサービス（`internal/service/storage_service.go`）

- [ ] `UploadImage(ctx, file, path) (url string, error)`
  - Firebase Storage SDK でアップロード
  - アップロード後に公開 URL を取得して返す
- [ ] `DeleteFile(ctx, url) error`（古い画像の削除）
- [ ] 対応形式: JPEG / PNG / WebP
- [ ] サイズ制限: 1 枚 5MB 以内のバリデーション

## サービス（`internal/service/user_service.go`）

- [ ] `GetProfile(ctx, handle, viewerID) (*dto.UserResponse, error)`
  - フォロワー数・フォロー数・is_following を含めて返す
- [ ] `UpdateProfile(ctx, userID, req) (*dto.UserResponse, error)`
- [ ] `UploadAvatar(ctx, userID, file) (url string, error)`
- [ ] `UploadBanner(ctx, userID, file) (url string, error)`
- [ ] `Follow(ctx, followerID, handle) error`
  - 自分自身のフォロー禁止チェック
  - 既にフォロー済みなら 409
- [ ] `Unfollow(ctx, followerID, handle) error`
- [ ] `GetFollowers(ctx, handle, cursor, limit) (*dto.FollowListResponse, error)`
- [ ] `GetFollowing(ctx, handle, cursor, limit) (*dto.FollowListResponse, error)`

## ハンドラー（`internal/handler/user_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [ ] `GET /api/v1/users/me`（要認証）— 自分のプロフィール
- [ ] `PUT /api/v1/users/me`（要認証）— プロフィールテキスト更新
- [ ] `PUT /api/v1/users/me/avatar`（要認証、multipart）— アバター画像更新
- [ ] `PUT /api/v1/users/me/banner`（要認証、multipart）— バナー画像更新
- [ ] `GET /api/v1/users/:handle` — 指定ユーザーのプロフィール
- [ ] `POST /api/v1/users/:handle/follow`（要認証）
- [ ] `DELETE /api/v1/users/:handle/follow`（要認証）
- [ ] `GET /api/v1/users/:handle/followers`
- [ ] `GET /api/v1/users/:handle/following`

## テーマ更新

- [ ] `PUT /api/v1/users/me/theme` — テーマ設定を更新（`light` | `dark`）

## 完了基準

- [ ] `GET /api/v1/users/:handle` でプロフィールが返る
- [ ] フォロー → フォロワー数が増える → アンフォローで減る
- [ ] アバター・バナー画像を Firebase Storage にアップロードして URL が DB に保存される
- [ ] Swagger UI で全ユーザーエンドポイントが表示される
