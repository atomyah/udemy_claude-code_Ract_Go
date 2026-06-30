# PHASE 01 — 開発環境・プロジェクト基盤構築

> 目標: コードを書き始める前に、ローカルで全サービスが動く状態を作る。

---

## バックエンド（Go）

- [ ] `backend/` ディレクトリ作成
- [ ] `go mod init github.com/<username>/sns-backend`
- [ ] 依存パッケージをインストール
  - [ ] `github.com/labstack/echo/v4`
  - [ ] `gorm.io/gorm` + `gorm.io/driver/postgres`
  - [ ] `github.com/golang-jwt/jwt/v5`
  - [ ] `github.com/google/uuid`
  - [ ] `github.com/swaggo/echo-swagger` + `github.com/swaggo/swag`
  - [ ] `firebase.google.com/go/v4`（Firebase Storage / Auth）
  - [ ] `github.com/caarlos0/env/v11`（環境変数読み込み）
- [ ] ディレクトリ構造作成
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
- [ ] `cmd/server/main.go` に Echo の最小起動コードを書く
- [ ] `GET /health` エンドポイントを実装（認証不要）
- [ ] `internal/config/config.go` で環境変数を読み込む構造体を定義
- [ ] `backend/.env` を作成（`.gitignore` に追加）

## air（ホットリロード）設定

- [ ] `go install github.com/air-verse/air@latest`
- [ ] `backend/.air.toml` を作成・設定
  - build cmd: `go build -o ./tmp/main ./cmd/server`
  - include_ext: `go`
  - exclude_dir: `docs,tmp,vendor`

## Docker / docker-compose

- [ ] `backend/Dockerfile.dev`（air を使う開発用）を作成
- [ ] `backend/Dockerfile`（本番用マルチステージビルド）を作成
- [ ] `docker-compose.yml` を作成（PostgreSQL + バックエンド）
- [ ] `.env.example` を作成してコミット用テンプレートとして残す
- [ ] `docker-compose up -d db` で PostgreSQL が起動することを確認
- [ ] `air` でバックエンドが起動し `GET /health` が 200 を返すことを確認

## フロントエンド（React）

- [ ] `frontend/` ディレクトリ作成
- [ ] `npm create vite@latest frontend -- --template react-ts`
- [ ] 依存パッケージをインストール
  - [ ] `@mui/material @emotion/react @emotion/styled`
  - [ ] `@mui/icons-material`
  - [ ] `@tanstack/react-query`
  - [ ] `react-router-dom`
  - [ ] `axios`
  - [ ] `openapi-typescript`（dev）
- [ ] `frontend/.env.local` を作成（`VITE_API_BASE_URL=http://localhost:8080`）
- [ ] `npm run dev` でフロントエンドが起動することを確認

## Swagger セットアップ

- [ ] `swag init -g cmd/server/main.go` で `docs/` を初回生成
- [ ] Echo に Swagger UI ルートを追加（`/swagger/*`、`ENV=development` のみ）
- [ ] ブラウザで `http://localhost:8080/swagger/index.html` が表示されることを確認

## 完了基準

- [ ] `docker-compose up` で PostgreSQL とバックエンドが両方起動する
- [ ] `curl http://localhost:8080/health` が `{"status":"ok"}` を返す
- [ ] `http://localhost:5173` でフロントエンドが表示される
- [ ] `http://localhost:8080/swagger/index.html` で Swagger UI が表示される
- [ ] `air` を止めて `main.go` を変更すると自動再起動される（ホットリロード確認）
