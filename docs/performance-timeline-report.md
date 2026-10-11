# タイムライン取得API 性能改善レポート（N+1 クエリ解消）

- **実施日**: 2026-10-11
- **対象**: `GET /api/v1/posts?limit=20`（探索タイムライン。ホームタイムラインも同じクエリ）
- **関連する非機能要件**: 性能「レスポンスタイム：一覧取得API 500ms 以内」「N+1 クエリを発生させない」（`.claude/CLAUDE.md`）
- **関連レポート**: `docs/non-functional-requirements-report.md`（1-1, 1-2 / TODO-1, TODO-9）

---

## 1. 結果サマリー

| 指標 | 改善前 | 改善後 | 変化 |
|------|-------:|-------:|------|
| 1 リクエストあたりの SQL 本数 | **128 本** | **8 本** | −120 本（−93.8%） |
| サーバー処理時間 平均（アクセスログの latency） | **75.5 ms** | **8.0 ms** | −67.5 ms（−89.4%、約 9.4 倍速） |
| サーバー処理時間 最小 / 最大 | 66.8 / 84.5 ms | 5.9 / 10.8 ms | |
| クライアント応答時間 平均（curl `time_total`） | **87.6 ms** | **23.7 ms** | −63.9 ms（−72.9%） |
| クライアント応答時間 最小 / 最大 | 73.1 / 105.9 ms | 9.6 / 38.4 ms | |
| 1 リクエスト分の SQL 実行時間合計（SQL ログ） | 76.6 ms | 13.7 ms | −82.1% |
| SQL 本数と投稿件数の関係 | 8 + 6 × 件数（**N+1**） | 件数によらず一定 | |

- 改善前・改善後とも、非機能要件の「一覧取得 API 500ms 以内」は満たしている。ただし改善前は SQL 本数が `8 + 6 × 件数` で増えるため、`limit=50` で 308 本になり、DB が遠い本番環境（ネットワーク往復が 1 本あたり 1ms 以上）では件数・データ量に比例して悪化する構造だった。
- 改善後は件数・閲覧者の有無によらず **8 本で一定**になった。

> 数値はローカルの Docker Desktop（Windows）上のテスト環境での計測値。絶対値は環境に依存するため、改善前後の**比率**と **SQL 本数**を主な評価指標とする。

---

## 2. 計測方法

### 2.1 環境（開発データには一切触れない）

| 項目 | 内容 |
|------|------|
| API | テスト用 API サーバー `api_test`（`ENV=test`、ホストの `127.0.0.1:8081`） |
| DB | テスト用 PostgreSQL 16 `db_test`（tmpfs。計測後に `make test-teardown` で破棄） |
| 実行 | `make perf-timeline LABEL=before` / `make perf-timeline LABEL=after` |
| ホスト | Windows 11 / Docker 29.5.3 / Docker Compose v5.1.4 |

`make perf-timeline` は `SQL_LOG=true` でテスト環境を起動し → `scripts/perf/measure-timeline.sh` で計測し → テスト環境を破棄する。開発用 `db` / `api` コンテナには触れない（スクリプトは `:8081` 以外の API を指定すると実行を拒否する）。

### 2.2 テストデータ（`scripts/perf/seed.sql`）

| データ | 件数 | 備考 |
|--------|-----:|------|
| ユーザー | 10 | 計測用ユーザー（API で登録してトークンを取得）+ 著者 9 人 |
| 投稿 | 550 | 通常投稿 300 / 返信 200 / リポスト 50 |
| メディア | 100 | 通常投稿の 3 件に 1 件 |
| いいね | 667 | 新しい 100 投稿にユーザーの約 2/3 が いいね |
| ブックマーク | 75 | 計測用ユーザーが 4 件に 1 件 |
| フォロー | 9 | 計測用ユーザー → 著者 9 人 |

### 2.3 計測手順

1. 計測用ユーザーを `POST /api/v1/auth/register` で登録し、アクセストークンを取得
2. `seed.sql` を `db_test` に投入（`ANALYZE` 実行済み）
3. ウォームアップ 1 回（集計対象外）の後、`Authorization: Bearer <token>` 付きで 10 回リクエスト
4. 各回の `time_total`（クライアント）、`X-DB-Query-Count` ヘッダー（SQL 本数）、アクセスログの `latency`（サーバー）を集計
5. 追加で 1 回リクエストし、その SQL ログを `.perf-results/timeline-<label>-sql.log` に保存（ヘッダーの本数と一致することを確認）

### 2.4 SQL 本数の計測の仕組み（今回追加）

| 追加物 | 内容 |
|--------|------|
| `internal/querycount` | GORM の全コールバック（Query / Create / Update / Delete / Row / Raw）でクエリ数をコンテキスト単位に数える。Preload の各クエリも 1 本として数える |
| `middleware.QueryCount()` | リクエストごとにカウンターを用意し、`X-DB-Query-Count` レスポンスヘッダーと `db query count` ログに出力 |
| `config.ShouldCountQueries()` | 上記は **開発・テスト環境のみ有効**（本番では DB 構成の情報を返さない） |
| `config.ShouldLogSQL()` / `SQL_LOG` | SQL ログ（全クエリ）は開発環境では常に、テスト環境では `SQL_LOG=true` のときだけ出力。本番では常に出力しない |
| `docker-compose.yml` | `api_test` に `SQL_LOG: ${SQL_LOG:-false}` を追加 |

---

## 3. 計測結果の詳細

### 3.1 各リクエストの値

| # | 改善前 応答時間 (ms) | 改善前 SQL 本数 | 改善後 応答時間 (ms) | 改善後 SQL 本数 |
|--:|---------------------:|----------------:|---------------------:|----------------:|
| 1 | 73.1 | 128 | 23.3 | 8 |
| 2 | 87.1 | 128 | 25.1 | 8 |
| 3 | 86.3 | 128 | 38.4 | 8 |
| 4 | 105.9 | 128 | 32.9 | 8 |
| 5 | 77.5 | 128 | 16.3 | 8 |
| 6 | 80.1 | 128 | 23.6 | 8 |
| 7 | 91.5 | 128 | 9.6 | 8 |
| 8 | 91.8 | 128 | 11.2 | 8 |
| 9 | 92.8 | 128 | 35.1 | 8 |
| 10 | 90.1 | 128 | 21.7 | 8 |
| **平均** | **87.6** | **128** | **23.7** | **8** |

### 3.2 SQL の内訳

**改善前（128 本 = 8 + 6 × 20 件）**

| 本数 | クエリ | 発生元 |
|-----:|--------|--------|
| 1 | `SELECT posts.* FROM posts JOIN users ... LIMIT 21` | 本体 |
| 7 | `users` / `media` / `posts`（リポスト元・返信先）/ その `users`・`media` の `IN (...)` | Preload（件数によらず一定） |
| **20** | `SELECT count(*) FROM likes WHERE post_id = ?` | `ToPostResponse` の `likeRepo.CountByPost` |
| **20** | `SELECT count(*) FROM posts WHERE reply_to = ? AND is_deleted = false` | `postRepo.CountComments` |
| **20** | `SELECT count(*) FROM posts WHERE repost_of = ? AND is_deleted = false` | `postRepo.CountReposts` |
| **20** | `SELECT count(*) FROM likes WHERE user_id = ? AND post_id = ?` | `likeRepo.IsLiked` |
| **20** | `SELECT count(*) FROM bookmarks WHERE user_id = ? AND post_id = ?` | `bookmarkRepo.IsBookmarked` |
| **20** | `SELECT * FROM posts WHERE user_id = ? AND repost_of = ? ... LIMIT 1` | `postRepo.FindActiveRepost` |

**改善後（8 本、件数によらず一定）**

| 本数 | クエリ | 発生元 |
|-----:|--------|--------|
| 1 | `SELECT posts.*, "User".* FROM posts INNER JOIN users "User" ... LIMIT 21` | 本体（投稿者を JOIN で同時取得） |
| 6 | `media` / `posts`（リポスト元・返信先）/ その `users`・`media` の `IN (...)` | Preload（件数によらず一定） |
| 1 | `SELECT p.id, (SELECT COUNT(*) ...) AS likes_count, ..., EXISTS (...) AS is_reposted FROM posts p WHERE p.id IN (...)` | `postRepo.FindStatsByIDs`（集計値と閲覧者の状態を一括取得） |

---

## 4. 実施した改善

### 4.1 N+1 が発生していた箇所

| 箇所 | 1 件あたりの追加クエリ | 影響する API |
|------|----------------------:|--------------|
| `service/post_service.go` `ToPostResponse`（一覧では `toPostListResponse` から件数分呼ばれる） | 3 本（未ログイン）/ 6 本（ログイン） | 探索・ホーム・ユーザー投稿・返信一覧・コメント一覧 |
| `service/bookmark_service.go` `GetBookmarks` | 6 本 | ブックマーク一覧 |
| `service/search_service.go` `toPostListResponse` | 3 / 6 本 | 投稿検索・ハッシュタグ検索 |
| `service/user_service.go` `toUserListResponse` | 2 / 3 本 | フォロワー一覧・フォロー中一覧 |
| `service/search_service.go` `SearchUsers` | 2 / 3 本 | ユーザー検索 |

### 4.2 対応内容

| 変更 | 内容 |
|------|------|
| `PostRepository.FindStatsByIDs`（新規） | いいね数・コメント数・リポスト数と、閲覧者の いいね済み／ブックマーク済み／リポスト済みを**相関サブクエリ 1 本**で一括取得。各サブクエリは既存インデックス（`likes.post_id`・`posts.reply_to`・`posts.repost_of` 等）で解決される |
| `FollowRepository.FindStatsByUserIDs`（新規） | フォロワー数・フォロー数・閲覧者のフォロー状態を 1 本で一括取得 |
| `PostService.ToPostResponses`（新規） | 投稿 ID をまとめて `FindStatsByIDs` を 1 回だけ呼び、map 参照でレスポンスを組み立てる。単体の `ToPostResponse` もこれに委譲（投稿詳細・作成・編集も 6 本 → 1 本） |
| `toUserResponsesWithStats`（新規） | ユーザー一覧（フォロワー・フォロー中・ユーザー検索）で `FindStatsByUserIDs` を 1 回だけ呼ぶ |
| `basePostQuery` / `SearchPosts` | 停止ユーザー除外のための `JOIN users` + `Preload("User")` を `InnerJoins("User")` に置き換え、投稿者を同じクエリで取得（−1 本） |
| `bookmark_service` / `search_service` | 件数分の `ToPostResponse` ループを `ToPostResponses` 1 回に置き換え |

集計値（件数）は GORM の `Preload` では取得できないため、`Preload` / `Joins` に加えて「ID をまとめた一括集計クエリ」で解消した。

### 4.3 追加したテスト

| テスト | 検証内容 |
|--------|----------|
| `repository/stats_repository_test.go` `TestPostRepository_FindStatsByIDs` | 集計値（削除済みコメント・削除済みリポストを除外）、閲覧者あり／なし、空入力 |
| 同 `TestPostRepository_TimelineQueryCountDoesNotGrowWithPosts` | 投稿 1 件と 5 件でタイムライン取得（`GetExplore` + `FindStatsByIDs`）の SQL 本数が同じ（= N+1 でない）こと、投稿者が JOIN で読み込まれること |
| 同 `TestFollowRepository_FindStatsByUserIDs` | フォロー集計・閲覧者のフォロー状態・空入力 |
| `service/post_service_test.go` `TestPostService_ToPostResponses_*` | 集計取得が 1 回だけで、個別集計メソッドが呼ばれないこと／空入力でクエリしないこと／エラー伝搬 |
| `querycount/querycount_test.go` | カウンター付きコンテキストのクエリだけを数えること |
| `middleware/query_count_test.go` | `X-DB-Query-Count` ヘッダーの付与（200 / 204） |
| `config/config_test.go` | `ShouldLogSQL` / `ShouldCountQueries` の環境別の判定、`SQL_LOG` の読み込み |

既存テストのうち、個別集計メソッドをモックしていたもの（`expectCounts`、検索・フォロー一覧・ブックマーク一覧）は一括取得のモックに置き換えた。

---

## 5. テスト結果（`make test-backend`）

| 実行 | 結果 | PASS | FAIL | カバレッジ（internal 全体） |
|------|------|-----:|-----:|----------------------------:|
| 改善前（変更前のコード） | 成功（exit 0） | 292 | 0 | 67.6% |
| 改善後 | 成功（exit 0） | 309 | 0 | 68.4% |

（PASS 数は `go test -v` のサブテストを含む `--- PASS` の件数。SKIP は 0 件。改善後の +17 件は今回追加したテスト）

---

## 6. 計測上の注意（再計測する人向け）

- **Windows では `localhost` ではなく `127.0.0.1` を使うこと。** `localhost` を指定すると curl が毎回 IPv6（`::1`）への接続を試みて約 220ms 待たされ、計測値がその待ち時間に埋もれる。最初の計測では `localhost` を使ってしまい、改善前 281.7ms → 改善後 226.5ms（−20%）と小さく見えていた（SQL 本数は同じく 128 → 8）。スクリプトの既定値は `127.0.0.1` に修正済み。
- サーバー処理時間は、ネットワークや Docker のポート転送の影響を受けないアクセスログの `latency` で評価するのが確実。
- テスト用 DB のスキーマは `AutoMigrate`（GORM モデルのタグ）で作られるため、本番のマイグレーション（`backend/migrations/`）と一部インデックスの定義が異なる（例: `posts.reply_to` / `repost_of` は部分インデックスではなく通常インデックス、`likes(user_id, post_id)` の UNIQUE 制約がない）。本番相当の絶対値が必要な場合は、マイグレーション適用済みの DB で測ること。
- 結果ファイル（`.perf-results/`）は `.gitignore` 対象。

---

## 7. 残課題

| 項目 | 内容 | 優先度 |
|------|------|--------|
| 固定の Preload 6 本 | リポスト元・返信先とその投稿者・メディアの Preload は件数によらず一定（N+1 ではない）なので残した。さらに減らすなら `ReplyToPost` / `RepostOfPost` を JOIN にする余地がある | Low |
| ブックマーク一覧の投稿者 | `bookmark_repository.GetByUser` は `bookmarks.created_at` を併せて取得する独自 Select のため、`Preload("User")`（1 本・一定）のままにした | Low |
| フォロワー一覧・ユーザー検索の計測 | 単体・統合テストで一括取得になったことは確認済みだが、`perf-timeline` のような実測スクリプトは未作成 | Low |
| 500ms 超の監視 | 本番での遅延監視（500ms 超を WARN で記録）は未実装。`non-functional-requirements-report.md` の TODO-9 で対応 | Medium |
