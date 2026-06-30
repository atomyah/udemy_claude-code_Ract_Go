# PHASE 12 — フロントエンド 検索・通知・管理者画面

> 目標: 検索・ハッシュタグ・通知・管理者ページを実装してフロントエンドを完成させる。

---

## 検索

### 検索 API フック（`src/features/search/hooks/`）

- [ ] `useSearchUsers.ts` — `useQuery` で `GET /api/v1/search/users?q=`（入力が 2 文字以上でリクエスト）
- [ ] `useSearchPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/posts?q=`
- [ ] `useHashtagPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/hashtags/:tag`

### 探索ページ強化（`src/pages/ExplorePage.tsx`）

- [ ] 検索バーに入力するとリアルタイムでユーザー候補をドロップダウン表示
- [ ] Enter 押下または「検索」ボタンで全件検索結果ページへ
- [ ] タブ切り替え: 「投稿」「ユーザー」

### ハッシュタグページ（`src/pages/HashtagPage.tsx`）

- [ ] `GET /api/v1/search/hashtags/:tag` で投稿一覧を取得
- [ ] `#tag` クリックでこのページへ遷移

---

## 通知

### 通知 API フック（`src/features/notifications/hooks/`）

- [ ] `useNotifications.ts` — `useInfiniteQuery` で `GET /api/v1/notifications`
- [ ] `useMarkAllRead.ts` — `useMutation` で `PUT /api/v1/notifications/read`
- [ ] `useUnreadCount.ts` — 未読通知数を定期取得（ポーリング or 将来的に WebSocket）

### 通知ページ（`src/pages/NotificationsPage.tsx`）

- [ ] 通知カードの種別ごとの表示
  - いいね: `❤ [ユーザー名] さんがあなたの投稿にいいねしました`
  - コメント: `💬 [ユーザー名] さんがコメントしました`
  - フォロー: `👤 [ユーザー名] さんがフォローしました`
  - リポスト: `🔁 [ユーザー名] さんがリポストしました`
  - メンション: `@ [ユーザー名] さんがメンションしました`
- [ ] 未読通知は背景色で区別
- [ ] 「すべて既読にする」ボタン
- [ ] 通知クリックで関連投稿 or プロフィールに遷移
- [ ] 無限スクロール

### サイドバーの通知バッジ

- [ ] `useUnreadCount` で未読数を取得して Sidebar に表示
- [ ] 通知ページを開くと自動で既読にする

---

## 管理者画面

### 管理者ガード（`src/components/AdminRoute.tsx`）

- [ ] `currentUser.role !== 'admin'` の場合は 403 ページを表示

### 管理者ページ（`src/pages/AdminPage.tsx`）

- [ ] ユーザー管理タブ
  - ユーザー一覧（ID・handle・email・role・is_suspended）
  - 「停止」ボタン（確認ダイアログ付き）
  - 「停止解除」ボタン
- [ ] 投稿管理タブ
  - 投稿 ID を入力して強制削除
- [ ] MUI `DataGrid` または手作りテーブルで表示

---

## 共通 UI の仕上げ

- [ ] 404 ページ（`src/pages/NotFoundPage.tsx`）
- [ ] エラーバウンダリ（`src/components/ErrorBoundary.tsx`）
- [ ] トースト通知（MUI `Snackbar`）— API エラー・成功メッセージを全体で使える仕組み
- [ ] ページ読み込み中のスピナー（`src/components/LoadingSpinner.tsx`）

---

## 完了基準

- [ ] 検索バーで入力するとユーザーサジェストが表示される
- [ ] 検索結果ページで投稿とユーザーをタブで切り替えられる
- [ ] `#tag` をクリックするとハッシュタグページに遷移して関連投稿が表示される
- [ ] 通知ページに通知が表示される
- [ ] サイドバーに未読通知バッジが表示される
- [ ] 管理者ユーザーのみ `/admin` にアクセスできる
- [ ] 存在しないページで 404 が表示される
