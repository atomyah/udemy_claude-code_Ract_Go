# セキュリティレビュー — PHASE01（開発環境・プロジェクト基盤構築）由来のリスク

- **調査日**: 2026-08-23
- **対象リビジョン**: `512c22e`（+ 未コミットの作業ツリー変更）
- **調査範囲**: `todo/PHASE01_project_setup.md` のチェック項目に対応する成果物のみ
  - `backend/internal/config/config.go`
  - `backend/cmd/server/main.go`（Echo 初期化・ミドルウェア・`/health`・Swagger ルート）
  - `backend/.env` / `.env.example` / `frontend/.env.local` / `.gitignore`
  - `docker-compose.yml` / `backend/Dockerfile` / `backend/Dockerfile.dev` / `backend/.air.toml`
  - `.mcp.json`（開発基盤セットアップの一部として同時期に作成）
- **調査手法**: 静的コードレビュー ＋ 稼働中のローカルコンテナに対する非破壊的な動的検証（HTTP レスポンス確認・ポート公開状況確認）
- **注記**: 本レポートに秘密情報の実値は記載していない。すべてマスク・特徴のみを記述している。

> PHASE02 以降で作り込まれた認証ロジック・投稿処理などの脆弱性は本レポートの対象外。
> 末尾の「付録B」に、調査中に視認できた PHASE01 スコープ外の論点のみ列挙している。

---

## 1. サマリー

| ID | 深刻度 | 概要 | 該当 |
|----|--------|------|------|
| P01-01 | **Critical** | `.gitignore` から `.env` 除外行が削除され、Firebase 秘密鍵入りの `backend/.env` が次回コミットで流出する状態 | `.gitignore` |
| P01-02 | **Critical** | `.mcp.json` が Git 追跡対象のまま Context7 API キーを保持（`.gitignore` 追記は無効） | `.mcp.json` |
| P01-03 | **High** | Adminer（DB 管理 UI）を `0.0.0.0:8081` で常時公開 | `docker-compose.yml:35-41` |
| P01-04 | **High** | PostgreSQL を `0.0.0.0:5432` で公開＋既定の弱いパスワード＋`sslmode=disable` | `docker-compose.yml:8,10` |
| P01-05 | **High** | 認証エンドポイントにレート制限なし（実測 15/15 通過） | `main.go:104-111` |
| P01-06 | **High** | JWT シークレットが低エントロピーの固定文字列、長さ検証なし | `config.go:33-38`, `backend/.env` |
| P01-07 | **Medium** | `ENV` 既定値が `development` のため、本番で未設定なら Swagger UI が公開される | `config.go:30`, `main.go:113-115` |
| P01-08 | **Medium** | `DATABASE_URL` の既定値に資格情報がハードコード＋TLS 無効 | `config.go:23` |
| P01-09 | **Medium** | リクエストボディサイズ制限なし（5MB がそのまま読み込まれる） | `main.go:104-111` |
| P01-10 | **Medium** | セキュリティヘッダー（`X-Content-Type-Options` 等）が一切付与されない | `main.go:104-111` |
| P01-11 | **Medium** | 本番イメージが root 実行＋`alpine:latest`（プロジェクト規約の禁止事項） | `backend/Dockerfile:12,14,19` |
| P01-12 | **Medium** | 本番 `golang:1.22`（EOL）と `go.mod` の `go 1.24` が不一致、`swag@latest` 未固定 | `Dockerfile:1`, `Dockerfile.dev:7` |
| P01-13 | **Low** | CORS が単一文字列固定で複数オリジン非対応、起動時の妥当性検証なし | `main.go:106-111` |
| P01-14 | **Low** | アクセスログにクエリ文字列（検索語など）がそのまま記録される | `main.go:104` |
| P01-15 | **Low** | `.playwright-mcp/` が Git 無視対象外でブラウザログが蓄積 | `.gitignore` |
| P01-16 | **Low** | Firebase サービスアカウント JSON がリポジトリ作業ツリー直下に配置 | リポジトリ直下 |
| P01-17 | **Info** | `api` サービスに healthcheck 未設定、`tmp/` のボリューム分離なし | `docker-compose.yml:18-33` |

**Critical 2 件・High 4 件**。とくに P01-01 と P01-02 は「すでに危険」ではなく「次の `git add -A && git commit` で不可逆的に危険になる」種類のため、最優先で対処すべき。

---

## 2. 詳細

### P01-01 【Critical】`.gitignore` の `.env` 除外行が削除され、Firebase 秘密鍵が流出寸前

**PHASE01 該当項目**: 「`backend/.env` を作成（`.gitignore` に追加）」

**状況**

コミット済み `.gitignore`（`512c22e`）には除外行が正しく存在していたが、**未コミットの作業ツリーで削除されている**。

```diff
 # 環境変数
-.env
-.env.local
-.env.*.local
-backend/.env
```

その結果、現在の無視判定は次のとおり:

```
backend/.env                                        NOT ignored   ← 危険
.mcp.json                                           NOT ignored   ← 危険
frontend/.env.local                                 IGNORED       （frontend/.gitignore の *.local が偶然カバー）
udemy-claudecode-react-go-firebase-adminsdk.json    IGNORED
```

`backend/.env` の中身（実値はマスク）:

| キー | 長さ | 内容の性質 |
|------|------|-----------|
| `FIREBASE_CREDENTIALS_JSON` | 2,369 | **サービスアカウント JSON 全文（`private_key` を含むことを確認済み）** |
| `JWT_SECRET` | 39 | `dev-` で始まる英数字ハイフンのみの固定文字列 |
| `JWT_REFRESH_SECRET` | 39 | 同上 |
| `POSTGRES_PASSWORD` | 12 | 既定値 |
| `DATABASE_URL` | 63 | 資格情報を含む接続文字列 |

**影響**

次回コミットで Firebase サービスアカウントの秘密鍵が Git 履歴に入る。リモートへ push した時点で、鍵の再発行だけでは足りず履歴の書き換え（`git filter-repo` 等）が必要になる。この鍵は Firebase Storage および Auth に対する管理者権限を持つため、漏洩すれば全ユーザーのメディア閲覧・削除、任意ユーザーとしての ID トークン発行が可能になる。

**確認済み**: Git 履歴（`--all`）に `.env` / `*adminsdk*` は現時点で**含まれていない**。まだ間に合う。

**修正**

```gitignore
# 環境変数
.env
.env.local
.env.*.local
backend/.env
frontend/.env.local
```

加えて、`FIREBASE_CREDENTIALS_JSON` に JSON 全文を直接埋める運用自体を見直し、ローカルではファイルパス参照（`GOOGLE_APPLICATION_CREDENTIALS`）、本番ではシークレットマネージャー経由にすることを推奨する。

---

### P01-02 【Critical】`.mcp.json` が追跡対象のまま API キーを保持

**状況**

`.gitignore` に `# Context7 API Key` として `.mcp.json` が追記されているが、**`.gitignore` はすでに Git が追跡しているファイルには効かない**。

```
$ git ls-files --error-unmatch .mcp.json
.mcp.json          ← 追跡されている
```

コミット済みバージョン（`512c22e`）では `"env": {}` で空だが、作業ツリーでは `CONTEXT7_API_KEY` に UUID 形式のキーが設定され、ファイルは `M`（変更済み）状態にある。

**影響**

次回 `git add .mcp.json` または `git add -A` で API キーがコミットされる。

**修正**

```bash
git rm --cached .mcp.json          # 追跡を外す（ファイルは残る）
# .gitignore の .mcp.json はそのまま有効になる
```

キーは環境変数参照（`"CONTEXT7_API_KEY": "${CONTEXT7_API_KEY}"`）またはユーザースコープ設定（`~/.claude/`）へ移すのが望ましい。あわせて `.mcp.json.example` をコミットしてテンプレートを共有する。

---

### P01-03 【High】Adminer を LAN に常時公開している

**PHASE01 該当項目**: 「`docker-compose.yml` を作成（PostgreSQL + バックエンド）」
※ Adminer は `.claude/rules/docker.md` の構成に存在しない、後から追加されたサービス。

**状況**

```yaml
# docker-compose.yml:35-41
adminer:
  image: adminer:4-standalone
  restart: unless-stopped
  ports:
    - "8081:8080"
```

実機確認:

```
http://localhost:8081 -> 200
ADMINER_VERSION=4.17.1 / PHP 8.4.23
$ docker compose port adminer 8080
0.0.0.0:8081                    ← 全ネットワークインターフェースに公開
```

**影響**

Adminer 自体は最新版（4.17.1）で既知の重大な脆弱性はないが、任意のホスト・任意の DB へ接続できる汎用 DB クライアントが LAN 上の誰からでも到達可能になっている。P01-04 の既定資格情報（`sns_user` / `sns_password`）と組み合わせると、同一 LAN（カフェ・コワーキング・社内ネットワーク）の第三者がブラウザだけで DB 全体を読み書きできる。`restart: unless-stopped` のため、開発していない間もバックグラウンドで公開され続ける。実際、本調査時点でコンテナは **6 週間前から起動しっぱなし**だった。

**修正**

```yaml
adminer:
  image: adminer:4.17.1-standalone   # タグを固定
  profiles: ["tools"]                # 既定では起動しない
  ports:
    - "127.0.0.1:8081:8080"          # ループバックのみ
```

`docker compose --profile tools up -d adminer` で必要なときだけ起動する運用にする。

---

### P01-04 【High】PostgreSQL の全インターフェース公開＋弱い既定資格情報＋TLS 無効

**状況**

```yaml
# docker-compose.yml:8,10
POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-sns_password}
ports:
  - "5432:5432"
```

```
$ docker compose port db 5432
0.0.0.0:5432
```

`.env.example` にも `sns_user` / `sns_password` / `sslmode=disable` が平文で記載されており、そのまま使われている（`backend/.env` の `POSTGRES_PASSWORD` は 12 文字＝既定値と一致）。

**影響**

LAN 上の任意のホストから `psql -h <開発機IP> -U sns_user` で接続試行が可能。パスワードはテンプレートに書かれた推測容易な値。`sslmode=disable` により通信は平文で、同一セグメントでの盗聴に対して無防備。

**修正**

```yaml
ports:
  - "127.0.0.1:5432:5432"
```

DB ポートはそもそも公開せず、必要時のみ `docker compose exec db psql` を使う運用が最も安全。`.env.example` のパスワードは `CHANGE_ME` にし、実値は各開発者が生成する。

---

### P01-05 【High】認証エンドポイントにレート制限がない

**PHASE01 該当項目**: 「`cmd/server/main.go` に Echo の最小起動コードを書く」（ミドルウェアスタックの構成）

**状況**

`main.go:104-111` のミドルウェアは `Logger` / `Recover` / `CORS` の 3 つのみ。`middleware.RateLimiter` は全コードベースに 1 箇所も存在しない。

`.claude/rules/security.md` は「`POST /auth/login`: IP あたり 10 回/分、`POST /auth/register`: IP あたり 5 回/分」を明記しているが未実装。

**実測**

```
$ for i in $(seq 1 15); do curl -X POST .../api/v1/auth/login -d '{...}'; done
401 401 401 401 401 401 401 401 401 401 401 401 401 401 401
```

15 回連続で 429 が一度も返らない。制限は存在しない。

**影響**

パスワードのブルートフォース、および登録エンドポイントを使ったアカウント大量生成が可能。加えて、実在アカウントに対するログイン試行では bcrypt（コスト 12）が毎回走るため、少数の並列リクエストで CPU を飽和させる DoS にもなる。

**修正**

```go
authGroup := v1.Group("/auth")
authGroup.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
    Store: middleware.NewRateLimiterMemoryStoreWithConfig(
        middleware.RateLimiterMemoryStoreConfig{
            Rate: rate.Every(6 * time.Second), Burst: 10, ExpiresIn: 3 * time.Minute,
        }),
    IdentifierExtractor: func(c echo.Context) (string, error) { return c.RealIP(), nil },
}))
```

---

### P01-06 【High】JWT シークレットが低エントロピーで、長さの検証もない

**PHASE01 該当項目**: 「`internal/config/config.go` で環境変数を読み込む構造体を定義」

**状況**

```go
// config.go:33-38 — 空文字チェックのみ
if cfg.JWTSecret == "" {
    log.Fatal("JWT_SECRET is required")
}
```

実際の値は `JWT_SECRET` / `JWT_REFRESH_SECRET` ともに **`dev-` で始まる 39 文字の英数字ハイフンのみの人間可読な文字列**。`.env.example` のプレースホルダ（`change-me-to-a-random-32-chars-string`）と同じ発想の固定値で、`.claude/rules/security.md` が要求する「32 バイト以上のランダム文字列」を満たしていない。

**影響**

HS256 の署名鍵が推測・辞書攻撃で復元されると、任意の `sub` と `role: "admin"` を持つトークンを偽造できる。`AdminOnly` ミドルウェア（`middleware/admin.go`）はトークン内の `role` クレームのみを見て DB を参照しないため、そのまま管理者 API（投稿削除・ユーザー停止）が通る。`.env.example` をそのまま本番へ持ち込むと、公開リポジトリの既定値で認証が破られる。

**修正**

```go
const minSecretLen = 32
if len(cfg.JWTSecret) < minSecretLen || len(cfg.JWTRefreshSecret) < minSecretLen {
    log.Fatal("JWT_SECRET / JWT_REFRESH_SECRET は 32 文字以上のランダム文字列が必要です")
}
if cfg.JWTSecret == cfg.JWTRefreshSecret {
    log.Fatal("JWT_SECRET と JWT_REFRESH_SECRET は異なる値にしてください")
}
```

値の生成は `openssl rand -base64 48`。`.env.example` は空欄＋生成コマンドのコメントにする。

---

### P01-07 【Medium】`ENV` の既定値が `development` のため、設定漏れで Swagger UI が公開される

**PHASE01 該当項目**: 「Echo に Swagger UI ルートを追加（`/swagger/*`、`ENV=development` のみ）」

**状況**

```go
// config.go:30
Env: getEnv("ENV", "development"),   // 未設定なら development に倒れる

// main.go:113-115
if cfg.IsDevelopment() {
    e.GET("/swagger/*", echoSwagger.WrapHandler)
}
```

**実測（ローカル）**

```
swagger/index.html -> 200
swagger/doc.json   -> 200
```

**影響**

ゲート自体は正しく実装されているが、**フェイルオープン**な既定値になっている。Render / Cloud Run で `ENV` の設定を忘れた場合、本番で全 API 定義（管理者エンドポイント含む）と実行可能な Swagger UI が無認証公開される。`.claude/rules/openapi.md` の「本番環境で Swagger UI を公開する」禁止事項に抵触するリスク。

**修正**

既定値を安全側に倒す。

```go
Env: getEnv("ENV", "production"),
```

もしくは `ENV` 未設定時に `log.Fatal` で起動を止める（明示必須化）。

---

### P01-08 【Medium】`DATABASE_URL` の既定値に資格情報がハードコードされ、TLS が無効

```go
// config.go:23
DatabaseURL: getEnv("DATABASE_URL",
    "postgres://sns_user:sns_password@db:5432/sns_db?sslmode=disable"),
```

**影響**

ソースコードに DB 資格情報が残る（`.claude/rules/security.md`「機密情報は全て環境変数で管理する」に抵触）。また `DATABASE_URL` の設定漏れに気づかず起動してしまい、意図しない DB へ平文接続する事故につながる。

**修正**: 既定値を空にし、未設定なら `log.Fatal`。ローカルの接続先は `backend/.env` のみで与える。

---

### P01-09 【Medium】リクエストボディのサイズ制限がない

**状況**: `middleware.BodyLimit` はコードベースに存在しない。

**実測**

```
5MB の JSON POST -> 400（413 ではない）
```

413 ではなく 400 が返る＝**サーバーは 5MB を全部メモリに読み込んでからパースに失敗している**。上限は事実上存在しない。

**影響**

巨大ボディの並列送信でメモリを枯渇させる DoS。`.claude/rules/security.md` は「Echo のリクエストサイズ制限ミドルウェアで上限を設ける」と規定。PHASE05 のメディアアップロード（画像 5MB × 4／動画 100MB）で必要になる上限も、この基盤層が受け皿になる。

**修正**

```go
e.Use(middleware.BodyLimit("2M"))                     // 既定は小さく
v1.POST("/posts", postHandler.CreatePost, jwtMiddleware, middleware.BodyLimit("110M")) // アップロード系のみ緩和
```

---

### P01-10 【Medium】セキュリティヘッダーが一切付与されていない

**実測（`GET /health` のレスポンスヘッダー全量）**

```
HTTP/1.1 200 OK
Content-Type: application/json
Vary: Origin
Date: ...
Content-Length: 16
```

`X-Content-Type-Options` / `X-Frame-Options` / `Strict-Transport-Security` / `Referrer-Policy` / `Content-Security-Policy` がいずれも存在しない。

**影響**: MIME スニッフィング、クリックジャッキング、HTTPS ダウングレード、リファラ経由の情報漏洩に対する多層防御が効かない。

**修正**

```go
e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
    XSSProtection:         "0",
    ContentTypeNosniff:    "nosniff",
    XFrameOptions:         "DENY",
    HSTSMaxAge:            31536000,
    ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
}))
```

（`HSTSMaxAge` は本番のみ有効化する。）

---

### P01-11 【Medium】本番イメージが root 実行、ベースタグが `latest`

```dockerfile
# backend/Dockerfile
FROM alpine:latest      # :12  ← タグ非固定
WORKDIR /root/          # :14  ← root のホームで実行
COPY --from=builder /app/server .
CMD ["./server"]        # :19  ← USER 指定なし = root
```

`.claude/rules/docker.md` の禁止事項「ベースイメージに `latest` タグを使う」に真っ向から違反している。

**影響**

アプリに RCE 級の脆弱性が生じた場合、コンテナ内 root 権限で実行されるためエスケープの前提条件が揃いやすくなる。`latest` タグはビルド時期によって中身が変わり、再現性がなく、脆弱性スキャン結果も追跡できない。

**修正**

```dockerfile
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /app/server .
USER 10001
EXPOSE 8080
CMD ["./server"]
```

---

### P01-12 【Medium】Go のバージョン不一致（本番は EOL）、ツールのバージョン非固定

| 箇所 | Go |
|------|-----|
| `backend/go.mod` | `go 1.24` |
| `backend/Dockerfile:1`（本番） | `golang:1.22-alpine` |
| `backend/Dockerfile.dev:1`（開発） | `golang:1.24-alpine` |

**影響**

本番ビルドだけ 1.22 系。Go 1.22 はサポート対象外（セキュリティパッチ提供終了）で、`go.mod` の `go 1.24` 指定によりビルド時にツールチェーンを自動ダウンロードするか、失敗する。いずれにせよ「開発で通ったコードが本番イメージで検証されていない」状態であり、標準ライブラリの既知脆弱性（`net/http`・`crypto/tls` 系）が本番イメージにのみ残るリスクがある。

また `Dockerfile.dev:7` の `swag@latest` はバージョン非固定で、上流の変更がそのままビルド環境に入るサプライチェーンリスクになる（`air` は `@v1.61.7` で固定できているので、同じ扱いにすべき）。

**修正**: 本番も `golang:1.24-alpine` に統一し、`swag` も `@v1.16.6`（`go.mod` と一致）に固定する。あわせて CI に `govulncheck ./...` を追加する。

---

### P01-13 【Low】CORS が単一文字列で、複数オリジン指定が機能しない

```go
// main.go:107
AllowOrigins: []string{cfg.CORSOrigins},   // カンマ分割していない
```

`.claude/rules/infra.md` は `strings.Split(os.Getenv("CORS_ORIGINS"), ",")` を指定しているが未実装。`CORS_ORIGINS="https://a.example,https://b.example"` と設定すると、その連結文字列そのものが 1 つのオリジンとして扱われ、両方とも拒否される。

**動的検証の結果、危険側の挙動はなかった**:

```
$ curl -X OPTIONS .../api/v1/posts -H "Origin: https://evil.example.com"
HTTP/1.1 204 No Content       ← Access-Control-Allow-Origin ヘッダーなし＝正しく拒否
```

Echo v4.13.3 の CORS は `AllowCredentials: true` と `*` の併用を `UnsafeWildcardOriginWithAllowCredentials`（既定 false）で明示的にブロックする実装になっており（`middleware/cors.go:231`）、仮に `CORS_ORIGINS=*` を設定してもクレデンシャル付きリクエストは反射されない。したがって深刻度は Low。

**影響**: 本番でステージング・カスタムドメインなど複数オリジンが必要になった際、「設定したのに通らない」→ `*` へ緩めるという誤った対処を誘発しやすい。

**修正**

```go
origins := strings.Split(cfg.CORSOrigins, ",")
for i := range origins { origins[i] = strings.TrimSpace(origins[i]) }
if !cfg.IsDevelopment() {
    for _, o := range origins {
        if o == "*" { log.Fatal("本番環境で CORS_ORIGINS に * は使用できません") }
    }
}
AllowOrigins: origins,
```

---

### P01-14 【Low】アクセスログにクエリ文字列がそのまま記録される

`main.go:104` の `middleware.Logger()` は既定フォーマットで `${uri}` を出力する。`/api/v1/search/users?q=...` や `/api/v1/search/posts?q=...` の検索語がすべて標準出力へ記録され、Cloud Run / Render のログに保持される。

`.claude/rules/security.md`「ログに機密情報（パスワード・トークン・メールアドレス）を出力しない」の趣旨からも、検索クエリは利用者の関心事そのものを含むため、保持期間・アクセス権の管理が必要になる。

**修正**: `LoggerWithConfig` でフォーマットを `${method} ${path}`（クエリを含まない `path`）に変更するか、`slog` ベースの構造化ログへ差し替える。

---

### P01-15 【Low】`.playwright-mcp/` が Git 無視対象外

ブラウザのコンソールログ（`console-*.log`）が 7 ファイル蓄積し、無視対象になっていない。現時点のログを走査した限り `Authorization` / `access_token` / `password` の文字列は**含まれていなかった**が、フロントエンドのエラーログは今後トークンやリクエストボディを吐きうる。

**修正**: `.gitignore` に `.playwright-mcp/` を追加する。

---

### P01-16 【Low】サービスアカウント JSON がリポジトリ作業ツリー直下に置かれている

`udemy-claudecode-react-go-firebase-adminsdk.json`（`private_key` を含む）がリポジトリ直下に存在する。現在は `.gitignore` の `*firebase-adminsdk*.json` でカバーされているが、ファイル名を変えた瞬間に保護が外れる**パターン依存の防御**であり、`backend/.env` の `FIREBASE_CREDENTIALS_JSON` と鍵が二重に存在している。

**修正**: リポジトリ外（例: `~/.secrets/`）へ移動し、`GOOGLE_APPLICATION_CREDENTIALS` でパス指定する。

---

### P01-17 【Info】compose の細部

- `api` サービスに `healthcheck` がない。PHASE01 で `/health` を実装済みなので、これを compose と本番プラットフォーム双方のヘルスチェックに接続すべき。
- `.claude/rules/docker.md` の構成にある `- /app/tmp`（air のビルド成果物をホストから分離する匿名ボリューム）が `docker-compose.yml` に存在せず、`backend/tmp/main`（ビルド済みバイナリ）がホスト側に生成されている。`.gitignore` されているため実害は小さいが、規約どおり分離するのが望ましい。

---

## 3. PHASE01 チェックリストとの突き合わせ

| PHASE01 の項目 | 記載 | 実際 |
|---------------|------|------|
| `backend/.env` を作成（`.gitignore` に追加） | `[x]` | **未達成** — 除外行が作業ツリーで削除済み（P01-01） |
| `internal/config/config.go` で環境変数を読み込む | `[x]` | 達成。ただし検証が空文字チェックのみ（P01-06/07/08） |
| Swagger UI を `ENV=development` のみ | `[x]` | 達成。ただし既定値がフェイルオープン（P01-07） |
| `backend/Dockerfile`（本番用マルチステージ） | `[x]` | 達成。ただし root 実行・`latest` タグ（P01-11/12） |
| `docker-compose.yml` を作成 | `[x]` | 達成。ただし規約外の Adminer 追加・全 IF 公開（P01-03/04） |
| `.env.example` をコミット用テンプレートとして残す | `[x]` | 達成。ただし推測可能な既定パスワード／シークレットを同梱（P01-04/06） |
| Echo の最小起動コード | `[x]` | 達成。セキュリティ系ミドルウェアが未導入（P01-05/09/10） |

チェックボックスは付いているが、**セキュリティ観点の完了基準が定義されていなかった**ため、上記が素通りしている。PHASE01 の完了基準に「シークレットが `.gitignore` されていることを `git check-ignore` で確認」「公開ポートがループバック限定であることを確認」を追加することを推奨する。

---

## 4. 推奨対応順序

### 即時（コミット前に必ず）

1. **P01-01** — `.gitignore` の `.env` 除外行を復元
2. **P01-02** — `git rm --cached .mcp.json`
3. 上記 2 件の後に `git status` で `backend/.env` / `.mcp.json` が消えていることを確認

```bash
# 検証コマンド
git check-ignore -v backend/.env .mcp.json frontend/.env.local
git status --porcelain | grep -E "\.env|\.mcp\.json"   # 何も出なければ OK
```

### 今週中

4. **P01-03 / P01-04** — ポートを `127.0.0.1` バインドに変更、Adminer を profile 化
5. **P01-06** — JWT シークレットを `openssl rand -base64 48` で再生成し、`config.go` に長さ検証を追加
6. **P01-05** — 認証エンドポイントにレート制限を追加

### 本番デプロイ（PHASE13）までに

7. **P01-07 / P01-08** — 環境変数の既定値を安全側へ、未設定時は起動失敗させる
8. **P01-09 / P01-10** — `BodyLimit` と `Secure` ミドルウェアを追加
9. **P01-11 / P01-12** — Dockerfile を非 root 化・タグ固定・Go バージョン統一、CI に `govulncheck` 追加
10. **P01-13 〜 P01-17** — CORS のカンマ分割、ログ整形、`.gitignore` 追補、鍵ファイルの外出し

---

## 付録A. 動的検証の実行記録

すべて開発者自身のローカル環境（`localhost`）に対する非破壊的な確認。

| 検証 | 結果 |
|------|------|
| セキュリティヘッダー（`curl -D - /health`） | `Content-Type` / `Vary` / `Date` / `Content-Length` のみ（P01-10） |
| Swagger 公開（`/swagger/index.html`, `/swagger/doc.json`） | いずれも `200`（P01-07） |
| CORS 拒否（`Origin: https://evil.example.com` でプリフライト） | `204`、`Access-Control-Allow-Origin` なし＝正しく拒否（P01-13） |
| レート制限（`/api/v1/auth/login` へ 15 回連続 POST） | `401` × 15、`429` なし（P01-05） |
| ボディ上限（5MB の JSON を POST） | `400`（`413` ではない＝全量読み込み後にパース失敗）（P01-09） |
| ポート公開範囲（`docker compose port db 5432` / `adminer 8080`） | `0.0.0.0:5432` / `0.0.0.0:8081`（P01-03/04） |
| Adminer バージョン（コンテナ内 `printenv`） | `ADMINER_VERSION=4.17.1`（既知の重大脆弱性なし） |
| Echo CORS 実装（コンテナ内 `middleware/cors.go:231`） | ワイルドカード＋クレデンシャルは既定でブロック（P01-13 を Low に引き下げた根拠） |
| Git 履歴の秘密情報走査（`git log --all --name-only`） | `.env` / `*adminsdk*` の該当なし＝未コミット |

---

## 付録B. PHASE01 スコープ外で視認した論点（別レビュー推奨）

本レポートの対象外だが、調査中に確認できた範囲で記録する。深掘りは各 PHASE 単位のレビューで行うべき。

- **PHASE02（`internal/config/db.go`）**: GORM のロガーが全環境で `logger.Info`。`INSERT INTO users` の実 SQL がログに出るため、`password_hash` とメールアドレスが標準出力に記録される。本番では `logger.Silent` / `logger.Error` にすべき。
- **PHASE03（`internal/middleware/auth.go`）**: 保護ルートで `is_suspended` を検証していない。ユーザーを停止しても、発行済みアクセストークンが切れるまで（最大 15 分）全 API を利用できる。
- **PHASE03/07（`internal/middleware/admin.go`）**: `role` を JWT クレームのみで判定し DB を参照しない。管理者権限を剥奪してもトークン有効期間中は admin として振る舞える。
- **PHASE03（`internal/handler/auth_handler.go:48,95,197`）**: `err.Error()` をそのままレスポンスの `message` に載せている箇所がある。内部エラー文言の外部露出につながらないか確認が必要。
- **横断**: `internal/**` にテストが 1 件も存在しない（`.claude/rules/testing.md` の優先項目「JWTMiddleware: 有効/無効/期限切れ」等が未実装）。
- **`backend/cmd/listbuckets_main.go`**: `cmd/` 直下に残っている調査用の `package main`。認証情報を扱う可能性があるため、削除するか `tools/` へ隔離すべき。
