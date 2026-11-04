# デプロイ手順

本番環境の構成と、初回セットアップから日常のリリースまでの手順をまとめる。

| 対象 | デプロイ先 | 方法 | 関連ファイル |
|------|-----------|------|-------------|
| バックエンド（Go / Echo） | Google Cloud Run | GitHub 連携の継続デプロイ（`main` への push で自動） | `backend/Dockerfile.prod`, `backend/.dockerignore` |
| フロントエンド（React / Vite） | Firebase Hosting | `make deploy-frontend`（手動） | `frontend/scripts/deploy.sh`, `frontend/.env.production`, `frontend/firebase.json`, `frontend/.firebaserc` |
| DB | Neon（サーバーレス PostgreSQL 16） | マイグレーションは golang-migrate で手動適用 | `backend/migrations/` |
| ファイル | Firebase Storage | バックエンドが Admin SDK でアップロード | — |

```
ブラウザ ──▶ Firebase Hosting（https://<project>.web.app）  静的ファイル（dist/）
   │
   └──(Cookie 付き CORS)──▶ Cloud Run（https://sns-backend-xxxx.asia-northeast1.run.app/api/v1）
                                 ├──▶ Neon（PostgreSQL 16。プール接続・TLS 必須のインターネット接続）
                                 └──▶ Firebase Storage / Firebase Auth（Admin SDK）
```

---

## 1. 前提条件

### アカウント・権限

- Google Cloud プロジェクト（Firebase プロジェクト `udemy-claudecode-react-go` と同じプロジェクトを使う）。課金を有効にしておく。
- 作業者の Google アカウントにプロジェクトの「オーナー」または以下のロール:
  `Cloud Run 管理者` / `Cloud Build 編集者` / `Artifact Registry 管理者` / `Secret Manager 管理者` / `サービス アカウント ユーザー` / `Firebase Hosting 管理者`
- GitHub リポジトリの管理者権限（Cloud Build の GitHub アプリをインストールするため）。
- Neon のアカウント（https://neon.tech 。GitHub / Google アカウントでサインアップできる）。

### 有効化する API

Google Cloud コンソールの「API とサービス」または以下のコマンドで有効化する。

```bash
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  secretmanager.googleapis.com
```

### ローカルに必要なツール

| ツール | 用途 |
|-------|------|
| `gcloud` CLI | シークレット登録・動作確認（UI だけでも作業可能） |
| Node.js 20 / npm | フロントエンドのビルド |
| Firebase CLI | Hosting へのデプロイ。未インストールなら `deploy.sh` が `npx firebase-tools@15.34.2` を使う |
| Docker | 本番 DB へのマイグレーション（`migrate/migrate` イメージ）・本番イメージのローカル確認 |
| make（Git Bash） | `make deploy-frontend` |

---

## 2. ブランチ運用

```
feature/xxx ──PR──▶ develop ──PR──▶ main ──(自動)──▶ Cloud Run（本番）
                                     └──(手動)──▶ make deploy-frontend ──▶ Firebase Hosting
```

- **`main` = 本番**。Cloud Run の継続デプロイは `main` ブランチ（正規表現 `^main$`）への push だけで動く。
- 開発は `feature/*` / `fix/*` → `develop` に PR でマージする（`rules/git.md`）。`develop` へのマージではデプロイされない。
- リリース時は `develop` → `main` の PR を作り、**CI（`.github/workflows/ci.yml`）がすべてグリーン**で、`/security-review` の結果を `docs/security-reviews/YYYY-MM-DD.md` に保存してからマージする。
- マージで Cloud Run のビルド・デプロイが自動で始まる。フロントエンドはバックエンドのデプロイ完了を確認してから `main` をチェックアウトして `make deploy-frontend` を実行する（API の変更とフロントの変更の順序をそろえるため）。
- DB スキーマの変更を含むリリースは、**マージ前に**本番 DB へマイグレーションを適用する（後方互換のある変更にしておく。[4. DB の準備とマイグレーション](#4-db-の準備とマイグレーション)）。

---

## 3. 初回デプロイの流れ

フロントエンドとバックエンドが互いの URL を必要とするため、次の順序で行う。

1. DB を作成し、マイグレーションを適用する（[4](#4-db-の準備とマイグレーション)）
2. シークレットを Secret Manager に登録する（[5](#5-シークレットの登録secret-manager)）
3. Cloud Run の継続デプロイを設定し、バックエンドをデプロイする。`CORS_ORIGINS` には Firebase Hosting の既定 URL（`https://<project>.web.app`）を先に入れておく（[6](#6-バックエンドcloud-run-の継続デプロイ設定)）
4. 発行された Cloud Run の URL を `frontend/.env.production` の `VITE_API_BASE_URL` に書き、`make deploy-frontend` を実行する（[8](#8-フロントエンドfirebase-hosting)）
5. ブラウザで動作確認する（[9](#9-動作確認)）

---

## 4. DB の準備とマイグレーション

### Neon プロジェクトの作成（UI）

1. Neon コンソール >「New Project」
2. Postgres version: **16**、Region: Cloud Run のリージョンにできるだけ近いものを選ぶ（東京が選べれば東京、なければ **AWS Asia Pacific (Singapore)**）
   - DB が遠いとクエリ 1 本ごとに往復の遅延が乗り、一覧 API 500ms 以内（非機能要件）を守りにくい。Singapore を選んだ場合は Cloud Run も **`asia-southeast1`（シンガポール）** にそろえる
3. Database name: `sns_db`（ロールは作成時のものを使うか、「Roles」で `sns_user` を作る）
4. Dashboard の **「Connect」** で接続文字列を 2 種類控える

   | 種類 | Connection pooling | ホスト名 | 用途 |
   |------|------|---------|------|
   | プール接続 | オン | `ep-xxxx-pooler.<region>.aws.neon.tech` | **アプリ（Cloud Run の `DATABASE_URL`）** |
   | 直接接続 | オフ | `ep-xxxx.<region>.aws.neon.tech` | **マイグレーション**（golang-migrate のアドバイザリロックはプーラー経由だと効かない） |

コピーした接続文字列は次のように整える。

```text
# Neon でコピーされる形
postgresql://sns_user:<PASSWORD>@ep-xxxx-pooler.ap-southeast-1.aws.neon.tech/sns_db?sslmode=require&channel_binding=require

# このアプリで使う形（channel_binding を外し、connect_timeout=5 を足す）
postgresql://sns_user:<PASSWORD>@ep-xxxx-pooler.ap-southeast-1.aws.neon.tech/sns_db?sslmode=require&connect_timeout=5
```

- `sslmode=require` は必須（Neon は TLS 以外の接続を受け付けない）。
- `channel_binding=require` は外す。利用中の pgx（v5.6.0）は対応しておらず、未知のパラメーターとしてサーバーに送られて接続に失敗する。
- `connect_timeout=5` は非機能要件の DB 接続タイムアウト。Neon はアイドル時にコンピュートが停止（scale to zero）し、再開に数百 ms〜数秒かかるが 5 秒以内に収まる。

### マイグレーションの適用

本番では `AutoMigrate` が動かない（`config.ShouldAutoMigrate()` が false）、`cmd/migrate` も development / test 以外では実行を拒否する。
本番 DB には `backend/migrations/*.sql` を **golang-migrate** で、**直接接続（`-pooler` なし）の URL** に対して適用する。

```bash
# リポジトリのルートで実行。<DIRECT_URL> は上の「直接接続」の接続文字列（sslmode=require 付き、channel_binding は外す）
docker run --rm -v "$(pwd)/backend/migrations:/migrations" migrate/migrate:v4.18.1 \
  -path=/migrations -database "<DIRECT_URL>" up

# 現在のバージョン確認 / 1 つ戻す
docker run --rm -v "$(pwd)/backend/migrations:/migrations" migrate/migrate:v4.18.1 -path=/migrations -database "<DIRECT_URL>" version
docker run --rm -v "$(pwd)/backend/migrations:/migrations" migrate/migrate:v4.18.1 -path=/migrations -database "<DIRECT_URL>" down 1
```

- Git Bash で `$(pwd)` が `/c/Users/...` 形式になり失敗する場合は、`MSYS_NO_PATHCONV=1` を先頭に付ける。
- Neon の **ブランチ**機能で本番 DB のコピーを作り、先にそちらでマイグレーションを試してから本番に適用すると安全。

---

## 5. シークレットの登録（Secret Manager）

秘密情報は Cloud Run の環境変数に直接書かず、Secret Manager に登録して参照させる。
`gcloud` CLI がなくても、値をローカルで作って Google Cloud コンソールに貼り付ければよい。

### 登録するシークレット

| シークレット名 | 値の作り方 | 参照する環境変数 |
|--------------|-----------|----------------|
| `jwt-secret` | `openssl rand -hex 32`（64 文字） | `JWT_SECRET` |
| `jwt-refresh-secret` | `openssl rand -hex 32`（**別に**もう一度実行） | `JWT_REFRESH_SECRET` |
| `admin-session-secret` | `openssl rand -hex 32`（**別に**もう一度実行） | `ADMIN_SESSION_SECRET` |
| `admin-root-password` | `openssl rand -hex 32`（管理画面のログインで使うので控えておく。12 文字以上なら自分で決めた値でもよい） | `ADMIN_ROOT_PASSWORD` |
| `database-url` | Neon の接続文字列を 4 章のとおり整えたもの | `DATABASE_URL` |
| `firebase-credentials-json` | Firebase のサービスアカウント JSON ファイル（ファイルごとアップロード） | `FIREBASE_CREDENTIALS_JSON` |

`jwt-secret` / `jwt-refresh-secret` / `admin-session-secret` はすべて異なる値にする（同じだと起動時にエラーになる）。

### コンソールでの登録手順（gcloud 不要）

1. Git Bash で値を作る（1 回実行するごとに別の値が出る）
   ```bash
   openssl rand -hex 32
   ```
2. Google Cloud コンソール > **Secret Manager** >「シークレットを作成」
3. 名前に上の表のシークレット名を入れる
4. 「シークレットの値」に 1 の出力を貼り付ける。**末尾に改行や空白を入れない**（値の一部として保存されてしまう）
   - `firebase-credentials-json` だけは「アップロード ファイル」で JSON ファイルを選ぶ
5. 「シークレットを作成」。表のシークレットの数だけ繰り返す

### gcloud CLI を使う場合

```bash
openssl rand -hex 32 | tr -d '\n' | gcloud secrets create jwt-secret --data-file=-
openssl rand -hex 32 | tr -d '\n' | gcloud secrets create jwt-refresh-secret --data-file=-
openssl rand -hex 32 | tr -d '\n' | gcloud secrets create admin-session-secret --data-file=-
printf '%s' '<ROOT_ADMIN_PASSWORD>' | gcloud secrets create admin-root-password --data-file=-
printf '%s' '<4 章で整えた DATABASE_URL>' | gcloud secrets create database-url --data-file=-
gcloud secrets create firebase-credentials-json --data-file=udemy-claudecode-react-go-firebase-adminsdk.json
```

Cloud Run の実行サービスアカウント（既定は `PROJECT_NUMBER-compute@developer.gserviceaccount.com`）に、各シークレットの **Secret Manager のシークレット アクセサー**（`roles/secretmanager.secretAccessor`）を付与する。
Cloud Run の UI でシークレットを参照すると、権限が足りない場合は付与を促す表示が出る。

値を変えるときは新しいバージョンを追加し（コンソールではシークレットの詳細 >「新しいバージョン」、CLI では `gcloud secrets versions add <name> --data-file=-`）、Cloud Run で新しいリビジョンをデプロイする（参照を `latest` にしていても、起動中のインスタンスには反映されない）。

---

## 6. バックエンド（Cloud Run）の継続デプロイ設定

### Dockerfile.prod の構成

- マルチステージビルド。ビルドステージ（`golang:1.24-alpine`）で静的バイナリを作り、実行ステージ（`gcr.io/distroless/static-debian12:nonroot`）にはバイナリ 1 つだけを入れる（約 57MB、シェルなし・非 root 実行）。
- 管理画面のテンプレート・静的ファイルは `go:embed`、Swagger 定義（アクセスログの `action` に使用）は `docs` パッケージとしてバイナリに含まれる。
- 環境変数・シークレットはイメージに含めない（`backend/.dockerignore` で `.env*` とサービスアカウント JSON を除外）。`ENV=production` だけイメージの既定値にしている。
- 待ち受けポートは Cloud Run が注入する `PORT` を `config.Load()` が読む。起動コマンドは `CMD ["/server"]` で完結するため、**Cloud Run 側のコンテナコマンド・引数は空欄のまま**にする。

ローカルでの確認:

```bash
cd backend
docker build -f Dockerfile.prod -t sns-backend:local .
docker run --rm -p 9090:9090 --network udemy-claudecode_reactgo_default --env-file .env \
  -e ENV=production -e PORT=9090 -e CORS_ORIGINS=https://example.web.app sns-backend:local
curl http://localhost:9090/health   # → {"status":"ok"}
```

### Cloud Run UI での設定手順

1. Google Cloud コンソール > **Cloud Run** >「サービスを作成」
2. **「リポジトリから継続的にデプロイする（GitHub、GitLab、Bitbucket）」** を選び、「Cloud Build を設定」を押す
   1. リポジトリ プロバイダ: **GitHub**。初回は GitHub 認証と「Cloud Build GitHub App」のインストールを求められるので、このリポジトリだけを許可する
   2. リポジトリ: このリポジトリを選択
   3. ブランチ: **`^main$`**
   4. ビルドタイプ: **Dockerfile**、ソースの場所: **`/backend/Dockerfile.prod`**（ビルドコンテキストは `backend/` になる）
   5. 「保存」
3. サービスの設定
   | 項目 | 値 |
   |------|-----|
   | サービス名 | `sns-backend` |
   | リージョン | Neon のリージョンに近い場所（Neon が Singapore なら `asia-southeast1`、東京なら `asia-northeast1`） |
   | 認証 | **公開アクセスを許可する**（未認証の呼び出しを許可。API 自体が JWT で保護している） |
   | 課金 | リクエスト ベース |
   | 最小インスタンス数 | `0`（コールドスタートを避けたい場合は `1`） |
   | 最大インスタンス数 | `1`〜`3`（レート制限がインスタンスごとのメモリ管理のため、増やすと実効上限が増える） |
   | Ingress | すべて |
4. 「コンテナ、ボリューム、ネットワーキング、セキュリティ」を開く
   - **コンテナ** タブ
     - コンテナポート: `8080`（Cloud Run がこの値を `PORT` として注入する。`PORT` を環境変数に自分で追加しない）
     - コンテナ コマンド / コンテナ引数: **空欄**（Dockerfile の `CMD` を使う）
     - メモリ `512MiB`、CPU `1`
     - リクエストのタイムアウト: **`30` 秒**（非機能要件）
     - ヘルスチェック: 起動プローブを HTTP `/health`（ポート 8080）に設定する
   - **変数とシークレット** タブ: [7.1 バックエンドの環境変数](#71-バックエンドcloud-run) の表のとおり登録する
   - Neon へはインターネット経由（TLS）で接続するため、Cloud SQL 接続や VPC の設定は不要
5. 「作成」を押すと、Cloud Build が `main` の最新コミットでイメージをビルドし、Artifact Registry（`cloud-run-source-deploy`）に push してデプロイする
6. 発行された URL（`https://sns-backend-xxxxxxxxxx.asia-northeast1.run.app`）を控える

以降は `main` に push するたびに自動でビルド・デプロイされる。ビルドの状況は **Cloud Build > 履歴**、デプロイ履歴は Cloud Run の「リビジョン」タブで確認する。

### ロールバック

Cloud Run > `sns-backend` >「リビジョン」タブ > 戻したいリビジョンの「…」>「トラフィックを管理」で 100% を割り当てる。
その後、問題のコミットを `main` で revert して push する（次のデプロイで再び壊れたリビジョンにならないように）。

---

## 7. 環境変数一覧

本番で使う環境変数をすべてここにまとめる。設定する場所は 3 つある。

| 設定場所 | 対象 | 読み込まれるタイミング | 秘密情報 |
|---------|------|----------------------|---------|
| Cloud Run の「変数とシークレット」（＋ Secret Manager） | バックエンド | コンテナ起動時 | 可（Secret Manager 経由） |
| `frontend/.env.production`（コミットする） | フロントエンド | `npm run build` 時に JS に埋め込み | **不可**（ブラウザから誰でも読める） |
| `make deploy-frontend` 実行時のシェル | デプロイスクリプト | デプロイ実行時 | — |

凡例: 必須 ○ = 必ず設定 / △ = 管理画面を使うなら設定 / 空欄 = 任意（既定値あり）

### 7.1 バックエンド（Cloud Run）

| 変数 | 必須 | 種類 | Secret Manager 名 | 本番の値 / 説明 |
|------|:----:|------|------------------|----------------|
| `ENV` | ○ | 変数 | — | `production`（イメージの既定値。明示しておくとよい）。Swagger UI 非公開・AutoMigrate 無効・SQL ログなしになる |
| `DATABASE_URL` | ○ | シークレット | `database-url` | Neon の**プール接続**の URL。`postgresql://USER:PASS@ep-xxxx-pooler.<region>.aws.neon.tech/sns_db?sslmode=require&connect_timeout=5`（`channel_binding` は付けない） |
| `JWT_SECRET` | ○ | シークレット | `jwt-secret` | 32 文字以上のランダム値。未設定・短い・見本のまま・`JWT_REFRESH_SECRET` と同じだと起動しない |
| `JWT_REFRESH_SECRET` | ○ | シークレット | `jwt-refresh-secret` | 同上（`JWT_SECRET` と別の値） |
| `FIREBASE_CREDENTIALS_JSON` | ○ | シークレット | `firebase-credentials-json` | Firebase Admin SDK のサービスアカウント JSON（ファイルの中身そのもの） |
| `FIREBASE_STORAGE_BUCKET` | ○ | 変数 | — | `udemy-claudecode-react-go.firebasestorage.app` |
| `MEDIA_SIGNED_URL_TTL_MINUTES` | | 変数 | — | 投稿メディアの署名付き URL の有効期限（分）。既定 `60`、1〜10080 |
| `CORS_ORIGINS` | ○ | 変数 | — | `https://udemy-claudecode-react-go.web.app,https://udemy-claudecode-react-go.firebaseapp.com`（カンマ区切り。CSRF 対策の Origin 検証にも使う。`*` は不可） |
| `FRONTEND_BASE_URL` | ○ | 変数 | — | `https://udemy-claudecode-react-go.web.app`（パスワードリセットリンクの URL。未設定時は `CORS_ORIGINS` の先頭） |
| `ADMIN_SESSION_SECRET` | △ | シークレット | `admin-session-secret` | 管理画面（`/admin`）の署名鍵。32 文字以上で JWT の鍵と別の値。**未設定なら管理画面は無効** |
| `ADMIN_ROOT_USERNAME` | △ | 変数 | — | ルート管理者のユーザー名（`ADMIN_ROOT_PASSWORD` とセット） |
| `ADMIN_ROOT_PASSWORD` | △ | シークレット | `admin-root-password` | 12 文字以上 |
| `PASSWORD_RESET_TOKEN_TTL_MINUTES` | | 変数 | — | パスワードリセットリンクの有効期限（分）。既定 `1440`、10〜10080 |
| `TRUSTED_PROXY_HOPS` | | 変数 | — | 本番の既定 `1`（Cloud Run の Google Front End）。前段に外部ロードバランサーを置くときは `2` |
| `LOG_LEVEL` | | 変数 | — | 既定 `info`。`debug` / `warn` / `error` |
| `TZ` | | 変数 | — | `Asia/Tokyo` を推奨。未設定だと Cloud Run では UTC。アプリのコードではなく Go の `time` パッケージが読み、ログの `time` などローカル時刻の表示に効く（実行イメージの distroless はタイムゾーンデータを含む）。DB の日時は `TIMESTAMP WITH TIME ZONE` で保存され、管理画面の日別集計は SQL で `Asia/Tokyo` 固定のため、この値には影響されない |
| `PORT` | — | 自動 | — | **設定しない**。Cloud Run がコンテナポート（8080）を注入する |

本番では効かない（設定しても無視される）変数: `COOKIE_SECURE`（開発のみ）、`AUTO_APPROVE_USERS` / `RATE_LIMIT_*_PER_MINUTE` / `SQL_LOG`（テストのみ）、`POSTGRES_*`（docker compose 用）。

**DB の接続情報は `DATABASE_URL` の 1 本だけ**で渡す。`DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` のような個別の変数はアプリが読まないため、設定しても使われない。各要素は URL の中に書く。

```text
postgresql://<DB_USER>:<DB_PASSWORD>@<DB_HOST>:<DB_PORT>/<DB_NAME>?sslmode=require&connect_timeout=5
              sns_user   (Neon のパスワード)  ep-xxxx-pooler.<region>.aws.neon.tech  5432（省略可）  sns_db
```

- パスワードに `@` `:` `/` `?` `#` `%` などの記号が含まれる場合は URL エンコードする（例: `@` → `%40`）。Neon が自動生成するパスワードは英数字のみなので通常は不要。
- URL 全体を Secret Manager の `database-url` に入れる（パスワードを含むため通常の環境変数にはしない）。

**設定方法**

- **UI**: Cloud Run > `sns-backend` >「新しいリビジョンの編集とデプロイ」> **変数とシークレット** タブ
  - 「種類: 変数」は「環境変数」の「変数を追加」で名前と値を入れる
  - 「種類: シークレット」は「シークレットを環境変数として参照」で、上の表の Secret Manager 名とバージョン（`latest`）を選ぶ（シークレットの作成は [5](#5-シークレットの登録secret-manager)）
  - 「デプロイ」で新しいリビジョンが作られて反映される
- **CLI**:
  ```bash
  gcloud run services update sns-backend --region asia-northeast1 \
    --update-env-vars "ENV=production,TZ=Asia/Tokyo,FIREBASE_STORAGE_BUCKET=udemy-claudecode-react-go.firebasestorage.app" \
    --update-env-vars "^@^CORS_ORIGINS=https://udemy-claudecode-react-go.web.app,https://udemy-claudecode-react-go.firebaseapp.com@FRONTEND_BASE_URL=https://udemy-claudecode-react-go.web.app@ADMIN_ROOT_USERNAME=root" \
    --update-secrets "DATABASE_URL=database-url:latest,JWT_SECRET=jwt-secret:latest,JWT_REFRESH_SECRET=jwt-refresh-secret:latest" \
    --update-secrets "FIREBASE_CREDENTIALS_JSON=firebase-credentials-json:latest,ADMIN_SESSION_SECRET=admin-session-secret:latest,ADMIN_ROOT_PASSWORD=admin-root-password:latest"
  ```
  （`CORS_ORIGINS` の値にカンマが入るため、2 行目は `^@^` で区切り文字を `@` に変えている）
- 継続デプロイで作られるリビジョンは、直前のリビジョンの環境変数・シークレット設定を引き継ぐ。コードを変えるたびに設定し直す必要はない。
- 現在の設定の確認: `gcloud run services describe sns-backend --region asia-northeast1 --format "yaml(spec.template.spec.containers[0].env)"`

### 7.2 フロントエンド（`frontend/.env.production`）

`npm run build`（`vite build --mode production`）のときに読み込まれ、値が JS に埋め込まれる。
**ブラウザから誰でも読める値なので秘密情報は書かない**（Firebase Web 設定は公開前提の値）。このファイルはコミットする。

| 変数 | 必須 | 本番の値 / 取得元 |
|------|:----:|----------------|
| `VITE_API_BASE_URL` | ○ | Cloud Run の URL + **`/api/v1`**（例: `https://sns-backend-123456789012.asia-northeast1.run.app/api/v1`）。URL は Cloud Run のサービス詳細画面の上部に表示される |
| `VITE_FIREBASE_API_KEY` | ○ | Firebase コンソール > プロジェクトの設定 > マイアプリ（ウェブアプリ）の `apiKey` |
| `VITE_FIREBASE_AUTH_DOMAIN` | ○ | 同 `authDomain`（`udemy-claudecode-react-go.firebaseapp.com`） |
| `VITE_FIREBASE_PROJECT_ID` | ○ | 同 `projectId`（`udemy-claudecode-react-go`） |
| `VITE_FIREBASE_STORAGE_BUCKET` | ○ | 同 `storageBucket`（バックエンドの `FIREBASE_STORAGE_BUCKET` と同じ値） |
| `VITE_FIREBASE_MESSAGING_SENDER_ID` | ○ | 同 `messagingSenderId` |
| `VITE_FIREBASE_APP_ID` | ○ | 同 `appId` |

- `VITE_API_BASE_URL` が空・見本（`REPLACE_ME`）のまま・`https://…/api/v1` の形でないときは、`make deploy-frontend` がビルド前にエラーで止まる。
- Vite の優先順位は `.env.production.local` > `.env.production` > `.env.local` > `.env`。開発用 `.env.local` の値は本番ビルドに使われない。一時的に別の値で試すときは `.env.production.local`（gitignore 済み）を作る。
- 値を変えたら再ビルド・再デプロイ（`make deploy-frontend`）が必要（実行時には読み込まれない）。

### 7.3 デプロイスクリプト（`make deploy-frontend`）

| 変数 | 既定 | 説明 |
|------|------|------|
| `FIREBASE_PROJECT` | `default`（`.firebaserc` の `udemy-claudecode-react-go`） | デプロイ先の Firebase プロジェクト |
| `SKIP_INSTALL` | なし | `1` で `npm ci` を省略する |

```bash
make deploy-frontend FIREBASE_PROJECT=udemy-claudecode-react-go SKIP_INSTALL=1
```

### 7.4 値をそろえる必要がある組み合わせ

片方だけ変えると動かなくなるため、変更するときは両方を更新する。

| 変えたもの | 合わせて更新するもの | ずれたときの症状 |
|-----------|-------------------|----------------|
| Cloud Run の URL（サービス名・リージョンの変更、独自ドメイン化） | `VITE_API_BASE_URL` → `make deploy-frontend` | API に接続できない |
| Firebase Hosting の URL（独自ドメイン化） | `CORS_ORIGINS`・`FRONTEND_BASE_URL`（Cloud Run） | CORS エラー、更新系 API が 403 `INVALID_ORIGIN`、リセットリンクの URL が古い |
| Storage バケット | `FIREBASE_STORAGE_BUCKET` と `VITE_FIREBASE_STORAGE_BUCKET` | アップロード失敗 |
| `JWT_SECRET` / `JWT_REFRESH_SECRET` | （単独で変更可） | 全ユーザーが再ログインになる |

### 7.5 開発・テスト環境との違い

| 環境 | バックエンド | フロントエンド |
|------|------------|--------------|
| 開発 | `backend/.env`（`.env.example` をコピー。コミットしない） | `frontend/.env.local`（コミットしない） |
| テスト / CI | `backend/.env.test`、`.github/workflows/ci.yml` の `env` | `frontend/.env.e2e` |
| 本番 | Cloud Run ＋ Secret Manager（7.1） | `frontend/.env.production`（7.2） |

---

## 8. フロントエンド（Firebase Hosting）

### 初回の準備

1. Firebase コンソール > **Hosting** >「始める」で Hosting を有効にする（サイト `udemy-claudecode-react-go` が作られる）
2. Firebase CLI にログインする
   ```bash
   npx firebase-tools@15.34.2 login   # グローバルに入れている場合は firebase login
   ```
3. Firebase コンソール > **Authentication** > 設定 >「承認済みドメイン」に Hosting のドメイン（`*.web.app` / `*.firebaseapp.com`）が含まれていることを確認する（Google ログインのポップアップに必要）

`frontend/firebase.json` は `dist/` を公開し、SPA のため全パスを `index.html` に書き換える。ハッシュ付きの `/assets/**` は 1 年キャッシュ、HTML は `no-cache`。
`frontend/.firebaserc` の `default` がデプロイ先プロジェクト。

### フロントエンドの環境変数

`frontend/.env.production` に書く。変数の一覧と取得元は [7.2](#72-フロントエンドfrontendenvproduction) を参照。

### デプロイ

```bash
make deploy-frontend
# デプロイ先を変える / npm ci を省く
make deploy-frontend FIREBASE_PROJECT=<project-id> SKIP_INSTALL=1
```

`frontend/scripts/deploy.sh`（`npm run deploy` でも同じ）が次を順に行い、どこかで失敗したら止まる。

1. `.env.production` の検証（`VITE_API_BASE_URL` が空・見本のまま・`https://…/api/v1` 以外ならエラー）
2. `npm ci`
3. `npm run gen:api`（`backend/docs/openapi.yaml` から API 型を生成）
4. `npm run build -- --mode production`（型チェック + `dist/` 生成）
5. `firebase deploy --only hosting --project <default>`

ロールバックは Firebase コンソール > Hosting >「リリース履歴」から以前のバージョンを「ロールバック」する。

---

## 9. 動作確認

```bash
curl https://sns-backend-xxxxxxxxxx.asia-northeast1.run.app/health        # → {"status":"ok"}
curl -o /dev/null -w '%{http_code}\n' https://sns-backend-xxxxxxxxxx.asia-northeast1.run.app/swagger/index.html  # → 404（本番は非公開）
```

ブラウザで `https://udemy-claudecode-react-go.web.app` を開き、次を確認する。

- [ ] 新規登録 → 「承認待ち」になる → 管理画面（`https://<Cloud Run URL>/admin`）で承認 → ログインできる
- [ ] 画像付きの投稿ができ、画像が表示される（Firebase Storage）
- [ ] リロードしてもログイン状態が保たれる（Cookie が送られている）
- [ ] Google ログインができる
- [ ] Cloud Run の「ログ」に JSON のアクセスログ（`message: "access"`）が出ている

---

## 10. 注意事項

- **クロスサイト Cookie**: フロント（`*.web.app`）と API（`*.run.app`）は別サイトのため、認証 Cookie は `SameSite=None; Secure` のサードパーティ Cookie として送られる。Safari（ITP）やサードパーティ Cookie をブロックする設定のブラウザではログインが維持できない。対応が必要になったら、独自ドメインで `app.example.com`（Hosting）と `api.example.com`（Cloud Run のドメインマッピング）のように**同じサイト**にそろえ、`CORS_ORIGINS` / `VITE_API_BASE_URL` / `FRONTEND_BASE_URL` を更新する。
- Firebase Hosting の rewrites で Cloud Run に中継する方法（同一オリジン化）は、Hosting が `__session` 以外の Cookie を削除するため、この構成では使えない。
- **リクエストサイズ**: Cloud Run の HTTP/1.1 リクエストは 32MiB が上限。これを超える動画はアプリの上限（50MB）より前に Cloud Run で拒否される。
- **Neon の scale to zero**: アイドルが続くとコンピュートが停止し、その後の最初のリクエストは DB の再開分だけ遅くなる。常に速くしたい場合は Neon の有料プランで自動停止を無効にする（Cloud Run の最小インスタンス数を 1 にしても DB 側の停止は防げない）。
- **Neon の接続数**: アプリは必ずプール接続（`-pooler`）を使う。直接接続はコンピュートサイズに応じた接続数の上限があり、Cloud Run のインスタンスが増えると上限に達しやすい。
- **レート制限**はインスタンスのメモリで数えるため、インスタンスが複数になると上限は「インスタンス数 × 設定値」になる。
- シークレットの値をローテーションしたら、新しいリビジョンをデプロイする。`JWT_SECRET` を変えると全ユーザーのアクセストークンが、`JWT_REFRESH_SECRET` を変えるとリフレッシュトークンが無効になる（再ログインが必要）。
