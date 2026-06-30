# API 設計規約

## 基本方針

- RESTful API。バージョンプレフィックスは `/api/v1/`。
- JSON レスポンス（`Content-Type: application/json`）。
- ファイルアップロードのみ `multipart/form-data`。
- 全エンドポイントを swaggo コメントで文書化し、OpenAPI 定義を自動生成する。

---

## URL 設計

### 命名規則

- リソース名は**複数形の小文字スネークケース**（`/posts`, `/users`, `/notifications`）。
- ネストは 2 階層まで（`/posts/:id/comments` は OK、それ以上は避ける）。
- アクション系は動詞を使わず、HTTP メソッドで表現する（`DELETE /posts/:id/like` など）。

### エンドポイント一覧

#### 認証
```
POST   /api/v1/auth/register       新規登録
POST   /api/v1/auth/login          ログイン
POST   /api/v1/auth/logout         ログアウト
POST   /api/v1/auth/refresh        トークンリフレッシュ
POST   /api/v1/auth/google         Google OAuth コールバック
```

#### ユーザー
```
GET    /api/v1/users/me            自分のプロフィール取得
PUT    /api/v1/users/me            自分のプロフィール更新
GET    /api/v1/users/:handle       指定ユーザーのプロフィール取得
GET    /api/v1/users/:handle/posts     投稿一覧
GET    /api/v1/users/:handle/followers フォロワー一覧
GET    /api/v1/users/:handle/following フォロー中一覧
POST   /api/v1/users/:handle/follow    フォロー
DELETE /api/v1/users/:handle/follow    アンフォロー
```

#### 投稿
```
GET    /api/v1/posts               探索（全体）タイムライン
GET    /api/v1/posts/home          ホームタイムライン（要認証）
POST   /api/v1/posts               投稿作成（要認証）
GET    /api/v1/posts/:id           投稿詳細
PUT    /api/v1/posts/:id           投稿編集（本人のみ）
DELETE /api/v1/posts/:id           投稿削除（本人のみ）
GET    /api/v1/posts/:id/comments  コメント一覧
```

#### インタラクション
```
POST   /api/v1/posts/:id/like      いいね
DELETE /api/v1/posts/:id/like      いいね取消
POST   /api/v1/posts/:id/repost    リポスト
DELETE /api/v1/posts/:id/repost    リポスト取消
POST   /api/v1/posts/:id/bookmark  ブックマーク
DELETE /api/v1/posts/:id/bookmark  ブックマーク取消
GET    /api/v1/bookmarks           自分のブックマーク一覧（要認証）
```

#### 検索
```
GET    /api/v1/search/users?q=     ユーザー検索
GET    /api/v1/search/posts?q=     投稿内容検索
GET    /api/v1/search/hashtags/:tag ハッシュタグ検索
```

#### 通知
```
GET    /api/v1/notifications       通知一覧（要認証）
PUT    /api/v1/notifications/read  全通知を既読にする（要認証）
```

#### 管理者（要 admin ロール）
```
DELETE /api/v1/admin/posts/:id             投稿強制削除
PUT    /api/v1/admin/users/:id/suspend     ユーザー停止
DELETE /api/v1/admin/users/:id/suspend     ユーザー停止解除
```

---

## ページネーション

**cursor ベース**のページネーションを使う。offset ベースは使わない。

### クエリパラメータ

```
GET /api/v1/posts?cursor=<uuid>&limit=20
```

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `cursor` | string (UUID) | なし（最初のページ） | 前回のレスポンスの `nextCursor` |
| `limit` | int | 20 | 最大 50 |

### レスポンス形式

```json
{
  "data": [...],
  "nextCursor": "uuid-or-null",
  "hasMore": true
}
```

- `nextCursor` が `null` または `hasMore` が `false` のとき最終ページ。

---

## レスポンス形式

### 成功時（単一リソース）
```json
{
  "data": { ... }
}
```

### 成功時（一覧）
```json
{
  "data": [...],
  "nextCursor": "uuid",
  "hasMore": true
}
```

### エラー時
```json
{
  "code": "POST_NOT_FOUND",
  "message": "投稿が見つかりません"
}
```

---

## HTTP ステータスコード

| コード | 使用場面 |
|-------|---------|
| 200 | 取得・更新成功 |
| 201 | 作成成功 |
| 204 | 削除成功（レスポンスボディなし） |
| 400 | バリデーションエラー |
| 401 | 未認証 |
| 403 | 権限なし（認証済みだが権限不足） |
| 404 | リソースが見つからない |
| 409 | 競合（重複いいねなど） |
| 422 | 処理不能（ビジネスルール違反） |
| 500 | サーバーエラー |

---

## 認証

- 保護されたエンドポイントは `Authorization: Bearer <access_token>` ヘッダーが必要。
- Swagger 定義に `securityDefinitions` で `BearerAuth` を定義し、各エンドポイントに `// @Security BearerAuth` を付ける。

---

## ファイルアップロード API

```
POST /api/v1/posts
Content-Type: multipart/form-data

content: "テキスト内容"
media[]: <file1>
media[]: <file2>
```

- `media[]` は最大 4 ファイル（画像）または 1 ファイル（動画）。
- 画像: JPEG / PNG / WebP、1 枚 5MB 以内。
- 動画: MP4 / MOV、100MB 以内、2 分 20 秒以内。
- バリデーションはバックエンドで行い、400 エラーを返す。

---

## OpenAPI / Swagger 規約

- swaggo の `swag init -g cmd/server/main.go` で `docs/` を生成する。
- 開発時は `/swagger/index.html` で Swagger UI を確認できる。
- 本番では Swagger UI は無効化する（`ENV=production` で非公開）。
- DTO 構造体には `// @Description` コメントを付ける。
