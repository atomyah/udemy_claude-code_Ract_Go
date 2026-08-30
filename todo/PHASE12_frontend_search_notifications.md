# PHASE 12 — フロントエンド 検索・通知・管理者画面

> 目標: 検索・ハッシュタグ・通知・管理者ページを実装してフロントエンドを完成させる。

---

## 検索

### 検索 API フック（`src/features/search/hooks/`）

- [ ] `useSearchUsers.ts` — `useQuery` で `GET /api/v1/search/users?q=`（入力が 2 文字以上でリクエスト）— 未実装
- [x] `useSearchPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/posts?q=`（`features/posts/hooks/useSearchPosts.ts` に実装、動作確認済み）
- [ ] `useHashtagPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/hashtags/:tag` — 未実装。`#tag` クリック時は専用エンドポイントではなく `useSearchPosts` に `#tag` を渡す形で代替している

### 探索ページ強化（`src/pages/ExplorePage.tsx`）

- [ ] 検索バーに入力するとリアルタイムでユーザー候補をドロップダウン表示 — 未実装
- [x] Enter 押下不要でリアルタイムに投稿検索結果を表示（仕様の「Enter/検索ボタン」とは異なり入力の都度検索する方式）
- [ ] タブ切り替え: 「投稿」「ユーザー」— ユーザー検索タブ自体が存在しない

### ハッシュタグページ（`src/pages/HashtagPage.tsx`）

- [ ] `GET /api/v1/search/hashtags/:tag` で投稿一覧を取得 — 未実装（専用ページ・専用エンドポイント呼び出しなし）
- [x] `#tag` クリックでこのページへ遷移 — 実際は `/explore?q=%23tag` に遷移し `ExplorePage` の投稿検索で代替表示（動作確認済み）

---

## 通知

### 通知 API フック（`src/features/notifications/hooks/`）

- [ ] `useNotifications.ts` — `useInfiniteQuery` で `GET /api/v1/notifications` — 未実装
- [ ] `useMarkAllRead.ts` — `useMutation` で `PUT /api/v1/notifications/read` — 未実装
- [x] `useUnreadCount.ts` — `useUnreadNotificationsCount.ts` として実装済み。`GET /notifications?limit=1` の `unread_count` を 30 秒間隔でポーリング

### 通知ページ（`src/pages/NotificationsPage.tsx`）

- [ ] 通知カードの種別ごとの表示（いいね/コメント/フォロー/リポスト/メンション）— 未実装
- [ ] 未読通知は背景色で区別 — 未実装
- [ ] 「すべて既読にする」ボタン — 未実装
- [ ] 通知クリックで関連投稿 or プロフィールに遷移 — 未実装
- [ ] 無限スクロール — 未実装

> `NotificationsPage.tsx` 自体は5行のプレースホルダー。

### サイドバーの通知バッジ

- [x] `useUnreadCount` で未読数を取得して Sidebar に表示（`Sidebar.tsx` の `Badge` コンポーネントで実装済み）
- [ ] 通知ページを開くと自動で既読にする — 通知ページ自体が未実装のため未対応

---

## 管理者画面

### 管理者ガード（`src/components/AdminRoute.tsx`）

- [x] `currentUser.role !== 'admin'` の場合はアクセス制限（`components/routing/AdminRoute.tsx`。ただし仕様の「403ページ表示」ではなく `/` へリダイレクトする実装）

### 管理者ページ（`src/pages/AdminPage.tsx`）

- [ ] ユーザー管理タブ（ユーザー一覧・停止/停止解除ボタン）— 未実装
- [ ] 投稿管理タブ（投稿 ID 入力で強制削除）— 未実装
- [ ] MUI `DataGrid` または手作りテーブルで表示 — 未実装

> `AdminPage.tsx` 自体は5行のプレースホルダー。対応するバックエンド API（`PHASE07_search_admin.md`）は完成済み。

---

## 共通 UI の仕上げ

- [x] 404 ページ（`src/pages/NotFoundPage.tsx`、簡易なメッセージ表示のみだが実装済み）
- [ ] エラーバウンダリ（`src/components/ErrorBoundary.tsx`）— 未実装
- [ ] トースト通知（MUI `Snackbar`）— 全体で使い回せる仕組みはなく、ログイン/サインアップ画面内でのみ個別に使用
- [ ] ページ読み込み中のスピナー（`src/components/LoadingSpinner.tsx`）— 共通コンポーネントはなく、各所で `CircularProgress` を都度インラインで使用

---

## 完了基準

- [ ] 検索バーで入力するとユーザーサジェストが表示される — 未達成
- [ ] 検索結果ページで投稿とユーザーをタブで切り替えられる — 未達成
- [ ] `#tag` をクリックするとハッシュタグページに遷移して関連投稿が表示される — 実質達成（`ExplorePage` の検索結果として表示、専用ページではない）
- [ ] 通知ページに通知が表示される — 未達成
- [x] サイドバーに未読通知バッジが表示される
- [x] 管理者ユーザーのみ `/admin` にアクセスできる（リダイレクト方式）
- [x] 存在しないページで 404 が表示される

## 備考

このフェーズは「投稿検索」「通知バッジ」「管理者ルートガード」「404ページ」のみ実装済みで、通知一覧・ユーザー検索・ハッシュタグ専用ページ・管理者画面本体・共通UI（エラーバウンダリ/トースト/ローディング共通化）は未着手。対応するバックエンド API は `PHASE06`〜`PHASE07` で完成済みのため、フロントエンド実装のみで着手できる。
