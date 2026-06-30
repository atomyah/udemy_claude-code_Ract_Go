# OpenAPI / Swagger 規約

## 方針

- バックエンドは **swaggo** のアノテーションコメントから OpenAPI 3.0 定義を自動生成する。
- フロントエンドは **openapi-typescript** で生成された型のみを使う。手書きの API 型は作らない。
- `docs/` ディレクトリは `swag init` で生成する。手動編集禁止。

---

## swaggo アノテーション規約（バックエンド）

### 必須タグ

全ハンドラー関数に以下のタグを付ける。1 つでも欠けたらレビューで指摘する。

```go
// FunctionName godoc
// @Summary      エンドポイントの1行説明（日本語可）
// @Tags         タグ名（機能グループ）
// @Accept       json（または multipart/form-data）
// @Produce      json
// @Param        パラメータ定義（後述）
// @Success      ステータスコード {object} dto.XxxResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Router       /api/v1/xxx [メソッド]
// @Security     BearerAuth（認証必要なエンドポイントのみ）
func (h *Handler) FunctionName(c echo.Context) error {
```

### @Tags の一覧

| Tag | 対象 |
|-----|------|
| `auth` | 認証関連 |
| `users` | ユーザー・プロフィール |
| `posts` | 投稿 |
| `interactions` | いいね・リポスト・ブックマーク |
| `search` | 検索 |
| `notifications` | 通知 |
| `admin` | 管理者 |

### @Param の書き方

```go
// @Param  id      path    string  true  "投稿 ID (UUID)"
// @Param  cursor  query   string  false "ページネーション cursor"
// @Param  limit   query   int     false "取得件数 (デフォルト: 20, 最大: 50)"
// @Param  body    body    dto.CreatePostRequest  true  "リクエストボディ"
// @Param  file    formData file   true  "アップロードファイル"
```

### DTO のアノテーション

```go
// CreatePostRequest godoc
type CreatePostRequest struct {
    Content string `json:"content" example:"こんにちは！" validate:"required,max=280"`
}
```

---

## swag コマンド

```bash
# docs/ を生成（backend/ ディレクトリで実行）
swag init -g cmd/server/main.go --output docs

# フォーマット（アノテーションの整形）
swag fmt
```

`swag init` は機能追加のたびに実行し、`docs/` を最新化する。

---

## main.go のグローバル設定

```go
// @title           SNS API
// @version         1.0
// @description     Twitter ライクな SNS アプリの REST API
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     "Bearer <token>" 形式で入力
func main() {
```

---

## openapi-typescript（フロントエンド）

### セットアップ

```bash
npm install -D openapi-typescript
```

`package.json` に型生成スクリプトを追加:

```json
{
  "scripts": {
    "gen:api": "openapi-typescript http://localhost:8080/swagger/doc.json -o src/api/generated/schema.ts"
  }
}
```

### 生成型の使い方

```ts
import type { paths, components } from '@/api/generated/schema';

// レスポンス型の例
type PostResponse = components['schemas']['PostResponse'];

// リクエスト型の例
type CreatePostRequest = components['schemas']['CreatePostRequest'];
```

### 型付き API ラッパー

生成型をそのまま axios で使う型付きヘルパーを `src/api/` に作る。

```ts
// src/api/posts.ts
import { api } from './client';
import type { components } from './generated/schema';

type PostResponse = components['schemas']['PostResponse'];

export const getPosts = async (cursor?: string): Promise<PostResponse[]> => {
  const { data } = await api.get('/posts', { params: { cursor, limit: 20 } });
  return data.data;
};
```

---

## Swagger UI

- 開発環境: `http://localhost:8080/swagger/index.html` で閲覧可能。
- 本番環境: `ENV=production` のとき Swagger UI ルートを登録しない（情報漏洩防止）。

```go
if cfg.Env != "production" {
    e.GET("/swagger/*", echoSwagger.WrapHandler)
}
```

---

## 禁止事項

- `docs/` ディレクトリを手動で編集する。
- フロントエンドで手書きの API レスポンス型を定義する（生成型を使う）。
- swaggo コメントのないエンドポイントをコミットする。
- 本番環境で Swagger UI を公開する。
