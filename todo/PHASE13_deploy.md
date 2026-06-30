# PHASE 13 — デプロイ・本番設定

> 目標: フロントエンドを Firebase Hosting に、バックエンドを Render または Cloud Run にデプロイして本番環境を稼動させる。

---

## 事前準備

- [ ] Firebase プロジェクトを作成（本番用）
- [ ] Firebase Storage のバケットを作成し、公開ルールを設定
- [ ] Firebase Auth で Google プロバイダーを有効化
- [ ] PostgreSQL 本番データベースを用意（Render PostgreSQL / Cloud SQL / Supabase など）
- [ ] 本番用の JWT シークレットを生成（32 バイト以上のランダム文字列）

---

## バックエンド デプロイ

### 本番用 Dockerfile 確認（`backend/Dockerfile`）

- [ ] マルチステージビルドで静的バイナリを生成
- [ ] `CGO_ENABLED=0 GOOS=linux` でビルド
- [ ] 最終イメージは `alpine:latest`（軽量化）

### Render へのデプロイ（選択肢 A）

- [ ] Render に新しい Web Service を作成
- [ ] GitHub リポジトリを連携
- [ ] `backend/` をルートディレクトリに設定
- [ ] 環境変数を Render のダッシュボードで設定
  - `DATABASE_URL`
  - `JWT_SECRET` / `JWT_REFRESH_SECRET`
  - `FIREBASE_CREDENTIALS_JSON`
  - `CORS_ORIGINS=https://<firebase-app>.web.app`
  - `ENV=production`
  - `PORT=8080`
- [ ] ヘルスチェックパス: `/health`
- [ ] デプロイ後に `https://<render-app>.onrender.com/health` が 200 を返すことを確認

### Google Cloud Run へのデプロイ（選択肢 B）

- [ ] `gcloud builds submit --tag gcr.io/<PROJECT_ID>/sns-backend ./backend`
- [ ] Secret Manager に JWT シークレット・Firebase 認証情報を登録
- [ ] Cloud Run サービスを作成してシークレットを環境変数としてマウント
- [ ] Cloud SQL Auth Proxy の設定（Cloud SQL 使用時）

---

## フロントエンド デプロイ

### 環境変数（本番用 `.env.production`）

- [ ] `VITE_API_BASE_URL=https://<backend-url>`
- [ ] `VITE_FIREBASE_API_KEY=...`
- [ ] `VITE_FIREBASE_AUTH_DOMAIN=...`
- [ ] `VITE_FIREBASE_PROJECT_ID=...`
- [ ] `VITE_FIREBASE_STORAGE_BUCKET=...`
- [ ] `VITE_FIREBASE_APP_ID=...`

### Firebase Hosting 設定

- [ ] `firebase login` & `firebase init hosting`
- [ ] `firebase.json` で SPA のリダイレクト設定
  ```json
  {
    "hosting": {
      "public": "frontend/dist",
      "rewrites": [{ "source": "**", "destination": "/index.html" }]
    }
  }
  ```
- [ ] `npm run build` でビルドが成功することを確認
- [ ] `firebase deploy --only hosting` でデプロイ

### Firebase Auth のドメイン設定

- [ ] Firebase コンソール → Authentication → Authorized domains に本番ドメインを追加
- [ ] Google OAuth のリダイレクト URI を更新

---

## 本番環境チェックリスト

### セキュリティ

- [ ] Swagger UI が本番環境で無効化されている（`ENV=production` で非公開）
- [ ] CORS が Firebase Hosting のドメインのみ許可
- [ ] JWT シークレットが十分な長さのランダム文字列
- [ ] HTTPS のみ（Render / Cloud Run / Firebase Hosting はデフォルトで HTTPS）
- [ ] `.env` ファイルがリポジトリに含まれていない

### 動作確認

- [ ] サインアップ・ログインが動作する
- [ ] Google OAuth が動作する
- [ ] 投稿・画像アップロードが Firebase Storage に保存される
- [ ] タイムラインが表示される
- [ ] スマホ・PC 両方のレスポンシブ表示を確認

### パフォーマンス

- [ ] Vite のビルド最適化（コードスプリッティング）を確認
- [ ] 画像の lazy loading が有効
- [ ] DB のインデックスが全て適用済み

---

## CI/CD（オプション）

- [ ] GitHub Actions で PR のビルド・テストを自動実行
- [ ] `main` ブランチにマージで本番自動デプロイ
  - フロントエンド: `firebase deploy`
  - バックエンド: Render の自動デプロイまたは `gcloud run deploy`

---

## 完了基準

- [ ] `https://<firebase-app>.web.app` でアプリが表示される
- [ ] 本番環境でフルフローが動作する（登録 → ログイン → 投稿 → いいね）
- [ ] スマホブラウザで操作して UI が崩れない
- [ ] Swagger UI が本番で表示されない
