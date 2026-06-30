# セキュリティ規約

## 認証・認可

- JWT の取り扱いは `rules/auth.md` を参照。
- 全ての保護エンドポイントに JWT ミドルウェアを必ず適用する。ルーティング設定でまとめて適用し、ハンドラー個別に処理しない。
- 管理者専用エンドポイントには JWT ミドルウェアの後に admin ロールチェックミドルウェアを適用する。

---

## 入力バリデーション

### バックエンド

- リクエストボディは DTO 構造体に bind した直後に必ずバリデーションする。
- `github.com/go-playground/validator/v10` を使い、DTO タグでルールを定義する。
- バリデーションエラーは 400 + `VALIDATION_ERROR` コードで返す。
- 文字列フィールドはサニタイズせず、バリデーションでサイズ・形式を制限する（DB は GORM のプリペアドステートメントが SQL インジェクションを防ぐ）。

```go
type CreatePostRequest struct {
    Content string `json:"content" validate:"required,max=280"`
}
```

### フロントエンド

- フォームバリデーションは `react-hook-form` のルールで行う。
- バックエンドがサニタイズ済みとして信頼せず、表示時は MUI コンポーネントの自然なエスケープを活用する（`dangerouslySetInnerHTML` は禁止）。

---

## SQL インジェクション

- GORM のプリペアドステートメントを常に使う。生の SQL 文字列結合は禁止。

```go
// OK
db.Where("handle = ?", handle).First(&user)

// NG（禁止）
db.Raw("SELECT * FROM users WHERE handle = '" + handle + "'").Scan(&user)
```

- 検索クエリの `ILIKE` も必ずプレースホルダーを使う。

```go
db.Where("content ILIKE ?", "%"+query+"%").Find(&posts)
```

---

## XSS（クロスサイトスクリプティング）

- React は JSX の変数展開でデフォルトエスケープするので、`{variable}` 形式で表示する限り安全。
- `dangerouslySetInnerHTML` は絶対に使わない。
- ハッシュタグ・メンションのリンク化は React コンポーネントで行い、文字列 HTML は生成しない。

---

## CORS

- 許可オリジンは環境変数 `CORS_ORIGINS` で管理する（カンマ区切り）。
- 開発: `http://localhost:5173`
- 本番: `https://<firebase-app>.web.app`
- `*`（ワイルドカード）は絶対に使わない。

```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: strings.Split(os.Getenv("CORS_ORIGINS"), ","),
    AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
}))
```

---

## ファイルアップロード

- ファイルの MIME タイプをサーバーサイドで検証する（`Content-Type` ヘッダーだけでなく実際のバイト列でも確認）。
- 許可する形式: 画像（JPEG / PNG / WebP）、動画（MP4 / MOV）のみ。
- サイズ制限: 画像 5MB、動画 100MB。Echo のリクエストサイズ制限ミドルウェアで上限を設ける。
- アップロードファイルを直接サーバーのファイルシステムに保存しない（Firebase Storage に転送する）。

---

## 機密情報管理

- JWT シークレット・DB パスワード・Firebase サービスアカウントは全て環境変数で管理する。
- シークレット値は 32 バイト以上のランダム文字列を使う。
- ログに機密情報（パスワード・トークン・メールアドレス）を出力しない。
- エラーレスポンスに内部スタックトレースを含めない（本番環境）。

```go
// 本番ではスタックトレースを隠す
if cfg.Env == "production" {
    e.HTTPErrorHandler = customErrorHandler // 詳細を隠す
}
```

---

## レート制限（推奨）

認証エンドポイントにはブルートフォース攻撃対策としてレート制限を設ける。

- `POST /api/v1/auth/login`: IP あたり 10 回/分
- `POST /api/v1/auth/register`: IP あたり 5 回/分
- Echo のミドルウェアまたは `golang.org/x/time/rate` で実装する。

---

## HTTPS

- 本番環境は必ず HTTPS を使う（Render / Cloud Run / Firebase Hosting はデフォルト HTTPS）。
- HTTP → HTTPS リダイレクトはデプロイ先のプラットフォームに任せる。
- Cookie の `Secure` 属性は本番環境でのみ有効にする（開発では false）。

---

## 禁止事項

- `dangerouslySetInnerHTML` の使用。
- CORS に `*` を設定する。
- 生の SQL 文字列結合（プリペアドステートメントを使う）。
- 機密情報（トークン・パスワード）のログ出力。
- 本番のエラーレスポンスにスタックトレースを含める。
- Firebase サービスアカウント JSON をコミットする。
