# PHASE 12 — フロントエンド 検索・通知・管理者画面

> 目標: 検索・ハッシュタグ・通知・管理者ページを実装してフロントエンドを完成させる。

---

## 検索

### 検索 API フック

- [x] `useSearchUsers.ts` — `useInfiniteQuery` で `GET /api/v1/search/users?q=`（2 文字以上で有効化）。`features/posts/hooks/useSearchUsers.ts` に実装
- [x] `useSearchPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/posts?q=`（`features/posts/hooks/useSearchPosts.ts`）
- [x] `useHashtagPosts.ts` — `useInfiniteQuery` で `GET /api/v1/search/hashtags/:tag`（`features/posts/hooks/useHashtagPosts.ts`）。404 はリトライせず空表示として扱う

> フックは仕様の `features/search/hooks/` ではなく `features/posts/hooks/` に集約している（投稿一覧系フックと同じ型・同じページネーションを共有するため）。

### 探索ページ強化（`src/pages/ExplorePage.tsx`）

- [x] 検索バーに入力するとリアルタイムでユーザー候補をドロップダウン表示（2 文字以上・最大 5 件、`Popper` + `ClickAwayListener`。候補クリックでプロフィールへ遷移、Escape で閉じる）
- [x] Enter 押下不要でリアルタイムに投稿検索結果を表示（仕様の「Enter/検索ボタン」とは異なり入力の都度検索する方式）
- [x] タブ切り替え: 「投稿」「ユーザー」

### ハッシュタグページ（`src/pages/HashtagPage.tsx`）

- [x] `GET /api/v1/search/hashtags/:tag` で投稿一覧を取得（無限スクロール対応）
- [x] `#tag` クリックでこのページへ遷移（`LinkifiedText.tsx` のリンク先を `/hashtag/:tag` に変更）
- [x] ルートは `/:handle` より前に定義してハンドルとして解釈されるのを防ぐ

---

## 通知

### 通知 API フック（`src/features/notifications/hooks/`）

- [x] `useNotifications.ts` — `useInfiniteQuery` で `GET /api/v1/notifications`
- [x] `useMarkAllRead.ts` — `useMutation` で `PUT /api/v1/notifications/read`
- [x] `useUnreadCount.ts` — `useUnreadNotificationsCount.ts` として実装。`GET /notifications?limit=1` の `unread_count` を 30 秒間隔でポーリング

### 通知ページ（`src/pages/NotificationsPage.tsx`）

- [x] 通知カードの種別ごとの表示（いいね/コメント/フォロー/リポスト/メンション）
- [x] 未読通知は背景色で区別（`bgcolor: 'action.hover'`）
- [x] 「すべて既読にする」ボタン
- [x] 通知クリックで関連投稿 or プロフィールに遷移
- [x] 無限スクロール

> フロントエンドは 5 種別すべてを表示できるが、**バックエンドが通知を作成するのは `like` / `comment` / `repost` の 3 種別のみ**。`follow` と `mention` の通知は未実装（`internal/service/user_service.go` の `Follow` が `notificationSvc.Notify` を呼んでいない）。PHASE12 の範囲外のため別途対応が必要。

### サイドバーの通知バッジ

- [x] `useUnreadCount` で未読数を取得して Sidebar に表示（`Sidebar.tsx` の `Badge`）
- [x] 通知ページを開くと自動で既読にする

---

## 管理者画面

### 管理者ガード（`src/components/routing/AdminRoute.tsx`）

- [x] `currentUser.role !== 'admin'` の場合はアクセス制限（仕様の「403 ページ表示」ではなく `/` へリダイレクトする実装）

### 管理者ページ（`src/pages/AdminPage.tsx`）

- [x] ユーザー管理タブ（ユーザー一覧・停止/停止解除ボタン・無限スクロール、結果はトーストで通知）
- [x] 投稿管理タブ（投稿 ID 入力で強制削除、結果はトーストで通知）
- [x] 手作りテーブル（`Stack` ベースのリスト）で表示。`DataGrid` は使用していない

---

## 共通 UI の仕上げ

- [x] 404 ページ（`src/pages/NotFoundPage.tsx`）
- [x] エラーバウンダリ（`src/components/ErrorBoundary.tsx`、`main.tsx` でアプリ全体をラップ）
- [x] トースト通知（`src/components/ToastProvider.tsx`。`useToast().showToast(message, severity)` で呼び出す）
  - 適用箇所: 投稿削除（成功/失敗）、フォロー・フォロー解除の失敗、メールアドレス変更、パスワード変更、管理者のユーザー停止/解除・投稿強制削除
  - ログイン/新規登録の認証エラーはフォーム内の `Alert` のまま（入力欄の近くで理由を示すため）
- [x] ページ読み込み中のスピナー（`src/components/LoadingSpinner.tsx`）

---

## 完了基準

- [x] 検索バーで入力するとユーザーサジェストが表示される
- [x] 検索結果ページで投稿とユーザーをタブで切り替えられる
- [x] `#tag` をクリックするとハッシュタグページに遷移して関連投稿が表示される
- [x] 通知ページに通知が表示される
- [x] サイドバーに未読通知バッジが表示される
- [x] 管理者ユーザーのみ `/admin` にアクセスできる（リダイレクト方式）
- [x] 存在しないページで 404 が表示される

---

## テスト

- [x] `frontend/e2e/search.spec.ts` — ハッシュタグ遷移 / 未使用タグの空表示 / ユーザーサジェスト / 投稿・ユーザータブ切り替え
- [x] `frontend/e2e/notifications.spec.ts` — いいね通知と未読バッジ解消 / コメント通知の種別表示 / 通知が無いときの表示
- [x] `frontend/e2e/posts.spec.ts` — 投稿削除時のトースト表示を追記

`make test-e2e` で 20 件すべて成功することを確認済み。

---

## 残課題（PHASE12 の範囲外）

- `follow` / `mention` の通知をバックエンドで作成していない（フロントの表示側は対応済み）
- プロフィールページの投稿一覧（`['users', handle, 'posts']`）が `features/posts/cacheHelpers.ts` の `isPostListQuery` の対象外のため、プロフィール上でいいね・リポスト・ブックマーク・削除をしても即時に画面へ反映されない
