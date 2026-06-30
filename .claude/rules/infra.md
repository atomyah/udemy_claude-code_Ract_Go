# インフラ・デプロイ規約

## ローカル開発環境

### Docker / docker-compose 構成

```yaml
# docker-compose.yml の構成方針
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: sns_db
      POSTGRES_USER: sns_user
      POSTGRES_PASSWORD: sns_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev     # air を使う開発用 Dockerfile
    volumes:
      - ./backend:/app               # ホットリロードのためにマウント
      - /app/tmp                     # air のビルド出力は除外
    environment:
      DATABASE_URL: postgres://sns_user:sns_password@db:5432/sns_db
      ENV: development
    ports:
      - "8080:8080"
    depends_on:
      - db
    command: air                     # go run ではなく air を使う

volumes:
  postgres_data:
```

### 開発用 Dockerfile（バックエンド）

```dockerfile
# backend/Dockerfile.dev
FROM golang:1.22-alpine
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
# ソースはボリュームマウントで提供するためここではコピー不要
CMD ["air"]
```

### 起動手順

```bash
# DB のみ起動（フロントは npm run dev で）
docker-compose up -d db
cd backend && air

# フロントエンド（別ターミナル）
cd frontend && npm run dev

# 全体まとめて起動
docker-compose up
```

### ポート一覧

| サービス | ポート |
|---------|-------|
| フロントエンド（Vite dev） | 5173 |
| バックエンド（Echo） | 8080 |
| PostgreSQL | 5432 |

---

## 環境変数管理

### ローカル

- `backend/.env` にローカル設定を書く（`.gitignore` に追加して絶対コミットしない）。
- `frontend/.env.local` に `VITE_API_BASE_URL=http://localhost:8080` などを書く。

### 本番

- 環境変数はデプロイ先（Render / Cloud Run / Firebase）の管理画面で設定する。
- シークレット（JWT シークレット、Firebase credentials など）はソースコードに絶対含めない。

### 必須環境変数（バックエンド）

```
DATABASE_URL=postgres://user:pass@host:5432/dbname
JWT_SECRET=<長いランダム文字列>
JWT_REFRESH_SECRET=<別の長いランダム文字列>
FIREBASE_CREDENTIALS_JSON=<サービスアカウント JSON の内容>
PORT=8080
ENV=development  # development | production
```

---

## 本番デプロイ

### フロントエンド → Firebase Hosting

```bash
cd frontend
npm run build              # dist/ を生成
firebase deploy --only hosting
```

- `firebase.json` でルーティング（SPA の `rewrites` 設定）を必ず設定する。
- 環境変数は `.env.production` または CI/CD の Secrets で管理する。

### バックエンド → Render または Google Cloud Run

#### Render の場合

- `backend/Dockerfile`（本番用）を使ってコンテナイメージをビルド。
- 環境変数は Render のダッシュボードで設定。
- ヘルスチェックエンドポイント `GET /health` を実装する。

#### Cloud Run の場合

```bash
gcloud builds submit --tag gcr.io/<PROJECT_ID>/sns-backend ./backend
gcloud run deploy sns-backend \
  --image gcr.io/<PROJECT_ID>/sns-backend \
  --platform managed \
  --region asia-northeast1 \
  --set-env-vars DATABASE_URL=...
```

### 本番用 Dockerfile（バックエンド）

```dockerfile
# backend/Dockerfile（本番用マルチステージビルド）
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]
```

---

## `.gitignore` で必ず除外するもの

```
# バックエンド
backend/.env
backend/tmp/
backend/docs/  # swaggo 生成ファイルはオプション（チームで方針決定）

# フロントエンド
frontend/.env.local
frontend/.env.*.local
frontend/dist/
frontend/node_modules/

# Firebase
*.serviceAccountKey.json
firebase-credentials.json

# その他
*.log
```

---

## ヘルスチェック

バックエンドに以下のエンドポイントを実装する（認証不要）:

```
GET /health
→ 200 OK: { "status": "ok" }
```

Render / Cloud Run のヘルスチェック設定でこのエンドポイントを指定する。

---

## CORS 設定

- 開発環境: `http://localhost:5173` を許可。
- 本番環境: Firebase Hosting のドメインのみ許可。
- Echo の CORS ミドルウェアで環境変数からオリジンを設定する。

```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: strings.Split(os.Getenv("CORS_ORIGINS"), ","),
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
    AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
}))
```
