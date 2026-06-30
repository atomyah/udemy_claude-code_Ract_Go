# 認証規約（JWT / Firebase Auth）

## JWT 設計

| トークン | 有効期限 | 保存場所 | 用途 |
|---------|---------|---------|------|
| Access Token | 15 分 | フロントエンド: `localStorage` | API リクエストヘッダー |
| Refresh Token | 7 日 | HttpOnly Cookie | Access Token の再発行 |

### Access Token

- `Authorization: Bearer <token>` ヘッダーで送受信する。
- ペイロードに含める最小限の情報: `sub`（user_id）, `role`, `exp`, `iat`
- 署名アルゴリズム: `HS256`（秘密鍵は環境変数 `JWT_SECRET`）

### Refresh Token

- HttpOnly + Secure + SameSite=Strict の Cookie でのみ送受信する。
- バックエンドの `POST /api/v1/auth/refresh` で Access Token を再発行する。
- 別の署名鍵（`JWT_REFRESH_SECRET`）を使う。

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

### エラーコード

| 状況 | HTTP | code |
|------|------|------|
| トークンなし | 401 | `MISSING_TOKEN` |
| トークン形式不正 | 401 | `INVALID_TOKEN` |
| トークン期限切れ | 401 | `EXPIRED_TOKEN` |
| アカウント停止 | 401 | `ACCOUNT_SUSPENDED` |
| 権限不足 | 403 | `FORBIDDEN` |

### パスワード

- `bcrypt` でハッシュ化する（コスト係数: 12）。
- 平文パスワードをログに出力しない。
- 最小 8 文字、最大 72 文字（bcrypt の上限）。

---

## Firebase Auth（Google OAuth）

- フロントエンドは `signInWithPopup` で ID Token を取得する。
- ID Token はバックエンドの `POST /api/v1/auth/google` に送り、バックエンドで Firebase Admin SDK を使って検証する。
- フロントエンドから Firebase Storage や Firestore に直接アクセスしない（Storage のみ許可、Admin SDK 経由）。
- サービスアカウントの JSON は環境変数 `FIREBASE_CREDENTIALS_JSON` で管理し、ファイルとしてコミットしない。

---

## フロントエンド実装規約

### トークン管理

- Access Token は `localStorage` に保存する（`access_token` キー）。
- Axios インターセプターで自動的にヘッダーに付与する。
- 401 レスポンスを受け取ったら Refresh Token で自動再取得を試みる。
- 再取得も失敗したらトークンを削除してログアウト状態にする。

```ts
// Axios インターセプター（概要）
api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401 && !error.config._retry) {
      error.config._retry = true;
      await refreshToken(); // POST /auth/refresh
      return api(error.config);
    }
    return Promise.reject(error);
  }
);
```

### ログアウト時の処理

1. `POST /api/v1/auth/logout` を呼ぶ（Refresh Token Cookie を削除）
2. `localStorage` から `access_token` を削除
3. TanStack Query のキャッシュを全クリア（`queryClient.clear()`）
4. `/login` にリダイレクト

---

## 禁止事項

- JWT シークレットをハードコードしない。
- Access Token を Cookie に保存しない（XSS リスク）。
- Refresh Token を `localStorage` に保存しない（XSS リスク）。
- Firebase サービスアカウント JSON をリポジトリにコミットしない。
- ログにトークンや password_hash を出力しない。
