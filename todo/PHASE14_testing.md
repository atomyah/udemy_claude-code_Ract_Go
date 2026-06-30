# PHASE 14 — テスト実装

> 目標: バックエンドのユニット/統合テストとフロントエンドのコンポーネント/フックテストを整備し、CI でテストが自動実行される状態を作る。
>
> タイミング: 各 PHASE の実装完了後に随時追加していくが、全機能実装後にこの PHASE で網羅的に補う。

---

## バックエンド テスト環境構築

- [ ] `go get github.com/stretchr/testify`（assert / mock）
- [ ] `go get github.com/stretchr/testify/mock`
- [ ] テスト用 DB の設定
  - `docker-compose.test.yml` を作成（テスト専用 PostgreSQL コンテナ）
  - または `DATABASE_URL` 環境変数をテスト時に切り替え
- [ ] `backend/internal/testutil/` ディレクトリを作成
  - `testutil/db.go` — テスト用 DB 接続・マイグレーション実行ヘルパー
  - `testutil/fixture.go` — テストデータ（ユーザー・投稿）を簡単に作れるヘルパー

---

## バックエンド ユニットテスト（サービス層）

### 認証サービス（`internal/service/auth_service_test.go`）

- [ ] `Register_Success` — 正常にユーザーが作成される
- [ ] `Register_DuplicateEmail` — メール重複で 409
- [ ] `Register_DuplicateHandle` — handle 重複で 409
- [ ] `Login_Success` — 正常に JWT が返る
- [ ] `Login_WrongPassword` — パスワード不一致で 401
- [ ] `Login_UserNotFound` — 存在しないメールで 401
- [ ] `RefreshAccessToken_Success` — 有効な Refresh Token で Access Token が再発行される
- [ ] `RefreshAccessToken_Expired` — 期限切れで 401

### 投稿サービス（`internal/service/post_service_test.go`）

- [ ] `CreatePost_Success` — テキスト・ハッシュタグが正しく保存される
- [ ] `CreatePost_ContentTooLong` — 281 文字でバリデーションエラー
- [ ] `UpdatePost_Success` — 本人が編集できる
- [ ] `UpdatePost_Forbidden` — 他人の投稿を編集しようとすると 403
- [ ] `DeletePost_Success` — is_deleted が true になる
- [ ] `DeletePost_Forbidden` — 他人の投稿を削除しようとすると 403

### いいねサービス（`internal/service/like_service_test.go`）

- [ ] `Like_Success` — いいねが作成される
- [ ] `Like_AlreadyLiked` — 重複いいねで 409
- [ ] `Like_SelfPost_NoNotification` — 自分の投稿にいいねしても通知が作られない
- [ ] `Unlike_Success` — いいねが削除される

### フォローサービス（`internal/service/follow_service_test.go`）

- [ ] `Follow_Success` — フォローが作成される
- [ ] `Follow_SelfFollow` — 自分をフォローしようとすると 400
- [ ] `Follow_AlreadyFollowing` — 重複フォローで 409
- [ ] `Unfollow_Success` — フォローが削除される

### JWT ミドルウェア（`internal/middleware/auth_test.go`）

- [ ] `ValidToken` — 有効なトークンでミドルウェアを通過
- [ ] `InvalidToken` — 不正なトークンで 401
- [ ] `ExpiredToken` — 期限切れで 401 `EXPIRED_TOKEN`
- [ ] `MissingToken` — ヘッダーなしで 401 `MISSING_TOKEN`

### 管理者ミドルウェア（`internal/middleware/admin_test.go`）

- [ ] `AdminRole` — admin ロールで通過
- [ ] `UserRole` — user ロールで 403

---

## バックエンド 統合テスト（ハンドラー層）

`httptest` + Echo を使い、実際の HTTP レスポンスを検証する。

- [ ] `POST /api/v1/auth/register` — 正常登録で 201
- [ ] `POST /api/v1/auth/login` — 正常ログインで 200 + JWT
- [ ] `POST /api/v1/posts` — 未認証で 401
- [ ] `POST /api/v1/posts` — 認証済みで 201
- [ ] `DELETE /api/v1/posts/:id` — 他人の投稿で 403
- [ ] `POST /api/v1/posts/:id/like` — 正常ないいねで 200
- [ ] `DELETE /api/v1/admin/posts/:id` — user ロールで 403
- [ ] `DELETE /api/v1/admin/posts/:id` — admin ロールで 204

---

## フロントエンド テスト環境構築

- [ ] `npm install -D vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event msw`
- [ ] `vite.config.ts` に Vitest 設定を追加
- [ ] `src/mocks/handlers.ts` — MSW ハンドラーの雛形を作成
- [ ] `src/mocks/server.ts` — MSW サーバーのセットアップ
- [ ] `src/setupTests.ts` — `@testing-library/jest-dom` のインポート
- [ ] `package.json` にテストスクリプトを追加
  ```json
  "test": "vitest",
  "test:coverage": "vitest --coverage"
  ```

---

## フロントエンド コンポーネントテスト

### PostCard（`src/components/PostCard/PostCard.test.tsx`）

- [ ] `投稿の本文・ユーザー名が表示される`
- [ ] `いいね数が表示される`
- [ ] `いいねボタンを押すと色が変わる`
- [ ] `本人の投稿には削除メニューが表示される`
- [ ] `他人の投稿には削除メニューが表示されない`
- [ ] `編集済みラベルが is_edited=true のとき表示される`

### PostForm（`src/features/posts/components/PostForm.test.tsx`）

- [ ] `テキストエリアに入力できる`
- [ ] `文字数カウンターが更新される`
- [ ] `281 文字入力で投稿ボタンが disabled になる`
- [ ] `テキスト未入力・メディアなしで投稿ボタンが disabled になる`
- [ ] `投稿ボタンを押すと API が呼ばれる`

### 認証関連

- [ ] `LoginPage`: `フォーム未入力で送信できない`
- [ ] `LoginPage`: `API エラーでエラーメッセージが表示される`
- [ ] `PrivateRoute`: `未ログイン時に /login にリダイレクトされる`
- [ ] `PrivateRoute`: `ログイン済みはコンテンツが表示される`

### 通知バッジ

- [ ] `未読通知が 0 のときバッジが表示されない`
- [ ] `未読通知があるときバッジに数字が表示される`

---

## フロントエンド フックテスト

### useLike（`src/features/posts/hooks/useLike.test.ts`）

- [ ] `いいねを押すと楽観的更新でカウントが増える`
- [ ] `API 失敗時にロールバックされる`

### useInfiniteTimeline

- [ ] `初回ロードで投稿が表示される`
- [ ] `次ページが存在するとき「さらに読み込む」が動作する`
- [ ] `nextCursor が null のとき終端を検知する`

---

## CI（GitHub Actions）設定

`.github/workflows/test.yml` を作成:

- [ ] PR 作成・更新時にトリガー
- [ ] バックエンドジョブ
  - [ ] `go test ./...` を実行
  - [ ] テスト用 PostgreSQL サービスを起動
  - [ ] カバレッジを 70% 以上に維持
- [ ] フロントエンドジョブ
  - [ ] `npm run test` を実行
  - [ ] `npm run build` でビルドエラーがないことを確認
- [ ] 型チェックジョブ（フロントエンド）
  - [ ] `npx tsc --noEmit` で型エラーがないことを確認

---

## 完了基準

- [ ] `go test ./...` が全て PASS する
- [ ] `npm run test` が全て PASS する
- [ ] バックエンドのサービス層テストカバレッジが 70% 以上
- [ ] PR 時に GitHub Actions で自動テストが走る
- [ ] テスト用 DB とアプリ用 DB が分離されている（本番データを汚染しない）
