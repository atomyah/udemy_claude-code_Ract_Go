# PHASE 01 — 開発環境・プロジェクト基盤構築

> 目標: コードを書き始める前に、ローカルで全サービスが動く状態を作る。

---

## バックエンド（Go）

- [x] `backend/` ディレクトリ作成
- [x] `go mod init github.com/<username>/sns-backend`
- [x] 依存パッケージをインストール
  - [x] `github.com/labstack/echo/v4`
  - [x] `gorm.io/gorm` + `gorm.io/driver/postgres`
  - [x] `github.com/golang-jwt/jwt/v5`
  - [x] `github.com/google/uuid`
  - [x] `github.com/swaggo/echo-swagger` + `github.com/swaggo/swag`
  - [x] `firebase.google.com/go/v4`（Firebase Storage / Auth）
  - [ ] `github.com/caarlos0/env/v11`（環境変数読み込み）— 代わりに `os.Getenv` で実装（rules/backend.md の許容範囲内）
- [x] ディレクトリ構造作成
  ```
  backend/
  ├── cmd/server/main.go
  ├── internal/
  │   ├── handler/
  │   ├── service/
  │   ├── repository/
  │   ├── model/
  │   ├── middleware/
  │   ├── dto/
  │   └── config/
  ├── migrations/
  └── docs/
  ```
- [x] `cmd/server/main.go` に Echo の最小起動コードを書く
- [x] `GET /health` エンドポイントを実装（認証不要）
- [x] `internal/config/config.go` で環境変数を読み込む構造体を定義
- [x] `backend/.env` を作成（`.gitignore` に追加）

## air（ホットリロード）設定

- [x] `go install github.com/air-verse/air@latest`
- [x] `backend/.air.toml` を作成・設定
  - build cmd: `go build -o ./tmp/main ./cmd/server`
  - include_ext: `go`
  - exclude_dir: `docs,tmp,vendor`

## Docker / docker-compose

- [x] `backend/Dockerfile.dev`（air を使う開発用）を作成
- [x] `backend/Dockerfile`（本番用マルチステージビルド）を作成
- [x] `docker-compose.yml` を作成（PostgreSQL + バックエンド）
- [x] `.env.example` を作成してコミット用テンプレートとして残す
- [x] `docker-compose up -d db` で PostgreSQL が起動することを確認
- [x] `air` でバックエンドが起動し `GET /health` が 200 を返すことを確認

## フロントエンド（React）

- [x] `frontend/` ディレクトリ作成
- [x] `npm create vite@latest frontend -- --template react-ts`
- [x] 依存パッケージをインストール
  - [x] `@mui/material @emotion/react @emotion/styled`
  - [x] `@mui/icons-material`
  - [x] `@tanstack/react-query`
  - [x] `react-router-dom`
  - [x] `axios`
  - [x] `openapi-typescript`（dev）
- [x] `frontend/.env.local` を作成（`VITE_API_BASE_URL=http://localhost:8080`）
- [x] `npm run dev` でフロントエンドが起動することを確認

## Swagger セットアップ

- [x] `swag init -g cmd/server/main.go` で `docs/` を初回生成
- [x] Echo に Swagger UI ルートを追加（`/swagger/*`、`ENV=development` のみ）
- [x] ブラウザで `http://localhost:8080/swagger/index.html` が表示されることを確認

## 完了基準

- [x] `docker-compose up` で PostgreSQL とバックエンドが両方起動する
- [x] `curl http://localhost:8080/health` が `{"status":"ok"}` を返す
- [x] `http://localhost:5173` でフロントエンドが表示される
- [x] `http://localhost:8080/swagger/index.html` で Swagger UI が表示される
- [x] `air` を止めて `main.go` を変更すると自動再起動される（ホットリロード確認）
