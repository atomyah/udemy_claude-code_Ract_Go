# CLAUDE.md — SNS アプリ プロジェクトガイド

このファイルは Claude Code がこのリポジトリで作業するときの基本設定・ルールを定義する。
詳細な要件・仕様は `.claude/SPEC.md` を参照すること。

---

## プロジェクト概要

Twitter ライクな SNS アプリ。テキスト・画像・動画を投稿し、いいね・コメント・リポスト・ブックマークでインタラクションできる。

**主な機能:**
- メール＋パスワード / Google OAuth ログイン（JWT 認証）
- 投稿（テキスト必須、画像最大 4 枚、動画 1 本）
- フォロー / フォロワー、ホームタイムライン（フォロー中）
- いいね・コメント・リポスト・ブックマーク
- ハッシュタグ・ユーザー・投稿内容の検索
- アプリ内通知
- ライト / ダークテーマ切り替え
- 管理者による投稿削除・ユーザー停止

---

## 技術スタック

### フロントエンド（`frontend/`）
- React 18 + TypeScript
- MUI (Material UI) v5 — カスタムテーマ（ライト / ダーク）
- openapi-typescript で Swagger 定義から型を自動生成
- TanStack Query（React Query）でサーバー状態管理
- React Router v6
- Axios

### バックエンド（`backend/`）
- Go 1.22+、Echo v4、GORM v2
- swaggo/echo-swagger で OpenAPI 3.0 定義を自動生成
- golang-jwt/jwt（JWT 認証）
- **air でホットリロード**（開発時。`go run` による毎回ビルドは使わない）

### インフラ
- PostgreSQL 16
- Firebase Storage（画像・動画・アバター・バナー）
- Firebase Auth（Google OAuth 連携）
- ローカル: Docker + docker-compose（PostgreSQL + バックエンド）
- 本番フロントエンド: Firebase Hosting
- 本番バックエンド: Render または Google Cloud Run

---

## ルールファイル一覧

詳細ルールは `.claude/rules/` 以下を参照。

| ファイル | 対象 |
|---------|------|
| `rules/frontend.md` | React / TypeScript / MUI のコーディング規約 |
| `rules/backend.md` | Go / Echo / GORM のコーディング規約 |
| `rules/api.md` | REST API 設計規則・OpenAPI / Swagger 規約 |
| `rules/database.md` | PostgreSQL スキーマ・マイグレーション規約 |
| `rules/infra.md` | Docker / デプロイ規約 |

---

## 開発コマンド実行ルール

> **重要: ホスト OS で `go` コマンドを直接実行しないこと。**
> すべての `go` コマンドは必ず `docker compose exec api` 経由で実行する。

```bash
# OK（コンテナ内で実行）
docker compose exec api go get github.com/labstack/echo/v4
docker compose exec api go mod tidy
docker compose exec api swag init -g cmd/server/main.go --output docs

# NG（ホスト OS で直接実行）
go get github.com/labstack/echo/v4   # 禁止
go mod tidy                           # 禁止
```

## 開発フロー

### 初回セットアップ
```bash
# 1. コンテナを起動（初回は air のビルドが失敗するが正常）
docker compose up -d

# 2. コンテナ内で依存パッケージを取得（go.mod + go.sum が更新される）
docker compose exec api go get github.com/labstack/echo/v4
docker compose exec api go mod tidy
# → air が自動的にリビルドしてサーバーが起動する

# 3. OpenAPI ドキュメントを生成
docker compose exec api swag init -g cmd/server/main.go --parseDependency --parseInternal --output docs
```

### 日常的な起動
```bash
# コンテナ起動（air によるホットリロードが有効）
docker compose up -d

# ログ確認
docker compose logs -f api
```

### 依存パッケージ追加
```bash
docker compose exec api go get <package>
docker compose exec api go mod tidy
```

### OpenAPI ドキュメント再生成（ハンドラー変更後）
```bash
docker compose exec api swag init -g cmd/server/main.go --parseDependency --parseInternal --output docs
```

### フロントエンド起動
```bash
cd frontend && npm run dev
```

### OpenAPI 型生成（フロントエンド）
```bash
# backend の swaggo で docs/swagger.json を生成後に実行
cd frontend && npm run gen:api
```

---

## 重要な設計方針

1. **API first**: バックエンドの swaggo コメントで OpenAPI 定義を生成し、フロントエンドは openapi-typescript で型を使う。手書き型は作らない。
2. **Cursor ベースのページネーション**: タイムライン・一覧系は `?cursor=<uuid>&limit=20` を使う。offset は使わない。
3. **論理削除**: 投稿は `is_deleted = true` で論理削除。物理削除はしない。
4. **JWT の扱い**: Access Token は `Authorization: Bearer <token>` ヘッダーで送る。Refresh Token は HttpOnly Cookie。
5. **Firebase Storage**: ファイルアップロードはバックエンド経由（multipart を受け取り Firebase SDK でアップロード）。フロントから直接アップロードしない。
6. **エラーレスポンス**: 全 API エラーは `{ "code": "ERROR_CODE", "message": "説明" }` の統一フォーマット。
