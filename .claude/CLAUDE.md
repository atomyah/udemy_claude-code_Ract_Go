# CLAUDE.md — SNS アプリ プロジェクトガイド

このファイルは Claude Code がこのリポジトリで作業するときの基本設定・ルールを定義する。
詳細な要件・仕様は `.claude/SPEC.md` を参照すること。

---

## プロジェクト概要

Twitter ライクな SNS アプリ。テキスト・画像・動画を投稿し、いいね・コメント・リポスト・ブックマークでインタラクションできる。

**主な機能:**
- メール＋パスワード / Google OAuth ログイン（JWT 認証）
- 投稿（テキスト必須、画像最大 4 枚、動画 1 本）
- フォロー / フォロワー、ホームタイムライン（全ユーザーの投稿）
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
# → 上記実行後、必ず以下も実行して openapi.yaml を更新すること（フックで自動実行される）
npx swagger2openapi ./docs/swagger.json -o ./docs/openapi.yaml -y
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

## Swagger → OpenAPI 変換ルール

- `swag init` で `docs/swagger.json` を生成・更新したら、**必ず** `openapi.yaml` への変換も実行すること。
- 変換コマンド: `npx swagger2openapi ./docs/swagger.json -o ./docs/openapi.yaml -y`（`backend/` ディレクトリで実行）
- フックにより `swag init` を含む Bash コマンドの実行後に自動で変換が走る。手動実行時も同様に実行すること。
- `docs/swagger.json` と `docs/openapi.yaml` は常に同期した状態を保つ。

---

## 重要な設計方針

1. **API first**: バックエンドの swaggo コメントで OpenAPI 定義を生成し、フロントエンドは openapi-typescript で型を使う。手書き型は作らない。
2. **Cursor ベースのページネーション**: タイムライン・一覧系は `?cursor=<uuid>&limit=20` を使う。offset は使わない。
3. **論理削除**: 投稿は `is_deleted = true` で論理削除。物理削除はしない。
4. **JWT の扱い**: Access Token は `Authorization: Bearer <token>` ヘッダーで送る。Refresh Token は HttpOnly Cookie。
5. **Firebase Storage**: ファイルアップロードはバックエンド経由（multipart を受け取り Firebase SDK でアップロード）。フロントから直接アップロードしない。
6. **エラーレスポンス**: 全 API エラーは `{ "code": "ERROR_CODE", "message": "説明" }` の統一フォーマット。

---

## 実装ルール（必ず守ること）

1. **要件の洗い出しを先に行う**: 実装に着手する前に、依頼された要件をすべて箇条書きで列挙すること。実装後はその一覧と照合し、**1 つも漏らさず実装されているか確認する**。
2. **環境別の動作要件を必ず実装する**: 「本番環境でのみ有効」「開発環境では無効」といった環境差分の要件は、必ずコードとして実装すること（例: Swagger UI は `ENV=development` のときだけ公開する、`AutoMigrate` は development / test のみ実行する）。判断は `config.Config` の `IsDevelopment()` / `IsTest()` / `IsProduction()` / `ShouldAutoMigrate()` で行う。
3. **新機能にはテストを書く**: 新しいハンドラー・サービス・リポジトリ・画面を実装したら、対応するテスト（バックエンド単体テスト／E2E テスト）も必ず同じ変更内で追加する。テストのない機能追加はしない。
4. **エラーメッセージはユーザー向けの日本語にする**: バリデーションエラーを含め、API が返す `message` は利用者が読んで理解できる日本語であること。`validator` の生エラー文字列をそのまま返さず、`handler.validationMessage()` を通す。

---

## テスト

### テスト環境（開発環境とは完全に分離）

| サービス | 用途 | ポート |
|---------|------|-------|
| `db_test` | テスト用 PostgreSQL（データは tmpfs で永続化しない） | 5433 |
| `api_test` | テスト用 API サーバー（`ENV=test`） | 8081 |
| `adminer` | 開発用 DB 管理 UI（8081 は api_test が使うため 8082 に変更） | 8082 |

- テスト環境は Compose の **profile `test`** に属し、`docker compose --profile test up -d` でのみ起動する。
- 設定は `backend/.env.test`（秘密情報を含まないためコミット可）。開発用 `backend/.env` とは別ファイル。
- `api_test` のエントリーポイント（`backend/scripts/test-entrypoint.sh`）が、テスト／サーバー起動前に**必ず DB マイグレーション**（`go run ./cmd/migrate`）を実行する。
- テスト実行で開発用 `db` / `api` コンテナのデータに影響を与えてはならない。`docker compose down -v` のような開発ボリュームごと消すコマンドは使わない。

### テストコマンドは Makefile 経由で実行する

```bash
make help          # 全コマンドの一覧
make dev-up        # 開発環境を起動
make dev-down      # 開発環境を停止
make dev-logs      # 開発環境のログ
make test-setup    # テスト環境を起動（profile: test）
make test-teardown # テスト環境を停止・削除
make test-backend  # テスト環境起動 → バックエンド単体テスト → テスト環境停止
make test-e2e      # テスト環境起動 → E2E テスト → テスト環境停止
make test          # test-backend と test-e2e を両方実行
```

- テストを直接 `go test` や `npx playwright test` で叩かず、**必ず Makefile 経由**で実行する。
- `make` が未インストールの場合は導入する（例: `winget install GnuWin32.Make`）。
- テストが失敗しても `test-teardown` が必ず実行され、テスト環境は残らない。

### テスト用コンテナは `api_test` を使う

> **重要: テスト関連の `go` コマンドは開発用 `api` コンテナではなく `api_test` コンテナで実行する。**

```bash
# OK（テスト用コンテナ。テスト用 DB に接続され、開発データに影響しない）
docker compose run --rm api_test go test -v ./...

# NG（開発用コンテナ。開発用 DB を汚す）
docker compose exec api go test ./...
```

依存パッケージの追加や `swag init` など、テスト以外の `go` コマンドは従来どおり `docker compose exec api` で実行する。

### 並列実行数の制限

マシン負荷を抑えるため、テストの並列度は必ず制限する。

- Go: `go test -parallel 2`（Makefile の `test-backend` で指定済み）
- Playwright: `workers: 1`（`frontend/playwright.config.ts` で指定済み）
- テストケース内のループ（投稿を複数作る等）は、**検証に必要な最小回数**にとどめる。大量データを作るループは書かない。

### バックエンド単体テスト

- 対象: ハンドラー層（ステータスコード・日本語エラーメッセージ）、サービス層（ビジネスロジック・認証・認可）、ミドルウェア層（JWT 認証）、リポジトリ層（テスト DB を使った統合テスト）。
- モックは `testify/mock` で各パッケージの `mocks_test.go` に定義する。
- カバレッジ目標は **60% 以上**（`internal/` 配下）。`make test-backend` の最後に集計値が表示される。

### E2E テスト（Playwright）

- 配置: `frontend/e2e/*.spec.ts`、共通処理は `frontend/e2e/helpers.ts`。
- 接続先はテスト用 API サーバー `http://localhost:8081`（`frontend/.env.e2e` の `VITE_API_BASE_URL`）。フロントは `npm run dev:e2e`（ポート 5174）で起動する。
- **テスト名は日本語**で書く（例: `test('投稿するとタイムラインに表示される', ...)`）。
- **セレクタは `data-testid` 属性のみ**を使う（`getByTestId`）。必要な `data-testid` はコンポーネント側に追加する。
- 入力値はバックエンドのバリデーションを満たすものにする（パスワード 8〜72 文字、ハンドル 3〜50 文字の英数字とアンダースコア、投稿本文 1〜280 文字）。
