# 非機能要件 充足度診断レポート

- **診断日**: 2026-10-11
- **対象リビジョン**: `64270c7`（main）
- **診断方法**: 静的コードレビュー（設定ファイル・ミドルウェア・環境変数・リポジトリ／サービス層・フロントエンド API クライアント）。負荷計測などの動的検証は未実施。
- **要件定義**: `.claude/CLAUDE.md` の「非機能要件」セクション
- **更新履歴**:
  - 2026-10-11: 1-2（N+1）を解消し「達成」に更新。1-1 に実測値を追記。詳細は `docs/performance-timeline-report.md`
  - 2026-10-11: 3-4（レートリミット）を実装し「達成」に更新
  - 2026-10-11: 3-2（認証トークンの HttpOnly Cookie 化・リフレッシュトークンの DB 管理）を実装し「達成」に更新

---

## 1. サマリー

| # | 区分 | 項目 | ステータス | 優先度 |
|---|------|------|-----------|--------|
| 1-1 | 性能 | レスポンスタイム（一覧 500ms 以内） | 部分的 | Medium |
| 1-2 | 性能 | N+1 クエリを発生させない | 達成（2026-10-11 対応） | — |
| 1-3 | 性能 | アップロード制限（ミドルウェアで画像 5MB / 動画 50MB） | **未達成** | **High** |
| 1-4 | 性能 | ページネーション（全件取得禁止・20 件単位） | 達成 | — |
| 1-5 | 性能 | タイムアウト（API 30 秒 / DB 接続 5 秒） | **未達成** | Medium |
| 2-1 | 運用・保守 | 構造化ログ（JSON） | 部分的 | Medium |
| 2-2 | 運用・保守 | 自動テスト（ユニット + E2E） | 達成 | — |
| 2-3 | 運用・保守 | 安全なエラーメッセージ | 部分的 | Medium |
| 3-1 | セキュリティ | CORS（フロントのオリジンのみ・credentials: true） | 達成 | — |
| 3-2 | セキュリティ | 認証（HttpOnly Cookie / SameSite=None; Secure / 1h・7d） | 達成 | — |
| 3-3 | セキュリティ | 入力検証・サニタイズ | 部分的 | Medium |
| 3-4 | セキュリティ | レートリミット（認証 5 回/分・一般 60 回/分） | 達成 | — |
| 3-5 | セキュリティ | security-review の定期実施 | 部分的 | Low |

集計: **達成 6 / 部分的 5 / 未達成 2**（初回診断時は 達成 3 / 部分的 5 / 未達成 5）

### 既存ルールとの矛盾（要判断）

非機能要件は既存の `.claude/rules/*` と以下の点で食い違う。CLAUDE.md では「非機能要件を優先する」と定めたが、対応する rules ファイルの更新は未実施。

| 非機能要件 | 既存ルール | ファイル |
|-----------|-----------|---------|
| Access Token も HttpOnly Cookie で管理、1 時間 | Access Token は `localStorage`、15 分。「Access Token を Cookie に保存しない」と明記 | `rules/auth.md` |
| Cookie は `SameSite=None; Secure` | `SameSite=Strict` | `rules/auth.md` |
| 動画上限 50MB | 動画 100MB | `rules/api.md`, `rules/security.md` |
| 認証系 5 回/分（実装済み） | login 10 回/分・register 5 回/分（推奨扱い） | `rules/security.md` |
| 全入力をサニタイズ | 「文字列フィールドはサニタイズせず、バリデーションで制限する」 | `rules/security.md` |
| 20 件単位 | `limit` は最大 50 まで指定可 | `rules/api.md` |

---

## 2. 項目別評価

### 1. 性能

#### 1-1. レスポンスタイム（一覧取得 API 500ms 以内）

- **ステータス**: 部分的
- **根拠**:
  - 一覧系はすべて `LIMIT n+1` でページングされ、タイムライン用インデックス（`idx_posts_created_at`, `idx_posts_user_id`, `idx_likes_post_id` など）は `backend/migrations/00*_*.up.sql` に揃っている。
  - 一方、レスポンス時間を計測・記録する仕組みは存在しない（アクセスログに `latency` は出るが閾値監視・テストはない）。
- **不足点**:
  - 1-2 の N+1 により、投稿一覧 20 件で最大 1 + 7（Preload）+ 20 × 6 = 約 128 クエリが発行され、データ量増加で 500ms を超えるリスクが高い。
  - 500ms を検証するテスト・ベンチマークがない。
- **改善提案**:
  1. 1-2 の N+1 解消を先行する（最大の改善要因）。
  2. リクエスト単位で処理時間を計測し、500ms 超を `WARN` で記録するミドルウェアを追加する（2-1 の slog 化と同時に実装）。
  3. リポジトリ統合テストに「投稿 20 件 + いいね付きで `GetExplore` → `toPostListResponse` が 500ms 以内」のテストを 1 本追加する（CLAUDE.md の「ループは最小回数」ルールに従い 20 件まで）。
- **2026-10-11 追記（実測）**: `make perf-timeline` でタイムライン（20 件・認証あり）を計測。N+1 解消後、サーバー処理時間 平均 8.0ms（改善前 75.5ms）、SQL 8 本（改善前 128 本）で 500ms を十分に下回る。本番での 500ms 超の監視は未実装のため「部分的」のまま（TODO-9）。

#### 1-2. N+1 クエリを発生させない

- **ステータス**: 達成（2026-10-11 対応。以下の根拠・提案は初回診断時の内容）
- **対応内容**: 改善提案 1 の「テーブルごとの一括メソッド」ではなく、集計値と閲覧者の状態を相関サブクエリ 1 本で取る `PostRepository.FindStatsByIDs` / `FollowRepository.FindStatsByUserIDs` を追加し、`PostService.ToPostResponses` / `toUserResponsesWithStats` で一覧を一括変換するようにした。投稿者は `InnerJoins("User")` で本体クエリと同時に取得。タイムラインは 128 本 → 8 本（件数によらず一定）。詳細は `docs/performance-timeline-report.md`。
- **根拠**:
  - `backend/internal/service/post_service.go:265` `toPostListResponse` が投稿ごとに `ToPostResponse` を呼び、`post_service.go:284-310` で 1 投稿あたり以下を個別発行している。
    - `likeRepo.CountByPost` / `postRepo.CountComments` / `postRepo.CountReposts`（常に 3 クエリ）
    - ログイン時はさらに `likeRepo.IsLiked` / `bookmarkRepo.IsBookmarked` / `postRepo.FindActiveRepost`（+3 クエリ）
  - `backend/internal/service/user_service.go:237, 253-272`（フォロワー／フォロー中一覧）、`backend/internal/service/search_service.go:54-70`（ユーザー検索）も 1 ユーザーあたり `CountFollowers` / `CountFollowing` / `Exists` を個別発行。
  - `Preload`（`post_repository.go:135-147`）や `media_repository.go:38` `FindByPostIDs` のように一括取得している箇所もあるが、集計値が N+1 になっている。
- **不足点**: 一覧 API すべて（探索・ホーム・ユーザー投稿・コメント・ブックマーク・検索・ハッシュタグ・フォロワー一覧・ユーザー検索）で N+1 が発生。
- **改善提案**:
  1. リポジトリに一括取得メソッドを追加する。
     ```go
     // LikeRepository
     CountByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]int64, error)
     LikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
     // PostRepository
     CountCommentsByPostIDs / CountRepostsByPostIDs / RepostedPostIDs
     // BookmarkRepository
     BookmarkedPostIDs
     // FollowRepository
     CountFollowersByUserIDs / CountFollowingByUserIDs / FollowingUserIDs
     ```
     実装は `SELECT post_id, COUNT(*) FROM likes WHERE post_id IN ? GROUP BY post_id` 形式。
  2. `toPostListResponse` で ID を集めて一括取得 → map 参照で `PostResponse` を組み立てる。単体取得の `ToPostResponse` は 1 件の slice を渡して同じ処理を共有する。
  3. リポジトリ統合テストで「20 件取得時の発行クエリ数が件数に依存しない」ことを GORM の `Callback().Query().After` でカウントして検証する。

#### 1-3. アップロード制限（ミドルウェアで画像 5MB / 動画 50MB）

- **ステータス**: **未達成**
- **根拠**:
  - `backend/cmd/server/main.go:110-117` に `middleware.BodyLimit` がなく、リクエストボディ上限が未設定。
  - サイズ検証はファイル受信後のサービス層のみ（`backend/internal/service/storage_service.go:20-21` で画像 5MB・**動画 100MB**、`user_handler.go:200` でアバター 5MB）。
  - `storage_service.go` の `upload` は `io.ReadAll` でファイル全体をメモリに読み込む。
- **不足点**:
  - 動画上限が 100MB で要件（50MB）と不一致。
  - ミドルウェアでの上限がないため、巨大なボディを送られると multipart 解析（ディスク／メモリ展開）が完了するまで拒否できない。
- **改善提案**:
  1. `storage_service.go` の `maxVideoSize` を `50 * 1024 * 1024` に変更し、Swagger の説明（`post_handler.go:84`）とフロントの検証も 50MB に揃える。
  2. ルート単位で `BodyLimit` を適用する。
     ```go
     // 一般 API（JSON）は小さく、アップロード系だけ上限を引き上げる
     e.Use(middleware.BodyLimit("1M"))                         // グローバル
     uploadImage := middleware.BodyLimit("6M")                 // 画像 1 枚 + フォーム
     uploadPost  := middleware.BodyLimit("51M")                // 動画 50MB or 画像 5MB×4
     v1.POST("/posts", postHandler.CreatePost, uploadPost, jwtMiddleware)
     v1.POST("/posts/:id/comments", postHandler.CreateComment, uploadPost, jwtMiddleware)
     v1.PUT("/users/me/avatar", userHandler.UpdateAvatar, uploadImage, jwtMiddleware)
     v1.PUT("/users/me/banner", userHandler.UpdateBanner, uploadImage, jwtMiddleware)
     ```
     グローバルの `BodyLimit` はルート個別より先に評価されるため、グローバルは `51M`、JSON 系ルートグループに `1M` を付ける構成でもよい。413 を `{code: "PAYLOAD_TOO_LARGE", message: "ファイルサイズが上限を超えています"}` に変換する（2-3 の共通エラーハンドラーで対応）。
  3. `io.ReadAll` をやめ、`io.Copy(w, src)` でストリーミング転送する。
  4. ハンドラーテストで 51MB 超のボディが 413 になることを検証する。

#### 1-4. ページネーション（全件取得禁止・20 件単位）

- **ステータス**: 達成
- **根拠**:
  - 全一覧ハンドラー（`post_handler.go:41,68,249`、`user_handler.go:368,396,424,452`、`interaction_handler.go:265`、`notification_handler.go:41`、`search_handler.go:42,70,95`、`admin_handler.go:127`）が `utils.ParseLimit(…, defaultListLimit=20, maxListLimit=50)`（`user_handler.go:22-23`、`utils/cursor.go:22`）を通している。
  - リポジトリは `created_at` によるカーソル + `LIMIT n+1` で `hasMore` を判定（`post_repository.go:149-164` など）。
- **不足点**（軽微）: クライアントが `limit=50` を指定できるため、厳密な「20 件単位」ではない。カーソルが `created_at` 単独のため、同一時刻の投稿が境界にあると取りこぼす可能性がある。
- **改善提案**（任意・Low）: 要件を厳密に満たすなら `maxListLimit` を 20 にする。カーソルを `(created_at, id)` の複合にして `WHERE (created_at, id) < (?, ?)` で比較する。

#### 1-5. タイムアウト（API 30 秒 / DB 接続 5 秒）

- **ステータス**: **未達成**
- **根拠**:
  - `backend/cmd/server/main.go:196` は `e.Start(addr)` のみで、`http.Server` の `ReadTimeout` / `WriteTimeout` 未設定。`middleware.TimeoutWithConfig` / `ContextTimeout` もなし。
  - `backend/internal/config/db.go:19` の `gorm.Open` は接続タイムアウト指定なし。`.env.example` / `backend/.env.test` の `DATABASE_URL` にも `connect_timeout` がない。コネクションプール設定（`SetMaxOpenConns` 等）もなし。
  - フロントの Axios（`frontend/src/api/client.ts:10-13`）にも `timeout` 未設定。
- **不足点**: 応答しないリクエスト・DB 障害時にゴルーチンと接続が無期限に滞留する。
- **改善提案**:
  1. サーバー側:
     ```go
     e.Use(middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{Timeout: 30 * time.Second}))
     e.Server.ReadHeaderTimeout = 10 * time.Second
     e.Server.ReadTimeout = 30 * time.Second
     e.Server.WriteTimeout = 35 * time.Second // ContextTimeout より少し長く
     ```
     リポジトリはすでに `WithContext(ctx)` を使っているため、コンテキストのキャンセルが DB クエリまで伝搬する。
  2. DB 側: `config.NewDB` で `DATABASE_URL` に `connect_timeout=5` が無ければ付与する（または `pgx` の `ConnectTimeout` を設定）。あわせて `sqlDB.SetMaxOpenConns(25)`, `SetMaxIdleConns(10)`, `SetConnMaxLifetime(30*time.Minute)` を設定。`.env.example` / `.env.test` / `ci.yml` の `DATABASE_URL` にも `connect_timeout=5` を追記。
  3. フロント: `axios.create({ timeout: 30_000 })`。動画アップロードだけ個別に延長するかは要件次第。
  4. `config_test.go` に「`connect_timeout` が付与される」テストを追加。

---

### 2. 運用・保守性

#### 2-1. 構造化ログ（アクセスログ・エラーログを JSON で記録）

- **ステータス**: 部分的
- **根拠**:
  - アクセスログ: `backend/cmd/server/main.go:110` の `middleware.Logger()`。Echo v4 のデフォルトフォーマットは JSON（`time`, `id`, `remote_ip`, `method`, `uri`, `status`, `latency` など）であり達成。
  - エラーログ:
    - 起動・設定系は標準 `log`（`main.go:79,81`、`config/config.go:35,38`、`config/db.go:23,38`、`cmd/migrate/main.go:19`）でプレーンテキスト。
    - GORM のロガー（`config/db.go:20` `logger.Default`）もプレーンテキスト。
    - ハンドラーで 500 を返す箇所が 43 箇所あるが、原因エラーをログ出力しているのは `post_handler.go` の `respondPostMediaError` 等 2 箇所のみ。大半の 500 は**原因がどこにも記録されない**。
  - `backend/cmd/listbuckets_main.go` に `fmt.Println` が残っている（`rules/backend.md` の禁止事項にも抵触）。
- **不足点**: エラーログの JSON 化、500 エラー原因の記録、リクエスト ID との相関。
- **改善提案**:
  1. `log/slog` の `slog.NewJSONHandler(os.Stdout, …)` を `main` で `slog.SetDefault` し、`log.*` を `slog.*` に置換する（`log.Fatalf` は `slog.Error` + `os.Exit(1)`）。
  2. `middleware.RequestID()` を追加し、アクセスログを `middleware.RequestLoggerWithConfig` + slog に統一（`request_id`, `status`, `latency_ms`, `user_id` を出力）。
  3. 2-3 の共通エラーハンドラーで 5xx の原因エラーを `slog.ErrorContext(ctx, "request failed", "err", err, "request_id", id)` で記録する。
  4. GORM は `slog` をラップしたロガー（`logger.New` に slog ベースの Writer）か、本番では `logger.Silent` + 遅いクエリのみ記録（`SlowThreshold: 200ms`）。
  5. 不要な `backend/cmd/listbuckets_main.go` は削除する。

#### 2-2. 自動テスト（ユニットテスト・E2E テスト）

- **ステータス**: 達成
- **根拠**:
  - バックエンド: `internal/` 配下に handler / service / middleware / repository / config / utils のテスト（Test 関数 約 210 件）。`make test-backend` でカバレッジ集計。
  - フロントエンド単体: `PostCard.test.tsx` / `PostForm.test.tsx` / `PrivateRoute.test.tsx` / `AuthContext.test.tsx`。
  - E2E: `frontend/e2e/*.spec.ts` 7 ファイル（auth / posts / comments / profile / search / notifications / admin）。
  - CI: `.github/workflows/ci.yml` で backend-test / frontend-build / e2e-test を実行。
- **不足点**（軽微）: フロントエンド単体テストは 4 ファイルのみで、カスタムフック（`useLike` など `rules/testing.md` の優先項目）のテストがない。CI にフロントエンド単体テスト（Vitest）ジョブがない。
- **改善提案**（任意・Low）: `ci.yml` に `npm run test` ジョブを追加。`useLike` / `useFollow` の楽観的更新を `renderHook` でテストする。

#### 2-3. エラーハンドリング（適切で安全なエラーメッセージ）

- **ステータス**: 部分的
- **根拠**:
  - 業務エラーは `dto.ErrorResponse{Code, Message}` の統一形式・日本語（`internal/dto/common.go`）で返され、内部エラーは `"サーバーエラーが発生しました"` に丸められている（例: `auth_handler.go:62`）。
  - バリデーションエラーは `handler.validationMessage()` で日本語化（`internal/handler/validation.go`）。
  - 一方、`e.HTTPErrorHandler` は未設定（`main.go`）。そのため以下は Echo デフォルトの `{"message":"Not Found"}` 形式・英語で返る。
    - 未定義ルート（404）、メソッド不一致（405）
    - `middleware.Recover()` が捕捉した panic（500）
    - 1-3 で追加する `BodyLimit` の 413
- **不足点**: エラーレスポンス形式の統一が Echo 標準エラーに及んでいない。5xx 原因のログがない（2-1）。
- **改善提案**:
  1. `internal/handler/error_handler.go` に共通ハンドラーを作り `e.HTTPErrorHandler = handler.NewHTTPErrorHandler(cfg)` を設定する。
     - `*echo.HTTPError` を `404 → NOT_FOUND「ページが見つかりません」`、`405 → METHOD_NOT_ALLOWED`、`413 → PAYLOAD_TOO_LARGE`、`429 → TOO_MANY_REQUESTS`、それ以外 → `INTERNAL_ERROR「サーバーエラーが発生しました」` に変換。
     - 5xx は slog で原因を記録。本番ではメッセージに内部情報を一切含めない。
  2. ハンドラーテストで未定義ルート・panic が統一形式の日本語で返ることを検証する。

---

### 3. セキュリティ

#### 3-1. CORS（フロントエンドのオリジンのみ許可・credentials: true）

- **ステータス**: 達成
- **根拠**:
  - `backend/cmd/server/main.go:112-117` で `AllowOrigins: cfg.AllowedOrigins()`（環境変数 `CORS_ORIGINS`、`config.go:29,66-75`）、`AllowCredentials: true`。ワイルドカードは不使用。
  - フロントは `withCredentials: true`（`frontend/src/api/client.ts:12,38`）。
- **不足点**（軽微）: 本番で `CORS_ORIGINS` 未設定だと `http://localhost:5173` にフォールバックする。`*` が設定された場合の拒否もない。
- **改善提案**（任意・Low）: `config.Load` で `IsProduction()` かつ `CORS_ORIGINS` 未設定、または `*` を含む場合に起動失敗させる。

#### 3-2. 認証（HttpOnly Cookie・SameSite=None; Secure・アクセス 1 時間 / リフレッシュ 7 日）

- **ステータス**: 達成（2026-10-11 対応）
- **対応内容**:
  - Access Token（JWT、ペイロードは `user_id` / `exp` のみ、1 時間）と Refresh Token（乱数、7 日）を `HttpOnly; Secure; SameSite=None` の Cookie で発行（`handler/auth_handler.go`、`internal/authtoken/`）。レスポンスボディにはトークンを含めない。
  - Refresh Token は DB（`refresh_tokens`）に HMAC ハッシュで保存し、`POST /auth/refresh` でローテーション。使用済みトークンの再利用を検知したら系列ごと失効。`POST /auth/logout` は失効 + Cookie 削除、`POST /auth/revoke-all` で全デバイスの Refresh Token を失効。
  - CSRF 対策として状態変更リクエストの Origin を検証（`middleware/origin_check.go`）。管理者判定はトークンではなく DB のロールで行う。
  - フロントエンドは `withCredentials: true` のみ。401 で自動リフレッシュ → 再送、失敗したらログイン画面へ（`src/api/client.ts`、`AuthContext.tsx`）。
  - 開発環境は既定で同じ属性（主要ブラウザは http://localhost で Secure Cookie を扱える）。Safari 等のみ `COOKIE_SECURE=false` で `SameSite=Lax` に切り替え可能。
- **初回診断時の根拠**:
  - Access Token はレスポンスボディで返し（`auth_handler.go:68-71,115-118`）、フロントが `localStorage` に保存（`frontend/src/api/client.ts:4-8`）、`Authorization` ヘッダーで送信（`client.ts:15-21`、`middleware/auth.go` の `JWTAuth`）。→ HttpOnly Cookie 管理ではない。
  - Access Token 有効期限は **15 分**（`internal/service/auth_service.go:278`）。Refresh Token は 7 日（`auth_service.go:289`、`auth_handler.go` `setRefreshTokenCookie`）で要件どおり。
  - Refresh Token Cookie は `HttpOnly: true, Secure: true` だが **`SameSite: Strict`**（`auth_handler.go` `setRefreshTokenCookie` / `Logout`）。
- **不足点**: Access Token の保存場所・有効期限・SameSite 属性の 3 点が要件と不一致。`localStorage` 保存は XSS 時にトークンが窃取される。
- **改善提案**:
  1. バックエンド:
     - `setAuthCookies(c, access, refresh)` を作り、`access_token`（`MaxAge` 3600, `Path: /`）と `refresh_token`（`MaxAge` 7 日, `Path: /api/v1/auth`）を `HttpOnly; Secure; SameSite=None` で設定。`Logout` は両方を削除。
     - `auth_service.go:278` を `time.Hour` に変更。
     - `JWTAuth` / `OptionalJWTAuth` は Cookie `access_token` を優先し、移行期間のみ `Authorization` ヘッダーも受け付ける。
     - `SameSite=None` は CSRF 防御を失うため、状態変更メソッド（POST/PUT/DELETE）に CSRF 対策を必ず追加する: `middleware.CSRFWithConfig`（Double Submit Cookie、`TokenLookup: "header:X-CSRF-Token"`）または `Origin` ヘッダーを `CORS_ORIGINS` と照合するミドルウェア。
     - 開発環境（http://localhost）では `Secure` Cookie が使えないブラウザがあるため、`SameSite=None; Secure` は `localhost` 例外で動作するか確認し、不可なら開発時のみ `Lax` にする分岐を `cfg.IsDevelopment()` で実装。
  2. フロントエンド:
     - `client.ts` の `localStorage` 操作と `Authorization` 付与を削除し、`withCredentials: true` のみにする。
     - ログイン状態判定は `GET /users/me` の成否で行う（`utils/jwt.ts` のトークンデコード依存を除去）。
     - CSRF トークンをヘッダーに付与するインターセプターを追加。
  3. Swagger の `securityDefinitions` を `apiKey in: cookie` に変更し `swag init` → `openapi.yaml` 再生成。
  4. `rules/auth.md`・`rules/frontend.md` を新方式に更新（現行ルールは Cookie 保存を禁止しているため）。
  5. テスト: ミドルウェアテスト（Cookie 認証・期限切れ・CSRF 不一致で 403）、E2E（ログイン → リロード後もログイン維持）。

#### 3-3. 入力検証・サニタイズ

- **ステータス**: 部分的
- **根拠**:
  - JSON ボディは DTO の `validate` タグ（`internal/dto/*.go` で 19 箇所）+ `h.validate.Struct` で検証。
  - アップロードはバイト列から MIME 判定（`storage_service.go` の `mimetype.Detect`）。
  - パスパラメータの UUID は `uuid.Parse` で検証。`limit` は `ParseLimit` で数値化・上限適用。
  - SQL はすべてプレースホルダー、表示は React のエスケープに依存（`dangerouslySetInnerHTML` 不使用）。
  - `ILIKE` 検索語の `\` `%` `_` は `search_repository.go:29-34` の `likeEscaper` でエスケープ済み。
- **不足点**:
  - クエリパラメータの検証が不揃い: `search_handler.go:37-38` のユーザー検索は最小 2 文字のみで最大長なし、`search_handler.go:65` の投稿検索は長さ検証なし。ハッシュタグ `:tag` も長さ・文字種未検証。`cursor` の不正値は各ハンドラー任せ。
  - サニタイズ（制御文字・不可視文字の除去、前後空白のトリム）は未実施。`rules/security.md` は「サニタイズしない」方針で、要件と矛盾している。
- **改善提案**:
  1. 検索・ハッシュタグ用のクエリ DTO（例: `SearchQuery{Q string validate:"required,min=1,max=100"}`）を作り `c.Bind` + `validate` に統一する。
  2. `internal/utils/sanitize.go` に `SanitizeText(s string) string`（`strings.TrimSpace`、`unicode.IsControl` のうち改行・タブ以外を除去、ゼロ幅文字 U+200B〜U+200D, U+FEFF 除去）を作り、投稿本文・プロフィール・検索語に適用。HTML エスケープは保存時には行わない（表示時に React がエスケープするため二重エスケープを避ける）。
  3. `rules/security.md` のサニタイズ方針を上記に合わせて更新する。
  4. `utils_test.go` に `SanitizeText` のテーブルテスト、ハンドラーテストに 101 文字検索語の 400 を追加。

#### 3-4. レートリミット（認証系 5 回/分・一般 60 回/分）

- **ステータス**: 達成（2026-10-11 対応）
- **対応内容**: `internal/middleware/rate_limit.go` にインメモリ・fixed window（1 分）のレートリミッターを実装（Cloud Run max instances = 1 前提。mutex で保護）。
  - 認証系（`/auth/login`・`/auth/register`・`/auth/google`）: IP 単位・API ごとに 5 回/分。一般（それ以外の `/api/v1/*`）: 認証済みはユーザーID、未認証は IP 単位・API ごとに 60 回/分。
  - 全レスポンスに `X-RateLimit-Limit` / `X-RateLimit-Remaining` / `X-RateLimit-Reset`（UNIX 秒）、超過時は 429 + `Retry-After`（秒）+ `RATE_LIMIT_EXCEEDED`。
  - クライアントIPは `ClientIPExtractor`（`X-Forwarded-For` の右端から `TRUSTED_PROXY_HOPS` 番目。本番既定 1、開発・テストは接続元アドレス）。
  - 上限の緩和は `ENV=test` のみ（`RATE_LIMIT_AUTH_PER_MINUTE` / `RATE_LIMIT_GENERAL_PER_MINUTE`）。
  - 以下の「改善提案」は初回診断時の案（token bucket の Echo 標準ミドルウェア）で、実装では要件どおり fixed window を採用した。
- **初回診断時の根拠**: `backend/cmd/server/main.go` に `middleware.RateLimiter` 等の適用なし。`golang.org/x/time/rate` も未使用。`docs/SECURITY_REVIEW_PHASE01.md` の P01-05 でも指摘済み（未対応のまま）。
- **不足点**: ログイン総当たり・登録スパム・API 乱用を防げない。
- **改善提案**:
  1. Echo 標準の `middleware.RateLimiterWithConfig` + `NewRateLimiterMemoryStoreWithConfig` を使う。
     ```go
     func newRateLimiter(perMinute int) echo.MiddlewareFunc {
         return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
             Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
                 Rate: rate.Limit(float64(perMinute) / 60), Burst: perMinute, ExpiresIn: 3 * time.Minute,
             }),
             IdentifierExtractor: func(c echo.Context) (string, error) { return c.RealIP(), nil },
             DenyHandler: func(c echo.Context, _ string, _ error) error {
                 return c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{Code: "TOO_MANY_REQUESTS", Message: "リクエストが多すぎます。しばらくしてから再度お試しください"})
             },
         })
     }
     auth := v1.Group("/auth", newRateLimiter(5))   // 認証系 5 回/分
     v1.Use(newRateLimiter(60))                       // 一般 60 回/分（auth にも重ねて掛かる点に注意）
     ```
     認証系に一般リミットが二重に掛からないよう、一般用は `Skipper` で `/api/v1/auth/` を除外する。
  2. 認証済みリクエストは `userID` をキーにし、未認証は IP をキーにする。
  3. `c.RealIP()` が正しく動くよう、本番（Render / Cloud Run）では `e.IPExtractor = echo.ExtractIPFromXFFHeader(…)` を設定。
  4. E2E は同一 IP から多数ログインするため、`ENV=test` では上限を緩める（`config` に `RATE_LIMIT_AUTH` / `RATE_LIMIT_GENERAL` を追加し、`.env.test` / `ci.yml` で大きな値を設定）。
  5. ハンドラーテストで 6 回目のログインが 429 になることを検証。
  6. 複数インスタンス構成にする場合は Redis ストアへの置き換えを検討（現状は単一インスタンス前提でメモリストアで可）。

#### 3-5. security-review による定期的なセキュリティレビュー

- **ステータス**: 部分的
- **根拠**: `docs/SECURITY_REVIEW_PHASE01.md`（2026-08-23、PHASE01 範囲のみ）が 1 回分存在する。定期実行の仕組み（CI ジョブ・スケジュール・チェックリスト）はない。同レポートの指摘（例: P01-05 レートリミット）が未対応のまま残っている。
- **不足点**: 定期性と、指摘事項の追跡がない。PHASE02 以降（認証・投稿・アップロード）はレビュー対象外。
- **改善提案**:
  1. 運用ルールを CLAUDE.md に明記（本対応で追記済み）: `main` へのマージ前に `/security-review` を実行し、結果を `docs/security-reviews/YYYY-MM-DD.md` に保存する。
  2. 機械的チェックを CI に追加: `govulncheck ./...`（Go 依存の既知脆弱性）、`npm audit --audit-level=high`、`gosec ./...`。
  3. 既存レポートの未対応項目を本レポートの TODO と統合して追跡する。
  4. 月 1 回の定期レビューは Claude Code の `/schedule` で自動化する案もある。

---

## 3. TODO（未達成・部分的項目）

優先度の基準: **High** = 攻撃・障害に直結する / 要件と明確に矛盾、**Medium** = 運用品質に影響、**Low** = 改善推奨。

### High

#### TODO-1: N+1 クエリの解消（1-2）— 完了（2026-10-11）
- [x] 投稿の一括集計メソッドを追加（`PostRepository.FindStatsByIDs`。相関サブクエリ 1 本に集約）
- [x] `FollowRepository` に一括集計メソッドを追加（`FindStatsByUserIDs`）
- [x] `postService.toPostListResponse` を一括取得 + map 参照に書き換え（`ToPostResponses`。ブックマーク・検索も同様）
- [x] `userService`（フォロー一覧）・`searchService.SearchUsers` を同様に書き換え
- [x] リポジトリ統合テスト（クエリ数が件数に依存しない）・サービス単体テストを追加

#### TODO-2: レートリミットの導入（3-4）— 完了（2026-10-11）
- [x] 自前の fixed window ミドルウェア（`internal/middleware/rate_limit.go`）で認証系 5 回/分・一般 60 回/分を適用（429 は統一エラー形式 + `Retry-After`）
- [x] `config` に `RATE_LIMIT_AUTH_PER_MINUTE` / `RATE_LIMIT_GENERAL_PER_MINUTE`（ENV=test のみ有効）を追加し、`.env.test` / `ci.yml` で E2E 用に緩和
- [x] `IPExtractor` 設定（`ClientIPExtractor`。`TRUSTED_PROXY_HOPS`、本番既定 1）
- [x] 429・ヘッダー・識別子・ウィンドウ切替・同時アクセスのテストを追加（`rate_limit_test.go`）
- [ ] パスワードリセットAPIを実装する際に `authRateLimit` を付ける（現状APIなし）

#### TODO-3: 認証トークンの HttpOnly Cookie 化（3-2）— 完了（2026-10-11）
- [x] Access Token を `HttpOnly; Secure; SameSite=None` Cookie で発行、有効期限 1 時間（ペイロードは user_id / exp のみ）
- [x] Refresh Token Cookie を `SameSite=None` に変更。DB（`refresh_tokens`）でローテーション・再利用検知・失効を管理
- [x] `JWTAuth` / `OptionalJWTAuth` を Cookie 対応
- [x] CSRF 対策（Origin 検証。`middleware/origin_check.go`）を状態変更 API に追加
- [x] フロント: `localStorage` 保存・`Authorization` 付与を廃止（Origin 検証方式のため CSRF ヘッダーは不要）
- [x] `swag init` → `openapi.yaml` 再生成（Swagger 2.0 は Cookie 認証を表現できないため `BearerAuth` の説明文のみ更新）
- [x] `rules/auth.md` / `rules/frontend.md`（ほか backend / security / api）を更新
- [x] ユニットテスト（発行・検証・リフレッシュ・失効・Cookie 属性）と E2E（`e2e/token.spec.ts`：期限切れ → 自動リフレッシュ → 操作継続など）を追加
- [ ] 期限切れ・失効済みリフレッシュトークンの定期削除（テーブルが増え続けるため）

#### TODO-4: リクエストボディ上限のミドルウェア化（1-3）
- [ ] `middleware.BodyLimit` を JSON 系（1MB）・画像アップロード（6MB）・投稿（51MB）で適用
- [ ] `maxVideoSize` を 50MB に変更（Swagger 説明・フロント検証・`rules/api.md` も合わせる）
- [ ] `storage_service.upload` の `io.ReadAll` をストリーミングに変更
- [ ] 413 を検証するテストを追加

### Medium

#### TODO-5: タイムアウト設定（1-5）
- [ ] `middleware.ContextTimeout`（30 秒）と `e.Server` の `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout`
- [ ] `DATABASE_URL` に `connect_timeout=5`（`config.NewDB` で補完 + `.env.example` / `.env.test` / `ci.yml`）
- [ ] DB コネクションプール設定
- [ ] Axios `timeout: 30000`
- [ ] `config_test.go` にテスト追加

#### TODO-6: 構造化ログの統一（2-1）
- [ ] `slog.NewJSONHandler` を `slog.SetDefault`、`log.*` を置換
- [ ] `middleware.RequestID()` + `RequestLoggerWithConfig` でアクセスログを slog に統一
- [ ] 5xx の原因エラーをログ出力
- [ ] GORM ロガーを slog 化 / 本番は遅いクエリのみ
- [ ] `backend/cmd/listbuckets_main.go` を削除

#### TODO-7: 共通 HTTPErrorHandler（2-3）
- [ ] `handler.NewHTTPErrorHandler` を実装し `e.HTTPErrorHandler` に設定（404/405/413/429/500 を統一形式・日本語に）
- [ ] 未定義ルート・panic のハンドラーテストを追加

#### TODO-8: 入力検証の統一とサニタイズ（3-3）
- [ ] 検索語・ハッシュタグのクエリ DTO と `validate` タグ（最大長）
- [ ] `utils.SanitizeText` を実装し投稿本文・プロフィール・検索語に適用
- [ ] `rules/security.md` のサニタイズ方針を更新
- [ ] テーブルテスト・ハンドラーテストを追加

#### TODO-9: レスポンスタイムの計測（1-1）
- [x] タイムラインの応答時間・SQL 本数の計測スクリプト（`make perf-timeline`、`X-DB-Query-Count` ヘッダー）
- [ ] TODO-1 完了後、500ms 超を WARN 記録するミドルウェア（TODO-6 と統合）
- [ ] 一覧取得 20 件の処理時間を検証する統合テスト

### Low

#### TODO-10: security-review の定期実施（3-5）
- [ ] `main` マージ前の `/security-review` 実行と `docs/security-reviews/` への保存を運用化
- [ ] CI に `govulncheck` / `npm audit` / `gosec` を追加
- [ ] `SECURITY_REVIEW_PHASE01.md` の未対応項目を追跡

#### TODO-11: 達成項目の改善（任意）
- [ ] ページネーション: 複合カーソル `(created_at, id)` 化、`maxListLimit` を 20 にするか判断（1-4）
- [ ] 本番で `CORS_ORIGINS` 未設定・`*` の場合に起動失敗させる（3-1）
- [ ] CI にフロントエンド単体テストジョブを追加、フックのテスト追加（2-2）
