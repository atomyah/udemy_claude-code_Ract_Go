# PHASE 08 — フロントエンド基盤・API 型生成・テーマ・レイアウト

> 目標: フロントエンドの骨格（ルーティング、テーマ、API クライアント、型生成）を整える。

---

## OpenAPI 型生成パイプライン

- [x] `frontend/package.json` に型生成スクリプトを追加
  ```json
  "scripts": {
    "gen:api": "openapi-typescript ../backend/docs/openapi.yaml -o src/api/generated/schema.ts"
  }
  ```
  （swaggo は Swagger 2.0 を出力するため、openapi-typescript が要求する OpenAPI 3.x 形式の
  `backend/docs/openapi.yaml`（`swagger2openapi` 変換後）を入力元とした）
- [x] `npm run gen:api` を実行して `src/api/generated/schema.ts` を生成
- [x] 生成ファイルを `.gitignore` に追加する方針に決定（`frontend/src/api/generated/` は既に `.gitignore` 済み。`src/api/types.ts` でクリーンな型エイリアスを再エクスポート）

## Axios クライアント（`src/api/client.ts`）

- [x] Axios インスタンスを作成（`baseURL: import.meta.env.VITE_API_BASE_URL`）
- [x] リクエストインターセプター: `localStorage` から Access Token を取得してヘッダーに付与
- [x] レスポンスインターセプター: 401 を受け取ったら Refresh Token で再取得 → 失敗したらログアウト
- [x] 生成型を使った型付きラッパー関数を `src/api/` に整備（手書き axios 直呼び禁止）

## MUI テーマ（`src/theme/`）

- [x] `lightTheme.ts` — ライトテーマ定義
  - primary: カスタムカラー（デフォルトの青から変更）
  - secondary: アクセントカラー
  - フォント: `"Noto Sans JP", "Roboto", sans-serif`
- [x] `darkTheme.ts` — ダークテーマ定義
- [x] `ThemeContext.tsx` — テーマ状態管理コンテキスト
  - 現在のテーマ名（`'light' | 'dark'`）を保持
  - 切り替え関数 `toggleTheme()`
  - 初期値は `localStorage` から復元（未ログイン）/ API から復元（ログイン済み）
- [x] `src/App.tsx` で `ThemeProvider` と `CssBaseline` をセット

## 認証コンテキスト（`src/features/auth/AuthContext.tsx`）

- [x] `currentUser`（User オブジェクトまたは null）
- [x] `isAuthenticated`（boolean）
- [x] `isLoading`（初期チェック中フラグ）
- [x] `login(email, password) => Promise<void>`
- [x] `logout() => Promise<void>`
- [x] アプリ起動時に Access Token の有効性を検証して currentUser を復元

## React Router（`src/App.tsx`）

- [x] パブリックルート（未ログインでもアクセス可）
  - `/login`, `/signup`, `/:handle`（プロフィール閲覧）, `/posts/:id`
- [x] プライベートルート（未ログイン時は `/login` にリダイレクト）
  - `/`, `/explore`, `/notifications`, `/bookmarks`, `/settings`
- [x] 管理者ルート（admin ロール以外はリダイレクト）
  - `/admin`（JWT ペイロードの `role` クレームで判定。バックエンドの `UserResponse` DTO に
    `role` フィールドが無いため、フロントは Access Token をデコードして判定している。
    実際の認可はバックエンドの admin ミドルウェアが担保）
- [x] `PrivateRoute` コンポーネントで認証チェック（`AdminRoute` も追加）

## レイアウト（`src/components/layout/`）

- [x] `AppLayout.tsx` — 全体レイアウト（サイドバー + メインコンテンツ + ヘッダー + ボトムナビ）
- [x] `Sidebar.tsx` — 左サイドバー（PC: 常時表示、スマホ: ボトムナビゲーション）
  - ナビリンク: ホーム・探索・通知・ブックマーク・プロフィール・設定
  - 未読通知バッジ
  - 投稿ボタン（FAB、投稿機能は PHASE10 で実装するため現時点では disabled）
- [x] `BottomNav.tsx` — スマホ用ボトムナビゲーション（MUI の `BottomNavigation`）
- [x] `Header.tsx` — ページタイトル・テーマ切り替えボタン

## TanStack Query セットアップ

- [x] `src/main.tsx` に `QueryClientProvider` を追加
- [x] `QueryClient` の設定（staleTime、retry 回数）
- [x] QueryKey の型定義・一覧を `src/api/queryKeys.ts` に集約

## 完了基準

- [x] `npm run gen:api` で型ファイルが生成される
- [x] ライト / ダークテーマが切り替わる
- [x] 未ログイン時に `/` にアクセスすると `/login` にリダイレクトされる
- [x] ログイン済み時にサイドバーが表示される
- [x] PC とスマホで適切なナビゲーション（サイドバー / ボトムナビ）が表示される

## 付随して対応したバックエンドの不具合

PHASE08 のフロントエンドを実データで検証する過程で、以下のバックエンド側の問題が発覚し対応した。

- [x] `backend/cmd/server/main.go`: CORS 設定に `AllowCredentials: true` が抜けており、
  `withCredentials: true` を使うフロントエンドからの全リクエストが CORS エラーで失敗していたため追加。
- [x] `backend/internal/handler/user_handler.go`: `GetMe`（`GET /users/me`）が未実装スタブ（501）のままで
  セッション復元ができなかったため、`user_repository.FindByID` と `service.ToUserResponse`
  （`auth_service.go` の `toUserResponse` を export）を使い最小実装した。
  - `UpdateProfile` / `GetProfile` / `GetUserPosts` / `GetFollowers` / `GetFollowing` /
    `Follow` / `Unfollow` / `UpdateTheme` は引き続き未実装（PHASE04 本来のスコープのため今回は対応せず）。
    フロントエンドはこれらが 501 を返しても壊れないよう設計している
    （例: 未読通知バッジはエラー時 0 表示、テーマのサーバー同期は fire-and-forget）。
