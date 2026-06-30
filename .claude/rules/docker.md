# Docker / docker-compose 規約

## ファイル構成

```
project-root/
├── docker-compose.yml          # ローカル開発用（DB + バックエンド）
├── docker-compose.prod.yml     # 本番確認用（オプション）
├── backend/
│   ├── Dockerfile              # 本番用（マルチステージビルド）
│   └── Dockerfile.dev          # 開発用（air によるホットリロード）
└── .env.example                # 環境変数のテンプレート（コミット可）
```

---

## docker-compose.yml（開発用）

```yaml
services:
  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-sns_db}
      POSTGRES_USER: ${POSTGRES_USER:-sns_user}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-sns_password}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-sns_user}"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    restart: unless-stopped
    volumes:
      - ./backend:/app
      - /app/tmp          # air のビルド出力を除外
    env_file:
      - ./backend/.env
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy
    command: air

volumes:
  postgres_data:
```

---

## Dockerfile.dev（開発用・バックエンド）

```dockerfile
FROM golang:1.22-alpine
WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

# ソースはボリュームマウントで提供するためここではコピー不要
CMD ["air"]
```

---

## Dockerfile（本番用・バックエンド）

```dockerfile
# ビルドステージ
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 実行ステージ
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
```

---

## 規約・ルール

### 環境変数

- `.env` ファイルは `docker-compose.yml` の `env_file` で読み込む。
- `.env` はコミットしない（`.gitignore` に追加）。
- `.env.example` はコミットして変数名の一覧を共有する（値は空にする）。
- 環境変数は全て `docker-compose.yml` の `environment` または `env_file` で渡す。コンテナ内にハードコードしない。

### ボリューム

- PostgreSQL データは名前付きボリューム（`postgres_data`）で永続化する。
- 開発時のバックエンドコードはバインドマウントで共有し、コンテナ再起動不要にする。
- `air` のビルド成果物 `tmp/` は別ボリュームで除外し、ホストに漏れさせない。

### ヘルスチェック

- `db` サービスに `healthcheck` を設定し、`backend` は `depends_on: condition: service_healthy` で DB の起動を待つ。
- バックエンドに `GET /health` を実装して本番デプロイ先のヘルスチェックにも使う。

### イメージ

- ベースイメージは常にバージョンを固定する（`postgres:16-alpine`、`golang:1.22-alpine`）。`latest` タグは使わない。
- 本番イメージは `alpine` ベースで軽量化する。

---

## よく使うコマンド

```bash
# DB のみ起動（バックエンドはローカルで air 実行）
docker-compose up -d db

# 全サービス起動
docker-compose up

# ログ確認
docker-compose logs -f backend

# DB に入る
docker-compose exec db psql -U sns_user -d sns_db

# コンテナを全て停止・削除（ボリュームは残す）
docker-compose down

# ボリュームごと削除（DB データリセット）
docker-compose down -v
```

---

## 禁止事項

- ベースイメージに `latest` タグを使う。
- 本番用 Dockerfile でソースコードを `COPY . .` したまま最終ステージに含める（マルチステージビルドで成果物のみにする）。
- `.env` ファイルをコミットする。
- `CMD go run ./cmd/server` を使う（`air` または ビルド済みバイナリを使う）。
