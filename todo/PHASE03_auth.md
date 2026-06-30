# PHASE 03 — バックエンド 認証機能

> 目標: メール/パスワード登録・ログインと Google OAuth を実装し、JWT で保護されたエンドポイントを作れる状態にする。

---

## DTO 定義（`internal/dto/auth.go`）

- [ ] `RegisterRequest`（email, password, handle, display_name）
- [ ] `LoginRequest`（email, password）
- [ ] `GoogleAuthRequest`（id_token: Firebase から取得したトークン）
- [ ] `AuthResponse`（access_token, user_id, handle, display_name, avatar_url）
- [ ] `ErrorResponse`（code, message）

## リポジトリ（`internal/repository/user_repository.go`）

- [ ] `UserRepository` インターフェース定義
- [ ] `FindByEmail(ctx, email) (*model.User, error)`
- [ ] `FindByHandle(ctx, handle) (*model.User, error)`
- [ ] `Create(ctx, user) (*model.User, error)`
- [ ] `FindByID(ctx, id) (*model.User, error)`

## サービス（`internal/service/auth_service.go`）

- [ ] パスワードハッシュ化（`bcrypt`）
- [ ] `Register(ctx, req) (*dto.AuthResponse, error)`
  - メール重複チェック
  - handle 重複チェック
  - パスワードハッシュ化して保存
- [ ] `Login(ctx, req) (*dto.AuthResponse, error)`
  - メール検索 → パスワード照合
- [ ] `LoginWithGoogle(ctx, idToken) (*dto.AuthResponse, error)`
  - Firebase Admin SDK で idToken を検証
  - 初回なら users に INSERT、2回目以降は取得のみ
- [ ] `GenerateTokenPair(userID) (accessToken, refreshToken, error)`
  - Access Token: JWT、有効期限 15 分
  - Refresh Token: JWT、有効期限 7 日
- [ ] `RefreshAccessToken(ctx, refreshToken) (accessToken, error)`

## ハンドラー（`internal/handler/auth_handler.go`）

swaggo コメントを全エンドポイントに付ける。

- [ ] `POST /api/v1/auth/register`
  - バリデーション（email 形式、パスワード 8 文字以上、handle 形式）
  - 成功: 201 + AuthResponse
- [ ] `POST /api/v1/auth/login`
  - 成功: 200 + AuthResponse + Refresh Token を HttpOnly Cookie にセット
- [ ] `POST /api/v1/auth/logout`
  - Refresh Token Cookie を削除
- [ ] `POST /api/v1/auth/refresh`
  - Cookie から Refresh Token を取得 → Access Token を再発行
- [ ] `POST /api/v1/auth/google`
  - Firebase ID Token を受け取り → 検証 → JWT を返す

## JWT ミドルウェア（`internal/middleware/auth.go`）

- [ ] `Authorization: Bearer <token>` ヘッダーを検証
- [ ] 検証済みユーザー ID を `echo.Context` にセット（キー: `"userID"`）
- [ ] 期限切れ → 401 `EXPIRED_TOKEN`
- [ ] 無効トークン → 401 `INVALID_TOKEN`

## Firebase Admin SDK 初期化

- [ ] `internal/config/firebase.go` で Firebase App を初期化
  - 環境変数 `FIREBASE_CREDENTIALS_JSON` からサービスアカウントを読み込む
- [ ] Auth クライアントと Storage クライアントを返すヘルパーを実装

## ルーティング（`cmd/server/main.go`）

- [ ] 認証不要グループ（`/api/v1/auth/*`）
- [ ] 認証必要グループ（`/api/v1/*`）に JWT ミドルウェアを適用

## 完了基準

- [ ] `POST /api/v1/auth/register` でユーザーが作成される
- [ ] `POST /api/v1/auth/login` で JWT が返る
- [ ] JWT を使って `GET /health`（または仮の保護済みエンドポイント）にアクセスできる
- [ ] 期限切れ・無効 JWT で 401 が返る
- [ ] Swagger UI で全認証エンドポイントが表示される
