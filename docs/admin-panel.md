# 管理画面（運営者向け）

アプリ運営者が使う管理画面です。Go の API サーバー内に `html/template` で実装しており、SNS アプリ（React）とは別の画面・別のアカウントでログインします。

- URL: `http://localhost:8080/admin`（本番は API サーバーのドメイン + `/admin`）
- 実装: `backend/internal/handler/admin_panel_handler.go`（ハンドラー）、`internal/service/admin_panel_service.go`（業務ロジック・監査ログ）、`internal/middleware/admin_session.go`（認証・CSRF・セキュリティヘッダー）、`internal/adminsession/`（セッション署名）、`internal/adminui/`（テンプレート・CSS。バイナリに埋め込み）
- DB: `backend/migrations/008_admin_panel.up.sql`

> 既存の「SNS アプリ内の管理者機能」（`users.role = 'admin'` のユーザーが使う `/api/v1/admin/*` と React の `/admin` 画面：投稿の強制削除・ユーザー停止）はそのまま残っています。本ドキュメントの管理画面はそれとは別物です。

---

## 1. 設計方針

### アカウントの種類

| 種類 | 定義場所 | できること |
|------|---------|-----------|
| ルート管理者 | 環境変数 `ADMIN_ROOT_USERNAME` / `ADMIN_ROOT_PASSWORD`（DB に保存しない） | すべて ＋ 管理者ユーザーの作成 |
| 管理者ユーザー | `admins` テーブル（ルート管理者が作成） | ダッシュボード・ユーザー審査・パスワードリセットリンク発行・管理者一覧の閲覧 |
| 一般ユーザー | `users` テーブル | 管理画面にはログインできない（アカウント体系が別。SNS の JWT でも入れない） |

- `admins.role` は現在 `admin` のみ。将来、閲覧専用（`viewer`）などの権限差分を追加できるよう列だけ用意しています（追加時は CHECK 制約と `AdminPanelService` の権限判定を拡張）。
- `admins.is_active = false` にした管理者は次のリクエストからログアウトされます（セッション確認時に毎回 DB を参照）。

### ユーザーの登録審査（`users.status`）

| status | 意味 | ログイン・API |
|--------|------|-------------|
| `pending` | 新規登録直後。管理者の承認待ち | ログイン 403 `ACCOUNT_PENDING`。登録時に Cookie を発行しない |
| `approved` | 承認済み | 通常どおり利用可 |
| `rejected` | 登録拒否 | ログイン 403 `ACCOUNT_REJECTED` |

- 既存ユーザーはマイグレーションで `approved` になります（`AutoMigrate` 経路でも `config.addUserStatusColumn` で同じ扱い）。
- 承認済みでないユーザーは `JWTAuth` で 401 `ACCOUNT_NOT_APPROVED`、`OptionalJWTAuth` で未ログイン扱い、リフレッシュも不可。ユーザー検索にも出ません。
- 状態遷移: 承認は `pending` / `rejected` → `approved`、拒否は `pending` → `rejected` のみ（承認済みユーザーの利用停止は既存の「停止」機能で行う）。条件付き UPDATE なので同時操作でも二重に処理されません。
- 承認・拒否してもメールは送りません。**運営が手動でメール連絡**してください。

### パスワードリセット

1. ユーザーがログイン画面の「パスワードをお忘れの方」（`/forgot-password`）からメールアドレスで申請（`POST /api/v1/auth/password-reset-requests`）。登録の有無にかかわらず同じ 202 を返す。対応中の申請がある間は重複して作らない。
2. 管理者が「パスワードリセット申請」画面で申請者を確認し、**リンクを発行**。リンクは発行直後の画面に 1 度だけ表示される。
3. 運営がリンクを手動でメール送付。
4. ユーザーがリンク（`<FRONTEND_BASE_URL>/reset-password?token=...`）から新しいパスワードを設定（`POST /api/v1/auth/password-reset`）。

- トークンは 256 ビットの乱数。DB には HMAC-SHA256 のハッシュだけを保存（`password_reset_tokens`）。
- **有効期限付き**（既定 24 時間）・**一回限り**（条件付き UPDATE で `used_at` を立てる）。再発行すると以前の未使用リンクは `revoked_at` で失効。
- 再設定するとそのユーザーの全リフレッシュトークンを失効させます（他端末もログアウト）。
- 承認済みでないユーザーにはリンクを発行できません。

### 認証・認可・セキュリティ

- セッション: `admin_session` Cookie（HS256 署名・有効期限 8 時間・`HttpOnly; SameSite=Strict; Path=/admin`、`Secure` は API の Cookie と同じ設定）。署名鍵は `ADMIN_SESSION_SECRET`（JWT の鍵とは別）。
- `/admin/login` と静的ファイル以外の全ルートに `AdminSession` ミドルウェアを適用（未ログインはログインページへリダイレクト）。
- CSRF: POST はフォームの `_csrf`（セッションに結びついた HMAC）を検証。加えて全体の `OriginCheck` と `SameSite=Strict`。
- レートリミット: 管理画面の全ルートに一般 API と同じ制限（IP あたり・ルートごとに 60 回/分）、ログイン POST には認証系の制限（5 回/分）。
- セキュリティヘッダー: `Content-Security-Policy`（自オリジンのみ・インラインスクリプトなし）、`X-Frame-Options: DENY`、`Cache-Control: no-store`、`Referrer-Policy: same-origin`（`no-referrer` だとフォーム送信の Origin が null になり CSRF 対策の Origin 検証で拒否されるため）。
- ルート管理者のパスワードは定数時間で比較。存在しない管理者名でも bcrypt 比較を行い応答時間で存在を推測させない。

### 監査ログ

管理操作（ログイン・ログアウト・承認・拒否・リセットリンク発行・管理者作成、およびその失敗）はすべて構造化ログに `event: "admin_audit"` で記録します。

| フィールド | 内容 |
|-----------|------|
| `performed_at` / `time` | いつ |
| `admin_kind` / `admin_id` / `admin_username` | 誰が（ルート管理者は `admin_id: "root"`） |
| `operation` | 何を（`admin.login` / `admin.logout` / `user.approve` / `user.reject` / `password_reset.issue_link` / `admin.create`） |
| `target_type` / `target_id` / `target_handle` / `target_username` | 誰に対して |
| `result` / `reason` | 結果（`success` / `failure` と理由） |
| `status_before` / `status_after` | 審査ステータスの変化（承認・拒否） |
| `request_id` | アクセスログと突き合わせるためのリクエストID |

メールアドレス・パスワード・リセットトークン/リンクはログに出しません。ログイン失敗は `attempted_username` と `reason` を WARN で記録します。

---

## 2. 環境変数

| 変数 | 必須 | 既定値 | 説明 |
|------|------|-------|------|
| `ADMIN_SESSION_SECRET` | 管理画面を使うなら必須 | なし | セッション Cookie の署名鍵。**空のときは管理画面が無効**（`/admin` は 404）。32 文字以上・`JWT_SECRET` / `JWT_REFRESH_SECRET` と別の値（違反すると起動しない）。`openssl rand -base64 48` で生成 |
| `ADMIN_ROOT_USERNAME` | ルート管理者を使うなら必須 | なし | ルート管理者のユーザー名 |
| `ADMIN_ROOT_PASSWORD` | ルート管理者を使うなら必須 | なし | ルート管理者のパスワード（12 文字以上・見本の値は不可）。ユーザー名とセットで設定 |
| `FRONTEND_BASE_URL` | 任意 | `CORS_ORIGINS` の先頭 | リセットリンクの URL に使うフロントエンドの URL（例: `https://<app>.web.app`） |
| `PASSWORD_RESET_TOKEN_TTL_MINUTES` | 任意 | `1440`（24 時間） | リセットリンクの有効期限（分）。10〜10080 |
| `AUTO_APPROVE_USERS` | E2E テストのみ | `false` | 新規登録ユーザーを自動で `approved` にする。下記「E2E テスト向けの自動承認」参照 |

本番では Cloud Run / Render の環境変数（シークレット）に設定してください。

---

## 3. ローカルでの使い方

1. `backend/.env` に追記する（`.env.example` に雛形あり）。

   ```bash
   ADMIN_SESSION_SECRET=<openssl rand -base64 48 の出力>
   ADMIN_ROOT_USERNAME=root
   ADMIN_ROOT_PASSWORD=<12文字以上のパスワード>
   FRONTEND_BASE_URL=http://localhost:5173
   ```

2. API を再起動する（環境変数の再読み込み）: `docker compose up -d --force-recreate api`
   - ログに `"message":"admin panel enabled"` が出れば有効。
3. ブラウザで `http://localhost:8080/admin` を開き、ルート管理者でログインする。
4. 「管理者ユーザー」画面から運営メンバー用の管理者ユーザーを作成する（ルート管理者のみ）。

### 動作確認の流れ

1. フロントエンド（`http://localhost:5173/signup`）で新規登録 → 「承認待ち」の案内が出る。ログインすると「承認待ち」エラー。
2. 管理画面「ユーザー」→「承認待ち」タブで対象を承認 → フロントエンドでログインできる。
3. フロントエンドの「パスワードをお忘れの方」から申請 → 管理画面「パスワードリセット申請」でリンクを発行 → 表示されたリンクを開いて再設定。

---

## 4. URL 一覧

| メソッド | URL | 内容 | 権限 |
|---------|-----|------|------|
| GET | `/admin/login` | ログインページ | 不要 |
| POST | `/admin/login` | ログイン（認証系レート制限） | 不要 |
| POST | `/admin/logout` | ログアウト | 管理者 |
| GET | `/admin` | ダッシュボード | 管理者 |
| GET | `/admin/users?status=pending\|approved\|rejected&cursor=` | ユーザー一覧（ステータス絞り込み・20 件ずつ） | 管理者 |
| GET | `/admin/users/:id` | ユーザー詳細（審査履歴・投稿数・リセット申請履歴） | 管理者 |
| POST | `/admin/users/:id/approve` | ユーザー承認 | 管理者 |
| POST | `/admin/users/:id/reject` | ユーザー拒否 | 管理者 |
| GET | `/admin/password-resets?status=pending\|issued\|completed&cursor=` | パスワードリセット申請一覧 | 管理者 |
| POST | `/admin/password-resets/:id/issue` | リセットリンク発行（再発行） | 管理者 |
| GET | `/admin/admins` | 管理者ユーザー一覧（ルートには作成フォームも表示） | 管理者 |
| POST | `/admin/admins` | 管理者ユーザー作成 | **ルート管理者のみ** |
| GET | `/admin/static/*` | CSS | 不要 |

一般ユーザー向けに追加した API（Swagger に記載）:

| メソッド | URL | 内容 |
|---------|-----|------|
| POST | `/api/v1/auth/password-reset-requests` | パスワードリセット申請（認証系レート制限） |
| POST | `/api/v1/auth/password-reset` | リセットリンクのトークンでパスワード再設定（認証系レート制限） |

フロントエンドに追加した画面: `/forgot-password`（申請）、`/reset-password?token=`（再設定）。

---

## 5. ダッシュボード

運営判断に使う指標を 1 画面にまとめています（集計はすべて GROUP BY / FILTER で数クエリにまとめ、件数に比例してクエリが増えない）。

- 承認待ちユーザー数・未対応のパスワードリセット申請数（0 件より多いと強調表示。クリックで該当一覧へ）
- ユーザー数（承認済み / 拒否 / 停止の内訳）、直近 7 日の新規登録数
- 直近 7 日のアクティブユーザー数（投稿したユーザー）、直近 7 日のいいね数
- 投稿数（今日の投稿数・コメント数）
- 有効な管理者ユーザー数
- 直近 14 日の投稿数・新規登録数の推移（日本時間で日別集計。SVG の棒グラフ）
- 承認待ちユーザーの最新 5 件（そのまま審査へ移動できる）

---

## 6. E2E テスト向けの自動承認（`AUTO_APPROVE_USERS`）

E2E テストは「新規登録 → そのまま投稿」を前提にしているため、テスト環境では新規登録ユーザーを承認待ちにせず `approved` で作成します。

- 有効になる条件: **`ENV=test` かつ `AUTO_APPROVE_USERS=true`** のときだけ（`config.Load` と `authService.initialStatus` の両方で確認）。
- 開発・本番（`ENV=development` / `production`）では、この変数を設定しても**無視され、必ず `pending`** になります。
- 設定箇所: `backend/.env.test`（`api_test` コンテナ）と `.github/workflows/ci.yml` の `env`。
- 有効時は起動ログに WARN（`AUTO_APPROVE_USERS is enabled ...`）が出ます。

**安全上の注意**

- 有効にすると誰でも審査なしで利用を開始できるため、**本番・ステージングなど実ユーザーがいる環境では絶対に `ENV=test` を使わない**こと（`ENV=test` はレート制限の緩和なども有効になる）。
- `.env.test` の値を本番の設定にコピーしないこと。
