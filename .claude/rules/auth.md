# 認証規約（JWT / Firebase Auth）

## トークン設計

| トークン | 形式 | 有効期限 | 保存場所 | 用途 |
|---------|------|---------|---------|------|
| Access Token | JWT（HS256） | 1 時間 | HttpOnly Cookie `access_token`（`Path=/api/v1`） | API の認証 |
| Refresh Token | 256 ビットの乱数（base64url） | 7 日 | HttpOnly Cookie `refresh_token`（`Path=/api/v1/auth`） | Access Token の再発行 |

- 実装: `backend/internal/authtoken/`（発行・検証）、`service/auth_service.go`（ローテーション・失効）、`handler/auth_handler.go`（Cookie の発行・削除）。
- Cookie 属性は `HttpOnly; Secure; SameSite=None`（フロントエンドと API が別オリジンのため）。
  - 本番・テストは常にこの属性。開発環境だけ `COOKIE_SECURE=false` で Secure を外せる（そのとき `SameSite=Lax`）。
  - Chrome / Edge / Firefox は `http://localhost` でも Secure Cookie を扱えるので、通常は設定不要。
- **トークンをレスポンスボディに含めない**。ログイン・登録・Google ログイン・リフレッシュは `Set-Cookie` で返し、ボディはユーザー情報のみ。

### Access Token

- ペイロードは **`user_id` と `exp` のみ**。ロールなどは含めない（ロールは DB を見る）。
- 署名鍵は環境変数 `JWT_SECRET`。検証時は `HS256` 以外のアルゴリズムと `exp` のないトークンを拒否する。
- ブラウザは Cookie で送る。Swagger UI・計測スクリプトなどブラウザ以外のクライアント向けに `Authorization: Bearer` ヘッダーも受け付ける（ヘッダーがあればそちらを優先）。

### Refresh Token

- DB の `refresh_tokens` テーブルに **HMAC-SHA256（鍵: `JWT_REFRESH_SECRET`）のハッシュだけ**を保存する。トークン本体は保存しない。
- `used_at`（ローテーション済み）・`revoked_at`（失効）で状態を管理する。`family_id` は 1 回のログインから続くローテーションの系列。
- `POST /api/v1/auth/refresh` で古いトークンを使用済みにし、新しい Access Token / Refresh Token を発行する（**ローテーション**。条件付き UPDATE で同時実行時の二重交換も防ぐ）。
- **使用済みトークンが再び使われたら漏洩とみなし、同じ系列をすべて失効させる**。
- ログアウト（`POST /auth/logout`）は系列を失効させ、Cookie を削除する。
- 全デバイスログアウト（`POST /auth/revoke-all`、要認証）はユーザーの全リフレッシュトークンを失効させる。他端末は Access Token の期限（最長 1 時間）でログアウトされる。
- 管理者がユーザーを停止したときも、そのユーザーの全リフレッシュトークンを失効させる。

---

## バックエンド実装規約

### ミドルウェア（`internal/middleware/auth.go`）

- 保護ルートには必ず JWT 検証ミドルウェアを適用する。
- 検証済みユーザー ID は `c.Set("userID", userID)` で Context に保存する。
- ハンドラーから取り出す際は型アサーション後に nil チェックをする。

```go
userID, ok := c.Get("userID").(uuid.UUID)
if !ok {
    return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
}
```

- 管理者ルートは `AdminOnly(userRepo)` で **DB のロール**を確認する（トークンにロールは入っていない）。

### CSRF 対策

- Cookie が `SameSite=None` のため、状態を変更するリクエスト（POST / PUT / PATCH / DELETE）は `OriginCheck` ミドルウェアで `Origin` ヘッダーを検証する。
- `CORS_ORIGINS` のオリジンと API 自身のオリジン以外は 403（`INVALID_ORIGIN`）。`Origin` がないリクエストはブラウザ以外のクライアントとして通す。

### エラーコード

| 状況 | HTTP | code |
|------|------|------|
| トークンなし | 401 | `MISSING_TOKEN` |
| トークン形式不正・失効済み・再利用 | 401 | `INVALID_TOKEN` |
| トークン期限切れ | 401 | `EXPIRED_TOKEN` |
| アカウント停止 | 401 | `ACCOUNT_SUSPENDED` |
| 権限不足 | 403 | `FORBIDDEN` |
| 許可されていない Origin | 403 | `INVALID_ORIGIN` |

### パスワード

- `bcrypt` でハッシュ化する（コスト係数: 12）。
- 平文パスワードをログに出力しない。
- 最小 8 文字、最大 72 文字（bcrypt の上限）。

---

## Firebase Auth（Google OAuth）

- フロントエンドは `signInWithPopup` で ID Token を取得する。
- ID Token はバックエンドの `POST /api/v1/auth/google` に送り、バックエンドで Firebase Admin SDK を使って検証する。検証後は通常のログインと同じく Cookie でトークンを返す。
- フロントエンドから Firebase Storage や Firestore に直接アクセスしない（Storage のみ許可、Admin SDK 経由）。
- サービスアカウントの JSON は環境変数 `FIREBASE_CREDENTIALS_JSON` で管理し、ファイルとしてコミットしない。

---

## フロントエンド実装規約

### トークン管理

- フロントエンドはトークンを**読まない・保存しない・送らない**。Cookie はブラウザが自動で送る。
- Axios インスタンス（`src/api/client.ts`）は `withCredentials: true`。`Authorization` ヘッダーは付けない。
- ログイン状態は起動時の `GET /users/me` の成否で判定する。ロール（管理者画面の表示判定）も `/users/me` の `role` を使う。
- 401 を受け取ったら `POST /auth/refresh` を呼び、元のリクエストを再送する。
  - 同時に複数の 401 が返ってもリフレッシュは 1 回だけ送る（ローテーションのため 2 回目は再利用扱いになる）。
  - 認証 API（login / register / refresh / google / logout）の 401 ではリフレッシュしない。
- リフレッシュも失敗したら、ログイン中だった場合はログアウト状態にしてログイン画面へ遷移する。

### ログアウト時の処理

1. `POST /api/v1/auth/logout` を呼ぶ（バックエンドがリフレッシュトークンを失効させ、Cookie を削除する）
2. TanStack Query のキャッシュを全クリア（`queryClient.clear()`）
3. `/login` にリダイレクト

---

## 禁止事項

- JWT シークレットをハードコードしない。
- トークンを `localStorage` / `sessionStorage` に保存しない（XSS で盗まれる）。
- トークンをレスポンスボディで返さない。
- 認証 Cookie から `HttpOnly` を外さない。
- リフレッシュトークンを平文で DB に保存しない。
- Firebase サービスアカウント JSON をリポジトリにコミットしない。
- ログにトークンや password_hash を出力しない。
