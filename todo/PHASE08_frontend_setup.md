# PHASE 08 — フロントエンド基盤・API 型生成・テーマ・レイアウト

> 目標: フロントエンドの骨格（ルーティング、テーマ、API クライアント、型生成）を整える。

---

## OpenAPI 型生成パイプライン

- [ ] `frontend/package.json` に型生成スクリプトを追加
  ```json
  "scripts": {
    "gen:api": "openapi-typescript http://localhost:8080/swagger/doc.json -o src/api/generated/schema.ts"
  }
  ```
- [ ] `npm run gen:api` を実行して `src/api/generated/schema.ts` を生成
- [ ] 生成ファイルを `.gitignore` に追加するか、CI で自動生成するか方針を決める

## Axios クライアント（`src/api/client.ts`）

- [ ] Axios インスタンスを作成（`baseURL: import.meta.env.VITE_API_BASE_URL`）
- [ ] リクエストインターセプター: `localStorage` から Access Token を取得してヘッダーに付与
- [ ] レスポンスインターセプター: 401 を受け取ったら Refresh Token で再取得 → 失敗したらログアウト
- [ ] 生成型を使った型付きラッパー関数を `src/api/` に整備（手書き axios 直呼び禁止）

## MUI テーマ（`src/theme/`）

- [ ] `lightTheme.ts` — ライトテーマ定義
  - primary: カスタムカラー（デフォルトの青から変更）
  - secondary: アクセントカラー
  - フォント: `"Noto Sans JP", "Roboto", sans-serif`
- [ ] `darkTheme.ts` — ダークテーマ定義
- [ ] `ThemeContext.tsx` — テーマ状態管理コンテキスト
  - 現在のテーマ名（`'light' | 'dark'`）を保持
  - 切り替え関数 `toggleTheme()`
  - 初期値は `localStorage` から復元（未ログイン）/ API から復元（ログイン済み）
- [ ] `src/App.tsx` で `ThemeProvider` と `CssBaseline` をセット

## 認証コンテキスト（`src/features/auth/AuthContext.tsx`）

- [ ] `currentUser`（User オブジェクトまたは null）
- [ ] `isAuthenticated`（boolean）
- [ ] `isLoading`（初期チェック中フラグ）
- [ ] `login(email, password) => Promise<void>`
- [ ] `logout() => Promise<void>`
- [ ] アプリ起動時に Access Token の有効性を検証して currentUser を復元

## React Router（`src/App.tsx`）

- [ ] パブリックルート（未ログインでもアクセス可）
  - `/login`, `/signup`, `/:handle`（プロフィール閲覧）, `/posts/:id`
- [ ] プライベートルート（未ログイン時は `/login` にリダイレクト）
  - `/`, `/explore`, `/notifications`, `/bookmarks`, `/settings`
- [ ] 管理者ルート（admin ロール以外は 403）
  - `/admin`
- [ ] `PrivateRoute` コンポーネントで認証チェック

## レイアウト（`src/components/layout/`）

- [ ] `AppLayout.tsx` — 全体レイアウト（サイドバー + メインコンテンツ + 右サイドバー）
- [ ] `Sidebar.tsx` — 左サイドバー（PC: 常時表示、スマホ: ボトムナビゲーション）
  - ナビリンク: ホーム・探索・通知・ブックマーク・プロフィール・設定
  - 未読通知バッジ
  - 投稿ボタン（FAB）
- [ ] `BottomNav.tsx` — スマホ用ボトムナビゲーション（MUI の `BottomNavigation`）
- [ ] `Header.tsx` — ページタイトル・テーマ切り替えボタン

## TanStack Query セットアップ

- [ ] `src/main.tsx` に `QueryClientProvider` を追加
- [ ] `QueryClient` の設定（staleTime、retry 回数）
- [ ] QueryKey の型定義・一覧を `src/api/queryKeys.ts` に集約

## 完了基準

- [ ] `npm run gen:api` で型ファイルが生成される
- [ ] ライト / ダークテーマが切り替わる
- [ ] 未ログイン時に `/` にアクセスすると `/login` にリダイレクトされる
- [ ] ログイン済み時にサイドバーが表示される
- [ ] PC とスマホで適切なナビゲーション（サイドバー / ボトムナビ）が表示される
