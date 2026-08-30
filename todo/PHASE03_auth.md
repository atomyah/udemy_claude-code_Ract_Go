# PHASE 03 — バックエンド 認証機能

> 目標: メール/パスワード登録・ログインと Google OAuth を実装し、JWT で保護されたエンドポイントを作れる状態にする。

---

## DTO 定義（`internal/dto/auth.go`）

- [x] `RegisterRequest`（email, password, handle, display_name）
- [x] `LoginRequest`（email, password）
- [x] `GoogleAuthRequest`（`GoogleLoginRequest` という名前で実装、id_token）
- [x] `AuthResponse`（access_token, user_id, handle, display_name, avatar_url）
- [x] `ErrorResponse`（code, message）

## リポジトリ（`internal/repository/user_repository.go`）

- [x] `UserRepository` インターフェース定義
- [x] `FindByEmail(ctx, email) (*model.User, error)`
- [x] `FindByHandle(ctx, handle) (*model.User, error)`
- [x] `Create(ctx, user) (*model.User, error)`
- [x] `FindByID(ctx, id) (*model.User, error)`

## サービス（`internal/service/auth_service.go`）

- [x] パスワードハッシュ化（`bcrypt`、コスト係数 12）
- [x] `Register(ctx, req) (*dto.AuthResponse, error)`
  - メール重複チェック
  - handle 重複チェック
  - パスワードハッシュ化して保存
- [x] `Login(ctx, req) (*dto.AuthResponse, error)`
  - メール検索 → パスワード照合
- [ ] `LoginWithGoogle(ctx, idToken) (*dto.AuthResponse, error)` — **未実装**。Firebase Admin SDK での idToken 検証は行われていない
- [x] `GenerateTokenPair(userID) (accessToken, refreshToken, error)`（`generateAccessToken` / `generateRefreshToken` として実装）
  - Access Token: JWT、有効期限 15 分
  - Refresh Token: JWT、有効期限 7 日
- [x] `RefreshAccessToken(ctx, refreshToken) (accessToken, error)`

## ハンドラー（`internal/handler/auth_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [x] `POST /api/v1/auth/register`
  - バリデーション（email 形式、パスワード 8 文字以上、handle 形式）
  - 成功: 201 + AuthResponse
- [x] `POST /api/v1/auth/login`
  - 成功: 200 + AuthResponse + Refresh Token を HttpOnly Cookie にセット
- [x] `POST /api/v1/auth/logout`
  - Refresh Token Cookie を削除
- [x] `POST /api/v1/auth/refresh`
  - Cookie から Refresh Token を取得 → Access Token を再発行（動作確認済み: 期限切れ時に自動リフレッシュされることをブラウザで確認）
- [ ] `POST /api/v1/auth/google` — **スタブのみ**。`501 NOT_IMPLEMENTED` を返すだけで Firebase ID Token の検証は未実装

## JWT ミドルウェア（`internal/middleware/auth.go`）

- [x] `Authorization: Bearer <token>` ヘッダーを検証
- [x] 検証済みユーザー ID を `echo.Context` にセット（キー: `"userID"`）
- [x] 期限切れ → 401 `EXPIRED_TOKEN`
- [x] 無効トークン → 401 `INVALID_TOKEN`

## Firebase Admin SDK 初期化

- [ ] `internal/config/firebase.go` で Firebase App を初期化 — 専用ファイルはなく、`internal/service/storage_service.go` 内で Firebase Storage 用の `firebase.App` を初期化（Storage 用途のみ）
- [ ] Auth クライアントと Storage クライアントを返すヘルパーを実装 — Storage クライアントのみ。Auth（IDトークン検証）クライアントは未実装

## ルーティング（`cmd/server/main.go`）

- [x] 認証不要グループ（`/api/v1/auth/*`）
- [x] 認証必要グループ（`/api/v1/*`）に JWT ミドルウェアを適用（`jwtMiddleware` / 一部 `optionalAuthMiddleware` / `adminMiddleware`）

## 完了基準

- [x] `POST /api/v1/auth/register` でユーザーが作成される
- [x] `POST /api/v1/auth/login` で JWT が返る
- [x] JWT を使って保護済みエンドポイント（`/users/me` 等）にアクセスできる
- [x] 期限切れ・無効 JWT で 401 が返る
- [x] Swagger UI で全認証エンドポイントが表示される

## 備考

- Google OAuth ログインはバックエンド・フロントエンドとも未実装（スコープ外として保留中）。対応する場合は `LoginWithGoogle` サービス実装 + Firebase Admin SDK の Auth クライアント追加が必要。
