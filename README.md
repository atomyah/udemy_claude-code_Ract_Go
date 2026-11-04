# udemy_claude-code_Ract_Go

## 画像付き投稿（Firebase Storage）

投稿に画像を最大 4 枚添付できます。アップロードと閲覧 URL の発行はすべてバックエンドが Firebase Admin SDK で行い、**フロントエンドは Firebase Storage に直接アクセスしません**（API が返す URL を `<img>` に渡すだけです）。

### 設計

| 項目 | 内容 |
|------|------|
| 対応形式 | JPEG / PNG / WebP（拡張子や `Content-Type` ではなく実際のバイト列で判定） |
| 最大サイズ・枚数 | 1 枚 5MB、1 投稿 4 枚（返信・引用リポストは 2 枚）。リクエスト全体もルート単位の `BodyLimit` で制限（投稿 51MB、返信・引用リポスト 11MB。超過は 413） |
| 保存先パス | `posts/{post_id}/{media_uuid}{.jpg\|.png\|.webp}`。拡張子も実バイトから決め、利用者のファイル名は使わない |
| 公開範囲 | 非公開オブジェクト（ACL を付けない）。Security Rules は全拒否のままでよい |
| DB | `media.storage_path` に**オブジェクトパスだけ**を保存する。署名付き URL は期限切れになるので保存しない。`media.url` は公開 URL で保存していた既存行のためだけに残している |
| 表示 URL | API レスポンスを組み立てるたびに V4 署名付き URL（GET 専用）を発行する。有効期限は既定 60 分（`MEDIA_SIGNED_URL_TTL_MINUTES`）。サービスアカウントの秘密鍵によるローカル署名のため、画像ごとの通信は発生しない |
| 失敗時 | 全ファイルを検証してからアップロードし、投稿と `media` は同じトランザクションで保存する。途中で失敗したらアップロード済みのオブジェクトを削除し、投稿は作らない |

動画（MP4 / MOV、1 本、50MB）も同じ経路で非公開保存・署名付き URL 配信になります。

**今回の対象外:** 投稿編集時の画像の差し替え・追加、投稿削除時の Storage オブジェクトの削除（投稿は論理削除のみで、画像は Storage に残ります）、アバター・バナーの非公開化（従来どおり公開 URL です）。

**既知の制限:** 署名付き URL の有効期限を過ぎるまで同じ画面を開いたままにすると画像が読み込めなくなります。画面の再読み込み・移動で新しい URL を取得します。

### 環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| `FIREBASE_CREDENTIALS_JSON` | ○ | サービスアカウントキーの JSON **文字列**（ファイルパスではなく中身） |
| `FIREBASE_STORAGE_BUCKET` | ○ | Storage バケット名（例: `<project-id>.firebasestorage.app`。`gs://` は付けない） |
| `MEDIA_SIGNED_URL_TTL_MINUTES` | | 署名付き URL の有効期限（分）。既定 60、1〜10080（7 日） |

開発・本番では、どちらかの必須変数が不正（未設定・JSON でない・秘密鍵で署名できない）だと API は起動しません。`ENV=test` だけは未設定でも起動し、画像のアップロードは失敗します（単体テスト・E2E は Firebase に接続しません）。

### ローカルで実際の Firebase Storage に接続して確認する

**秘密情報をリポジトリに置かないでください。** キーファイルはリポジトリの外に保存し、値は `.gitignore` 済みの `backend/.env` にだけ書きます。チャット・Issue・ログにも貼らないでください。

1. Firebase コンソール > プロジェクトの設定 > サービスアカウント >「新しい秘密鍵の生成」でキーファイルをダウンロードし、**リポジトリの外**に置きます。
2. キーを 1 行の JSON 文字列にします（bash / PowerShell 共通。パスは区切りに `/` を使います）。
   ```bash
   node -p "JSON.stringify(require('C:/secrets/my-project-firebase-adminsdk.json'))"
   ```
3. Firebase コンソール > Storage でバケット名を確認し（`gs://` の後ろの部分）、`backend/.env` に書きます。JSON は**シングルクォートで囲みます**（`private_key` の `\n` を変換させないため）。
   ```dotenv
   FIREBASE_CREDENTIALS_JSON='{"type":"service_account","project_id":"...","private_key":"-----BEGIN PRIVATE KEY-----\n...","client_email":"...", ...}'
   FIREBASE_STORAGE_BUCKET=<project-id>.firebasestorage.app
   # 任意（期限切れの確認をしたいときは 1 にする）
   # MEDIA_SIGNED_URL_TTL_MINUTES=60
   ```
4. API とフロントエンドを起動します。`.env` の変更は再起動では反映されないので、コンテナを作り直します。
   ```bash
   docker compose up -d --force-recreate api   # media.storage_path 列は開発環境では起動時に自動で追加される
   docker compose logs -f api                  # "server started" を確認。"firebase storage init failed" なら設定を見直す
   cd frontend && npm run dev                  # http://localhost:5173
   ```
   Docker を使わずに API を動かす場合は、同じ値をシェルの環境変数に設定してから起動します。

   ```powershell
   $env:FIREBASE_CREDENTIALS_JSON = (node -p "JSON.stringify(require('C:/secrets/my-project-firebase-adminsdk.json'))")
   $env:FIREBASE_STORAGE_BUCKET = "<project-id>.firebasestorage.app"
   ```
5. 画像投稿を確認します。
   1. アカウントを登録し、管理画面（`http://localhost:8080/admin`）で承認してからログインします。
   2. 投稿欄の画像アイコンから 1〜4 枚選び、本文を入力して投稿します。タイムラインと投稿詳細に画像が表示されれば成功です。
   3. 表示された画像の URL に `X-Goog-Signature` が含まれていることを、ブラウザの開発者ツールで確認します。
   4. 同じ URL から `?` 以降を取り除いて開くと `403 / AccessDenied` になること（非公開であること）を確認します。
   5. Firebase コンソール > Storage に `posts/{post_id}/` が作られていること、`docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT storage_path, url FROM media ORDER BY created_at DESC LIMIT 4"'` で `storage_path` だけが入っていることを確認します。
   6. 異常系として、5 枚目の追加・GIF・5MB 超の画像がエラーになること、画像なしの投稿が従来どおりできることを確認します。

### Firebase Storage のセキュリティルール

Admin SDK はルールの影響を受けず、署名付き URL もルールを経由しません。クライアントからのアクセスは使わないので、すべて拒否します（Firebase コンソール > Storage > ルール）。

```
rules_version = '2';
service firebase.storage {
  match /b/{bucket}/o {
    match /{allPaths=**} {
      allow read, write: if false;
    }
  }
}
```

### 本番

Cloud Run で Secret Manager のシークレットを環境変数 `FIREBASE_CREDENTIALS_JSON` に注入します（アプリ側はローカルと同じく環境変数を読むだけです）。バケット名は通常の環境変数です。デプロイ前に `backend/migrations/009_add_media_storage_path.up.sql` を適用してください。手順は [docs/deploy.md](docs/deploy.md) を参照してください。

## 管理画面（運営者向け）

API サーバーの `/admin`（ローカルは `http://localhost:8080/admin`）で、ユーザーの登録審査・パスワードリセットリンクの発行・管理者ユーザーの管理・ダッシュボードを利用できます。
新規登録したユーザーは管理者が承認するまで利用できません（`pending`）。

利用するには `backend/.env` に次の環境変数を設定して API を再起動します（`docker compose up -d --force-recreate api`）。

| 変数 | 説明 |
|------|------|
| `ADMIN_SESSION_SECRET` | 管理画面セッションの署名鍵（32 文字以上・JWT の鍵と別）。空なら管理画面は無効 |
| `ADMIN_ROOT_USERNAME` / `ADMIN_ROOT_PASSWORD` | ルート管理者のログイン情報（DB に保存しない。パスワード 12 文字以上） |
| `FRONTEND_BASE_URL` | パスワードリセットリンクに使うフロントエンドの URL（既定: `CORS_ORIGINS` の先頭） |
| `PASSWORD_RESET_TOKEN_TTL_MINUTES` | リセットリンクの有効期限（分、既定 1440） |
| `AUTO_APPROVE_USERS` | **E2E テスト専用**。`ENV=test` のときだけ新規登録を自動承認する（それ以外の環境では無視） |

URL 一覧・運用手順・監査ログ・セキュリティ上の注意は [docs/admin-panel.md](docs/admin-panel.md) を参照してください。
